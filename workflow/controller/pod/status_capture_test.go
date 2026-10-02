package pod

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func captureFixture() (*wfv1.Workflow, *apiv1.Pod) {
	wf, pod := workflowCleanupFixture()
	pod.Status.Phase = apiv1.PodSucceeded
	node := wf.Status.Nodes["node"]
	node.TaskResultSynced = new(true)
	wf.Status.Nodes["node"] = node
	return wf, pod
}

func TestStatusCaptureGate(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, tc := range []struct {
		name    string
		mutate  func(*wfv1.Workflow, *apiv1.Pod)
		allow   bool
		pending bool
	}{
		{"captured", func(*wfv1.Workflow, *apiv1.Pod) {}, true, false},
		{"completed terminal agent", func(w *wfv1.Workflow, p *apiv1.Pod) {
			p.Labels[common.LabelKeyComponent] = "agent"
			w.Status.Nodes = nil
		}, true, false},
		{"unfinished terminal agent", func(w *wfv1.Workflow, p *apiv1.Pod) {
			p.Labels[common.LabelKeyComponent] = "agent"
			w.Status.Nodes = nil
			w.Status.Phase = wfv1.WorkflowRunning
			w.Labels[common.LabelKeyCompleted] = "false"
		}, false, true},
		{"capture not persisted", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.CapturedPodUID = ""
			w.Status.Nodes["node"] = n
		}, false, true},
		{"replacement capture", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.CapturedPodUID = "old-pod"
			w.Status.Nodes["node"] = n
		}, false, true},
		{"task unsynced", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.TaskResultSynced = new(false)
			w.Status.Nodes["node"] = n
		}, false, true},
		{"legacy task unsynced", func(w *wfv1.Workflow, p *apiv1.Pod) {
			w.Status.TaskResultsCompletionStatus = map[string]bool{"node": false}
		}, false, true},
		{"legacy task missing", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.TaskResultSynced = nil
			w.Status.Nodes["node"] = n
		}, true, false},
		{"legacy nil task unsynced", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.TaskResultSynced = nil
			w.Status.Nodes["node"] = n
			w.Status.TaskResultsCompletionStatus = map[string]bool{"node": false}
		}, false, true},
		{"legacy nil task synced", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.TaskResultSynced = nil
			w.Status.Nodes["node"] = n
			w.Status.TaskResultsCompletionStatus = map[string]bool{"node": true}
		}, true, false},
		{"legacy nil task missing map key", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.TaskResultSynced = nil
			w.Status.Nodes["node"] = n
			w.Status.TaskResultsCompletionStatus = map[string]bool{"another-node": false}
		}, true, false},
		{"unfulfilled phase", func(w *wfv1.Workflow, p *apiv1.Pod) {
			n := w.Status.Nodes["node"]
			n.Phase = wfv1.NodeRunning
			w.Status.Nodes["node"] = n
		}, false, true},
		{"replaced owner", func(w *wfv1.Workflow, p *apiv1.Pod) { w.UID = "new-workflow" }, true, false},
		{"deleting owner", func(w *wfv1.Workflow, p *apiv1.Pod) {
			now := metav1.Now()
			w.DeletionTimestamp = &now
			w.Status.OffloadNodeStatusVersion = "unreadable"
		}, true, false},
		{"node mismatch", func(w *wfv1.Workflow, p *apiv1.Pod) { p.Annotations[common.AnnotationKeyNodeName] = "different" }, false, true},
		{"legacy node name resolves", func(w *wfv1.Workflow, p *apiv1.Pod) {
			delete(p.Annotations, common.AnnotationKeyNodeID)
			n := w.Status.Nodes["node"]
			n.ID = w.ResolveNodeID(n.Name)
			w.Status.Nodes = wfv1.Nodes{n.ID: n}
		}, true, false},
		{"wrong instance", func(w *wfv1.Workflow, p *apiv1.Pod) { w.Labels[common.LabelKeyControllerInstanceID] = "other" }, false, false},
		{"wrong label", func(w *wfv1.Workflow, p *apiv1.Pod) { p.Labels[common.LabelKeyWorkflow] = "other" }, false, false},
		{"unknown owner", func(w *wfv1.Workflow, p *apiv1.Pod) { p.OwnerReferences = nil }, false, true},
		{"nonterminal deleting", func(w *wfv1.Workflow, p *apiv1.Pod) {
			p.Status.Phase = apiv1.PodPending
			now := metav1.Now()
			p.DeletionTimestamp = &now
			w.Status.Nodes = nil
		}, true, false},
		{"failed retirement", func(w *wfv1.Workflow, p *apiv1.Pod) {
			p.Status.Phase = apiv1.PodFailed
			n := w.Status.Nodes["node"]
			n.Phase = wfv1.NodePending
			n.CapturedPodUID = ""
			n.RestartingPodUID = string(p.UID)
			n.TaskResultSynced = new(false)
			w.Status.Nodes["node"] = n
		}, true, false},
		{"wrong retirement", func(w *wfv1.Workflow, p *apiv1.Pod) {
			p.Status.Phase = apiv1.PodFailed
			n := w.Status.Nodes["node"]
			n.Phase = wfv1.NodePending
			n.CapturedPodUID = ""
			n.RestartingPodUID = "old"
			w.Status.Nodes["node"] = n
		}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wf, pod := captureFixture()
			tc.mutate(wf, pod)
			c := identityTestController(t, fake.NewSimpleClientset(pod))
			c.lookupWorkflow = func(_ context.Context, ns, name string, hydrate bool) (*wfv1.Workflow, error) {
				assert.False(t, hydrate)
				return wf.DeepCopy(), nil
			}
			allowed, err := c.allowPodCleanup(t.Context(), pod)
			assert.Equal(t, tc.allow, allowed)
			if tc.pending {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStatusCaptureReadsReferencedStorage(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, storage := range []string{"compressed", "offloaded"} {
		for _, broken := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/broken=%t", storage, broken), func(t *testing.T) {
				wf, pod := captureFixture()
				c := identityTestController(t, fake.NewSimpleClientset(pod))
				var reads []bool
				c.lookupWorkflow = func(_ context.Context, ns, name string, hydrate bool) (*wfv1.Workflow, error) {
					reads = append(reads, hydrate)
					workflowCopy := wf.DeepCopy()
					if !hydrate {
						workflowCopy.Status.Nodes = nil
						if storage == "compressed" {
							workflowCopy.Status.CompressedNodes = "compressed-result"
						} else {
							workflowCopy.Status.OffloadNodeStatusVersion = "referenced-version"
						}
						return workflowCopy, nil
					}
					if broken {
						return nil, fmt.Errorf("referenced node storage unavailable")
					}
					return workflowCopy, nil
				}
				allowed, err := c.allowPodCleanup(t.Context(), pod)
				assert.Equal(t, !broken, allowed)
				assert.Equal(t, []bool{false, true}, reads)
				if broken {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestStatusCaptureActionsRetryWithoutPodEvent(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, action := range []podCleanupAction{removeFinalizer, labelPodCompleted, deletePod, deletePodByUID} {
		for _, failure := range []string{"uncommitted", "forbidden", "storage"} {
			t.Run(action+"/"+failure, func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, pod := captureFixture()
				client := fake.NewSimpleClientset(pod)
				c := identityTestController(t, client)
				q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
				c.workqueue = q
				ready := false
				c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
					if !ready {
						switch failure {
						case "forbidden":
							return nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("denied"))
						case "storage":
							return nil, fmt.Errorf("offload unavailable")
						}
					}
					out := wf.DeepCopy()
					if !ready {
						n := out.Status.Nodes["node"]
						n.CapturedPodUID = ""
						out.Status.Nodes["node"] = n
					}
					return out, nil
				}
				c.workqueue.Add(newPodCleanupKeyWithUID(pod.Namespace, pod.Name, action, string(pod.UID)))
				require.True(t, c.processNextPodCleanupItem(ctx))
				require.Len(t, client.Actions(), 1)
				assert.Equal(t, "get", client.Actions()[0].GetVerb())
				assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
				ready = true
				require.True(t, c.processNextPodCleanupItem(ctx))
				current, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
				if action == deletePod || action == deletePodByUID {
					assert.True(t, apierr.IsNotFound(err))
				} else {
					require.NoError(t, err)
					assert.Equal(t, []string{"example.com/keep"}, current.Finalizers)
					if action == labelPodCompleted {
						assert.Equal(t, "true", current.Labels[common.LabelKeyCompleted])
					}
				}
			})
		}
	}
}

func TestStatusCaptureVerifiedOrphanAndFeatureOff(t *testing.T) {
	for _, mode := range []string{"absent owner", "replaced owner", "flag off", "no barrier"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
			wf, pod := captureFixture()
			c := identityTestController(t, fake.NewSimpleClientset(pod))
			reads := 0
			c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
				reads++
				if mode == "absent owner" {
					return nil, apierr.NewNotFound(schema.GroupResource{Resource: "workflows"}, wf.Name)
				}
				wf.UID = "replacement"
				return wf, nil
			}
			if mode == "flag off" {
				t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
			}
			if mode == "no barrier" {
				pod.Finalizers = nil
			}
			allowed, err := c.allowPodCleanup(t.Context(), pod)
			require.NoError(t, err)
			assert.True(t, allowed)
			if mode == "flag off" || mode == "no barrier" {
				assert.Zero(t, reads)
			} else {
				assert.Equal(t, 1, reads)
			}
		})
	}
}

func TestStatusCaptureRecoveryAfterRestart(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, scenario := range []string{"completed no GC", "delete lost after barrier removal", "retirement", "stopped daemon", "stopped failed", "agent", "deleting pending"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureFixture()
			want := labelPodCompleted
			switch scenario {
			case "delete lost after barrier removal":
				pod.Finalizers = []string{"example.com/keep"}
				wf.Spec.PodGC = &wfv1.PodGC{Strategy: wfv1.PodGCOnWorkflowCompletion, DeleteDelayDuration: "5s"}
				want = deletePod
			case "retirement":
				pod.Status.Phase = apiv1.PodFailed
				n := wf.Status.Nodes["node"]
				n.Phase = wfv1.NodePending
				n.RestartingPodUID = string(pod.UID)
				n.CapturedPodUID = ""
				wf.Status.Nodes["node"] = n
				want = deletePodByUID
			case "stopped daemon", "stopped failed":
				pod.Status.Phase = apiv1.PodRunning
				if scenario == "stopped failed" {
					n := wf.Status.Nodes["node"]
					n.Phase = wfv1.NodeFailed
					wf.Status.Nodes["node"] = n
				}
				want = terminateContainers
			case "agent":
				pod.Labels[common.LabelKeyComponent] = "agent"
				pod.Finalizers = nil
				pod.Status.Phase = apiv1.PodRunning
				wf.Status.Nodes = nil
				want = deletePod
			case "deleting pending":
				pod.Status.Phase = apiv1.PodPending
				now := metav1.Now()
				pod.DeletionTimestamp = &now
				wf.Status.Nodes = nil
				want = removeFinalizer
			}
			c := identityTestController(t, fake.NewSimpleClientset(pod))
			q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
			c.workqueue = q
			c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) { return wf.DeepCopy(), nil }
			c.callBack = func(*apiv1.Pod) error { return nil }
			c.addPodEvent(ctx, pod)
			require.True(t, c.processNextPodCleanupItem(ctx))
			require.Equal(t, 1, c.workqueue.Len())
			key, quit := c.workqueue.Get()
			require.False(t, quit)
			c.workqueue.Done(key)
			assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, want, string(pod.UID)), key)
			if scenario == "delete lost after barrier removal" {
				assert.Equal(t, []time.Duration{5 * time.Second}, q.delays)
			}
		})
	}
}

func TestStatusCaptureWorkflowSweepIsolationAndRetry(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureFixture()
	foreignNamespace := pod.DeepCopy()
	foreignNamespace.Namespace = "other"
	foreignOwner := pod.DeepCopy()
	foreignOwner.Name = "wrong-owner"
	foreignOwner.OwnerReferences[0].UID = "other"
	foreignInstance := pod.DeepCopy()
	foreignInstance.Name = "wrong-instance"
	foreignInstance.Labels[common.LabelKeyControllerInstanceID] = "other"
	client := fake.NewSimpleClientset(pod, foreignNamespace, foreignOwner, foreignInstance)
	c := identityTestController(t, client)
	q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
	c.workqueue = q
	broken := true
	client.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
		if broken {
			return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, "", fmt.Errorf("denied"))
		}
		return false, nil, nil
	})
	c.QueueWorkflowCleanup(ctx, wf)
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
	broken = false
	require.True(t, c.processNextPodCleanupItem(ctx))
	require.Equal(t, 1, c.workqueue.Len())
	key, _ := c.workqueue.Get()
	c.workqueue.Done(key)
	assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, reconcilePodCleanup, string(pod.UID)), key)
}

func TestStatusCaptureRecoveryReadFailureRetriesIdlePod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureFixture()
	pod.Status.Phase = apiv1.PodRunning
	node := wf.Status.Nodes["node"]
	node.Phase = wfv1.NodeRunning
	node.CapturedPodUID = ""
	wf.Status.Nodes["node"] = node
	c := identityTestController(t, fake.NewSimpleClientset(pod))
	q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
	c.workqueue = q
	broken := true
	c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
		if broken {
			return nil, fmt.Errorf("API unavailable")
		}
		return wf.DeepCopy(), nil
	}
	c.ReconcilePodCleanup(ctx, pod)
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
	broken = false
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Zero(t, c.workqueue.Len())
	assert.Len(t, q.delays, 1)
}

func TestStatusCaptureMissingLegacyResultDiagnostic(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	wf, pod := captureFixture()
	n := wf.Status.Nodes["node"]
	n.CapturedPodUID = ""
	wf.Status.Nodes["node"] = n
	c := identityTestController(t, fake.NewSimpleClientset(pod))
	c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) { return wf.DeepCopy(), nil }
	allowed, err := c.allowPodCleanup(t.Context(), pod)
	assert.False(t, allowed)
	require.ErrorContains(t, err, "no supported legacy capture proof")
	now := metav1.Now()
	wf.DeletionTimestamp = &now
	allowed, err = c.allowPodCleanup(t.Context(), pod)
	assert.True(t, allowed)
	require.NoError(t, err)
}

func TestStatusCaptureHydrationNotFoundIsNotOrphan(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	wf, pod := captureFixture()
	wf.Status.OffloadNodeStatusVersion = "missing-version"
	wf.Status.Nodes = nil
	c := identityTestController(t, fake.NewSimpleClientset(pod))
	c.lookupWorkflow = func(_ context.Context, _, _ string, hydrate bool) (*wfv1.Workflow, error) {
		if hydrate {
			return nil, apierr.NewNotFound(schema.GroupResource{Resource: "offloaded-nodes"}, "missing-version")
		}
		return wf.DeepCopy(), nil
	}
	allowed, err := c.allowPodCleanup(t.Context(), pod)
	assert.False(t, allowed)
	require.Error(t, err)
}

func TestStatusCaptureStartupRetriesUncommittedResult(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	ctx := logging.TestContext(t.Context())
	wf, pod := captureFixture()
	client := fake.NewSimpleClientset(pod)
	c := identityTestController(t, client)
	q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
	c.workqueue = q
	c.callBack = func(*apiv1.Pod) error { return nil }
	committed := false
	c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
		out := wf.DeepCopy()
		if !committed {
			node := out.Status.Nodes["node"]
			node.CapturedPodUID = ""
			out.Status.Nodes["node"] = node
		}
		return out, nil
	}
	c.addPodEvent(ctx, pod)
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
	require.Len(t, client.Actions(), 1)
	committed = true
	require.True(t, c.processNextPodCleanupItem(ctx))
	require.True(t, c.processNextPodCleanupItem(ctx))
	result, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, result.Finalizers, common.FinalizerPodStatus)
	assert.Equal(t, "true", result.Labels[common.LabelKeyCompleted])
}

func TestStatusCaptureKeepsStoppedRunningPodInRecoveryWatch(t *testing.T) {
	for _, enabled := range []string{"true", "false"} {
		t.Run(enabled, func(t *testing.T) {
			t.Setenv(common.EnvVarPodStatusCaptureFinalizer, enabled)
			wf, pod := captureFixture()
			pod.Status.Phase = apiv1.PodRunning
			c := identityTestController(t, fake.NewSimpleClientset(pod))
			c.EnactAnyPodCleanup(logging.TestContext(t.Context()), labels.Everything(), pod, wfv1.PodGCOnPodNone, wf.Status.Phase, 0)
			key, _ := c.workqueue.Get()
			c.workqueue.Done(key)
			want := labelPodCompleted
			if enabled == "true" {
				want = reconcilePodCleanup
			}
			assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, want, string(pod.UID)), key)
		})
	}
}
