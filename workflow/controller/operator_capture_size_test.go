package controller

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"
	clocktesting "k8s.io/utils/clock/testing"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// A small configured limit keeps this an inexpensive real compression test.
// Incompressible result data must exceed the limit after compression, while
// the previously persisted Workflow and its error disposition still fit.
func oversizedCaptureMessage(t *testing.T) string {
	t.Helper()
	data := make([]byte, 16*1024)
	_, err := rand.New(rand.NewSource(1)).Read(data)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(data)
}

func TestCapturedPodUnsupportedSizePublishesErrorWithoutCapture(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "false")
	t.Setenv("MAX_WORKFLOW_SIZE", "2048")
	for _, phase := range []wfv1.WorkflowPhase{wfv1.WorkflowSucceeded, wfv1.WorkflowFailed, wfv1.WorkflowError} {
		for _, write := range []string{"accepted", "rejected", "conflict", "lost-response"} {
			t.Run(string(phase)+"/"+write, func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, pod := captureWriterFixture()
				wf.Labels = map[string]string{common.LabelKeyCompleted: "false"}
				wf.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "capture-size"}}}
				pod.Finalizers = []string{common.FinalizerPodStatus, "example.com/keep"}
				cancel, controller := newController(ctx, wf, func(c *WorkflowController) { c.Config.Parallelism = 1 })
				defer cancel()
				controller.hydrator = hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
				acquired, _, _, _, err := controller.syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
				require.NoError(t, err)
				require.True(t, acquired)
				wf.ResourceVersion = "2"
				client := controller.wfclientset.(*fakewfclientset.Clientset)
				_, err = client.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
				require.NoError(t, err)
				held, err := util.ToUnstructured(wf)
				require.NoError(t, err)
				_, err = controller.dynamicInterface.Resource(wfv1.SchemeGroupVersion.WithResource("workflows")).Namespace(wf.Namespace).Update(ctx, held, metav1.UpdateOptions{})
				require.NoError(t, err)
				require.Eventually(t, func() bool {
					cached, found, cacheErr := controller.wfInformer.GetIndexer().GetByKey(wf.Namespace + "/" + wf.Name)
					return cacheErr == nil && found && cached.(metav1.Object).GetResourceVersion() == "2"
				}, time.Second, time.Millisecond, "the completion filter must observe the last persisted lock ownership")
				large, err := packer.IsLargeWorkflow(wf)
				require.NoError(t, err)
				require.False(t, large, "the last persisted Workflow must fit")

				key := wf.Namespace + "/" + wf.Name
				require.True(t, controller.throttler.Admit(key))
				contender := wf.DeepCopy()
				contender.Name, contender.UID = "contender", "contender-uid"
				contender.Status = wfv1.WorkflowStatus{}
				contenderKey := contender.Namespace + "/" + contender.Name
				controller.throttler.Add(contenderKey, 0, time.Now())
				require.False(t, controller.throttler.Admit(contenderKey))
				acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
				require.NoError(t, err)
				require.False(t, acquired)

				client.ClearActions()
				client.PrependReactor("update", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
					candidate := action.(k8stesting.UpdateAction).GetObject().(*wfv1.Workflow)
					assert.Equal(t, wfv1.WorkflowError, candidate.Status.Phase)
					assert.Equal(t, "true", candidate.Labels[common.LabelKeyCompleted])
					assert.Equal(t, wf.Status.Nodes, candidate.Status.Nodes, "a storage failure must not publish unsaved capture receipts")
					assert.False(t, controller.throttler.Admit(contenderKey), "capacity remains held before confirmation")
					candidate.ResourceVersion = "3"
					switch write {
					case "rejected":
						return true, nil, apierr.NewForbidden(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, wf.Name, fmt.Errorf("write denied"))
					case "conflict":
						return true, nil, apierr.NewConflict(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, wf.Name, fmt.Errorf("resource version changed"))
					case "lost-response":
						require.NoError(t, client.Tracker().Update(wfv1.SchemeGroupVersion.WithResource("workflows"), candidate.DeepCopy(), wf.Namespace))
						return true, nil, apierr.NewTimeoutError("response lost after commit", 1)
					default:
						return false, nil, nil
					}
				})
				woc := newWorkflowOperationCtx(ctx, wf, controller)
				old := wf.Status.Nodes[wf.Name]
				selected := woc.assessNodeStatus(ctx, pod, &old)
				require.NotNil(t, selected)
				selected.Phase = wfv1.NodePhase(phase)
				selected.Message = oversizedCaptureMessage(t)
				woc.wf.Status.Nodes[wf.Name] = *selected
				woc.wf.Status.Phase = phase
				woc.wf.Labels[common.LabelKeyCompleted] = "true"
				woc.updated = true
				require.NotPanics(t, func() { woc.persistUpdates(ctx) })
				assert.Equal(t, write != "accepted", woc.reapplyFailed)
				var updates int
				for _, action := range client.Actions() {
					if action.GetVerb() == "update" {
						updates++
					}
				}
				assert.Equal(t, 1, updates, "fallback attempts only the small Error snapshot")
				persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
				require.NoError(t, err)
				committed := write == "accepted" || write == "lost-response"
				if committed {
					assert.Equal(t, wfv1.WorkflowError, persisted.Status.Phase)
					assert.Contains(t, persisted.Status.Message, "maximum allowed size")
					assert.Contains(t, persisted.Status.Message, sqldb.ErrOffloadNotSupported.Error())
					large, err = packer.IsLargeWorkflow(persisted)
					require.NoError(t, err)
					assert.False(t, large)
				} else {
					assert.Equal(t, wf.Status, persisted.Status)
				}
				assert.Equal(t, wf.Status.Nodes, persisted.Status.Nodes)
				assert.Empty(t, persisted.Status.Nodes[wf.Name].CapturedPodUID)
				assert.False(t, controller.throttler.Admit(contenderKey))
				acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
				require.NoError(t, err)
				assert.False(t, acquired, "no lock release before the persisted completion is observed")

				if committed {
					// The typed and dynamic fake clients have separate stores. Forward
					// only the authoritative committed snapshot to the real informer.
					observed, err := util.ToUnstructured(persisted)
					require.NoError(t, err)
					_, err = controller.dynamicInterface.Resource(wfv1.SchemeGroupVersion.WithResource("workflows")).Namespace(wf.Namespace).Update(ctx, observed, metav1.UpdateOptions{})
					require.NoError(t, err)
					require.Eventually(t, func() bool { return controller.throttler.Admit(contenderKey) }, time.Second, time.Millisecond, "confirmed completion releases controller parallelism")
					acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
					require.NoError(t, err)
					assert.True(t, acquired, "confirmed completion releases the shared mutex")
				}

				podClient := fake.NewClientset(pod)
				captureBridgeStartup(ctx, t, controller, podClient, false)
				captureBridgePod(ctx, t, podClient, pod, false)
				for _, action := range podClient.Actions() {
					assert.NotEqual(t, "patch", action.GetVerb())
					assert.NotEqual(t, "delete", action.GetVerb())
				}
			})
		}
	}
}

func TestCapturedPodUnsupportedSizeCompletesDespiteOlderPartialStatus(t *testing.T) {
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "false")
	t.Setenv("MAX_WORKFLOW_SIZE", "2048")
	for _, partial := range []string{"daemon", "task-result"} {
		t.Run(partial, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureWriterFixture()
			old := wf.Status.Nodes[wf.Name]
			if partial == "daemon" {
				old.Daemoned = new(true)
			} else {
				old.TaskResultSynced = new(false)
				wf.Status.TaskResultsCompletionStatus = map[string]bool{wf.Name: false}
			}
			wf.Status.Nodes[wf.Name] = old
			cancel, controller := newController(ctx, wf)
			defer cancel()
			controller.hydrator = hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			selected := old.DeepCopy()
			selected.Phase = wfv1.NodeSucceeded
			selected.Daemoned = nil
			selected.TaskResultSynced = new(true)
			selected.CapturedPodUID = string(pod.UID)
			selected.Message = oversizedCaptureMessage(t)
			woc.wf.Status.Nodes[wf.Name] = *selected
			woc.wf.Status.TaskResultsCompletionStatus = map[string]bool{wf.Name: true}
			woc.wf.Status.Phase = wfv1.WorkflowSucceeded
			woc.wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			woc.updated = true
			require.NotPanics(t, func() { woc.persistUpdates(ctx) })
			require.False(t, woc.reapplyFailed)
			persisted, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, wfv1.WorkflowError, persisted.Status.Phase)
			assert.Equal(t, "true", persisted.Labels[common.LabelKeyCompleted])
			assert.Equal(t, wf.Status.Nodes, persisted.Status.Nodes)
			assert.Equal(t, wf.Status.TaskResultsCompletionStatus, persisted.Status.TaskResultsCompletionStatus)
			assert.Empty(t, persisted.Status.Nodes[wf.Name].CapturedPodUID)
		})
	}
}

func TestCapturedPodUnsupportedSizeWorkerRetainsCapacityOnFailedUpdate(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "false")
	t.Setenv("MAX_WORKFLOW_SIZE", "2048")
	for _, rejected := range []bool{false, true} {
		t.Run(fmt.Sprintf("rejected=%t", rejected), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureWriterFixture()
			pod.Status.Phase = apiv1.PodFailed
			pod.Status.Message = oversizedCaptureMessage(t)
			pod.Finalizers = []string{common.FinalizerPodStatus, "example.com/keep"}
			cancel, controller := newController(ctx, wf, func(c *WorkflowController) { c.Config.Parallelism = 1 })
			defer cancel()
			controller.hydrator = hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
			_, err := controller.kubeclientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				_, found, cacheErr := controller.PodController.TestingPodInformer().GetIndexer().GetByKey(pod.Namespace + "/" + pod.Name)
				return cacheErr == nil && found
			}, time.Second, time.Millisecond)
			contenderKey := wf.Namespace + "/contender"
			controller.throttler.Add(contenderKey, 0, time.Now())
			require.False(t, controller.throttler.Admit(contenderKey))
			client := controller.wfclientset.(*fakewfclientset.Clientset)
			client.ClearActions()
			if rejected {
				client.PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierr.NewForbidden(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, wf.Name, fmt.Errorf("write denied"))
				})
			}
			require.Eventually(t, func() bool { return controller.wfQueue.Len() > 0 }, time.Second, time.Millisecond)
			require.True(t, controller.processNextItem(ctx))
			persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			if rejected {
				assert.Equal(t, wfv1.WorkflowRunning, persisted.Status.Phase)
			} else {
				assert.Equal(t, wfv1.WorkflowError, persisted.Status.Phase)
				assert.Contains(t, persisted.Status.Message, "maximum allowed size")
			}
			assert.Equal(t, !rejected, controller.throttler.Admit(contenderKey), "the worker defer must release capacity only after confirmed Error persistence")
			assert.Empty(t, persisted.Status.Nodes[wf.Name].CapturedPodUID)
			captureBridgePod(ctx, t, controller.kubeclientset.(*fake.Clientset), pod, false)
		})
	}
}

func TestCapturedPodUnsupportedSizeDoesNotRewritePersistedOutcome(t *testing.T) {
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "false")
	t.Setenv("MAX_WORKFLOW_SIZE", "2048")
	for _, phase := range []wfv1.WorkflowPhase{wfv1.WorkflowSucceeded, wfv1.WorkflowFailed, wfv1.WorkflowError} {
		t.Run(string(phase), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, _ := captureWriterFixture()
			wf.Status.Phase = phase
			wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			cancel, controller := newController(ctx, wf)
			defer cancel()
			controller.hydrator = hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			node := woc.wf.Status.Nodes[wf.Name]
			node.Message = oversizedCaptureMessage(t)
			woc.wf.Status.Nodes[wf.Name] = node
			woc.updated = true
			client := controller.wfclientset.(*fakewfclientset.Clientset)
			client.ClearActions()
			require.NotPanics(t, func() { woc.persistUpdates(ctx) })
			assert.True(t, woc.reapplyFailed)
			for _, action := range client.Actions() {
				assert.NotEqual(t, "update", action.GetVerb(), "an already persisted terminal outcome is immutable")
			}
			persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, wf.Status, persisted.Status)
		})
	}
}

func TestCapturedPodUnsupportedSizeReleasesNewLocksAfterObservedCompletion(t *testing.T) {
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "false")
	t.Setenv("MAX_WORKFLOW_SIZE", "2048")
	for _, artifactGC := range []bool{false, true} {
		for _, lostResponse := range []bool{false, true} {
			t.Run(fmt.Sprintf("artifactGC=%t/lostResponse=%t", artifactGC, lostResponse), func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, pod := captureWriterFixture()
				wf.Labels = map[string]string{common.LabelKeyCompleted: "false"}
				wf.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "new-capture-size"}}}
				if artifactGC {
					wf.Finalizers = []string{common.FinalizerArtifactGC}
				}
				cancel, controller := newController(ctx, wf, func(c *WorkflowController) { c.Config.Parallelism = 1 })
				defer cancel()
				controller.hydrator = hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
				woc := newWorkflowOperationCtx(ctx, wf, controller)
				// This operation acquires a mutex absent from the previous API
				// snapshot, then discovers that the completed result cannot fit.
				acquired, _, _, _, err := controller.syncManager.TryAcquire(ctx, woc.wf, "", wf.Spec.Synchronization)
				require.NoError(t, err)
				require.True(t, acquired)
				require.Nil(t, woc.orig.Status.Synchronization)
				contender := wf.DeepCopy()
				contender.Name, contender.UID = "contender", "contender-uid"
				contender.Status = wfv1.WorkflowStatus{}
				contenderKey := contender.Namespace + "/" + contender.Name
				controller.throttler.Add(contenderKey, 0, time.Now())
				require.False(t, controller.throttler.Admit(contenderKey))
				acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
				require.NoError(t, err)
				require.False(t, acquired)
				client := controller.wfclientset.(*fakewfclientset.Clientset)
				client.PrependReactor("update", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
					candidate := action.(k8stesting.UpdateAction).GetObject().(*wfv1.Workflow)
					assert.NotNil(t, candidate.Status.Synchronization, "persist lock ownership even when it was acquired in this operation")
					assert.Equal(t, wf.Status.Nodes, candidate.Status.Nodes)
					candidate.ResourceVersion = "2"
					if lostResponse {
						require.NoError(t, client.Tracker().Update(wfv1.SchemeGroupVersion.WithResource("workflows"), candidate.DeepCopy(), wf.Namespace))
						return true, nil, apierr.NewTimeoutError("response lost after commit", 1)
					}
					return false, nil, nil
				})
				selected := wf.Status.Nodes[wf.Name]
				selected.Phase = wfv1.NodeSucceeded
				selected.CapturedPodUID = string(pod.UID)
				selected.Message = oversizedCaptureMessage(t)
				woc.wf.Status.Nodes[wf.Name] = selected
				woc.wf.Status.Phase = wfv1.WorkflowSucceeded
				woc.wf.Labels[common.LabelKeyCompleted] = "true"
				woc.updated = true
				woc.persistUpdates(ctx)
				persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
				require.NoError(t, err)
				require.Equal(t, wfv1.WorkflowError, persisted.Status.Phase)
				assert.Equal(t, wf.Status.Nodes, persisted.Status.Nodes)
				observed, err := util.ToUnstructured(persisted)
				require.NoError(t, err)
				_, err = controller.dynamicInterface.Resource(wfv1.SchemeGroupVersion.WithResource("workflows")).Namespace(wf.Namespace).Update(ctx, observed, metav1.UpdateOptions{})
				require.NoError(t, err)
				require.Eventually(t, func() bool {
					acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
					return err == nil && acquired
				}, time.Second, time.Millisecond, "confirmed completion releases even a newly acquired mutex while artifact GC remains")
				if artifactGC {
					assert.False(t, controller.throttler.Admit(contenderKey), "normal capacity accounting still includes unfinished artifact GC")
				} else {
					require.Eventually(t, func() bool { return controller.throttler.Admit(contenderKey) }, time.Second, time.Millisecond)
				}
			})
		}
	}
}

func TestCapturedPodUnsupportedSizeRejectedUpdateRetriesWithoutEvent(t *testing.T) {
	t.Setenv("ALWAYS_OFFLOAD_NODE_STATUS", "false")
	t.Setenv("MAX_WORKFLOW_SIZE", "2048")
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprintf("conflict=%t", conflict), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			// Keep the dynamic informer empty: only the typed fake contains the
			// Workflow, so no informer callback can replace the explicit retry.
			cancel, controller := newController(ctx)
			defer cancel()
			wf, _ := captureWriterFixture()
			client := controller.wfclientset.(*fakewfclientset.Clientset)
			_, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
			require.NoError(t, err)
			clock := clocktesting.NewFakeClock(time.Unix(1000, 0))
			controller.wfQueue.ShutDown()
			controller.wfQueue = workqueue.NewTypedRateLimitingQueueWithConfig(
				workqueue.DefaultTypedControllerRateLimiter[string](),
				workqueue.TypedRateLimitingQueueConfig[string]{Clock: clock},
			)
			t.Cleanup(controller.wfQueue.ShutDown)
			controller.hydrator = hydrator.New(sqldb.ExplosiveOffloadNodeStatusRepo)
			client.PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
				resource := schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}
				if conflict {
					return true, nil, apierr.NewConflict(resource, wf.Name, fmt.Errorf("resource version changed"))
				}
				return true, nil, apierr.NewForbidden(resource, wf.Name, fmt.Errorf("write denied"))
			})
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			node := woc.wf.Status.Nodes[wf.Name]
			node.Phase = wfv1.NodeSucceeded
			node.Message = oversizedCaptureMessage(t)
			woc.wf.Status.Nodes[wf.Name] = node
			woc.wf.Status.Phase = wfv1.WorkflowSucceeded
			woc.updated = true
			woc.persistUpdates(ctx)
			require.True(t, woc.reapplyFailed)
			require.Eventually(t, func() bool { return clock.Waiters() >= 2 }, time.Second, time.Millisecond)
			clock.Step(30*time.Second - time.Nanosecond)
			assert.Never(t, func() bool { return controller.wfQueue.Len() != 0 }, 10*time.Millisecond, time.Millisecond)
			clock.Step(time.Nanosecond)
			require.Eventually(t, func() bool { return controller.wfQueue.Len() == 1 }, time.Second, time.Millisecond)
			key, quit := controller.wfQueue.Get()
			require.False(t, quit)
			controller.wfQueue.Done(key)
			assert.Equal(t, wf.Namespace+"/"+wf.Name, key)
		})
	}
}
