package controller

import (
	"context"
	"fmt"
	"slices"
	"sync"
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
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	podcontroller "github.com/argoproj/argo-workflows/v4/workflow/controller/pod"
)

// Reconstruct work through the production informer's initial LIST/Add, with
// zero automatic workers so each actual queue action has an observable oracle.
// The Workflow callback uses the writer's client and store, not a separate stub.
func captureBridgeStartup(ctx context.Context, t *testing.T, controller *WorkflowController, client *fake.Clientset, expectedRelease bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader := podcontroller.NewController(ctx, &controller.Config, controller.restConfig, "default", client, controller.wfInformer, controller.metrics, func(*apiv1.Pod) error { return nil }, controller.lookupWorkflowForPodCleanup)
	go reader.Run(ctx, 0)
	require.True(t, cache.WaitForCacheSync(ctx.Done(), reader.TestingPodInformer().HasSynced))
	require.Eventually(t, func() bool { return reader.TestingQueueLen() > 0 }, time.Second, time.Millisecond)
	require.True(t, reader.TestingProcessNextItem(ctx)) // startup recovery reads authoritative Workflow
	if expectedRelease {
		require.Eventually(t, func() bool { return reader.TestingQueueLen() > 0 }, time.Second, time.Millisecond)
		require.True(t, reader.TestingProcessNextItem(ctx)) // barrier mutation repeats the authoritative check
	}
}

func captureBridgePod(ctx context.Context, t *testing.T, client *fake.Clientset, pod *apiv1.Pod, released bool) {
	t.Helper()
	current, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, pod.UID, current.UID)
	assert.Equal(t, pod.ResourceVersion, current.ResourceVersion, "the fake API does not generate RVs; no Pod update/event is injected")
	assert.Equal(t, !released, hasCaptureBridgeFinalizer(current))
	assert.Contains(t, current.Finalizers, "example.com/keep")
	if released {
		assert.Equal(t, "true", current.Labels[common.LabelKeyCompleted])
	}
}

func hasCaptureBridgeFinalizer(pod *apiv1.Pod) bool {
	return slices.Contains(pod.Finalizers, common.FinalizerPodStatus)
}

func TestCapturedPodUnknownCommitRecoveryBridge(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprintf("committed=%v", committed), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureWriterFixture()
			pod.Finalizers = []string{common.FinalizerPodStatus, "example.com/keep"}
			podClient := fake.NewClientset(pod)
			cancel, controller := newController(ctx, wf)
			defer cancel()
			client := controller.wfclientset.(*fakewfclientset.Clientset)
			loseReply := true
			client.PrependReactor("update", "workflows", func(action clienttesting.Action) (bool, runtime.Object, error) {
				if !loseReply {
					return false, nil, nil
				}
				if committed {
					candidate := action.(clienttesting.UpdateAction).GetObject().(*wfv1.Workflow)
					require.NoError(t, client.Tracker().Update(wfv1.SchemeGroupVersion.WithResource("workflows"), candidate.DeepCopy(), wf.Namespace))
				}
				return true, nil, apierr.NewTimeoutError("Workflow Update response lost", 1)
			})
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			old := wf.Status.Nodes[wf.Name]
			captured := woc.assessNodeStatus(ctx, pod, &old)
			require.NotNil(t, captured)
			woc.wf.Status.Nodes[wf.Name] = *captured
			woc.wf.Status.Phase = wfv1.WorkflowSucceeded
			woc.wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			woc.updated = true
			woc.persistUpdates(ctx)
			require.True(t, woc.reapplyFailed)
			require.Zero(t, controller.PodController.TestingQueueLen())

			captureBridgeStartup(ctx, t, controller, podClient, committed)
			captureBridgePod(ctx, t, podClient, pod, committed)
			if !committed {
				for _, action := range podClient.Actions() {
					assert.NotEqual(t, "patch", action.GetVerb())
					assert.NotEqual(t, "delete", action.GetVerb())
				}
				// A later real writer commit resolves the uncertainty. Rebuild the
				// lost queue again from the unchanged Pod, without reassessment.
				loseReply = false
				persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
				require.NoError(t, err)
				retry := newWorkflowOperationCtx(ctx, persisted, controller)
				retry.wf.Status = *woc.wf.Status.DeepCopy()
				retry.wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
				retry.updated = true
				retry.persistUpdates(ctx)
				require.False(t, retry.reapplyFailed)
				captureBridgeStartup(ctx, t, controller, podClient, true)
				captureBridgePod(ctx, t, podClient, pod, true)
			}
			persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			actual := persisted.Status.Nodes[wf.Name]
			assert.Equal(t, captured.Phase, actual.Phase)
			assert.Equal(t, captured.CapturedPodUID, actual.CapturedPodUID)
			assert.Equal(t, captured.Outputs, actual.Outputs)
			assert.Equal(t, captured.FinishedAt, actual.FinishedAt)
			assert.Equal(t, wfv1.WorkflowSucceeded, persisted.Status.Phase)
		})
	}
}

func TestCapturedPodControlTerminalRaceBridge(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, phase := range []apiv1.PodPhase{apiv1.PodPending, apiv1.PodRunning} {
		for _, strategy := range []wfv1.ShutdownStrategy{wfv1.ShutdownStrategyStop, wfv1.ShutdownStrategyTerminate} {
			t.Run(string(phase)+"/"+string(strategy), func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, observed := captureWriterFixture()
				node := wf.Status.Nodes[wf.Name]
				node.Phase, node.TaskResultSynced = wfv1.NodePhase(phase), new(false)
				wf.Status.Nodes[wf.Name] = node
				observed.Status.Phase = phase
				observed.Status.ContainerStatuses = nil
				observed.Finalizers = []string{common.FinalizerPodStatus, "example.com/keep"}
				cancel, controller := newController(ctx, wf)
				defer cancel()
				woc := newWorkflowOperationCtx(ctx, wf, controller)
				woc.execWf.Spec.Shutdown = strategy
				woc.applyExecutionControl(ctx, observed, &sync.RWMutex{})
				selected := woc.wf.Status.Nodes[wf.Name]
				require.Equal(t, wfv1.NodeFailed, selected.Phase)
				require.Equal(t, string(observed.UID), selected.CapturedPodUID)

				// This is the actual Pod object returned to the reader's API GET:
				// same UID, now terminal, before the selected Failed result commits.
				terminal := observed.DeepCopy()
				terminal.Status.Phase = apiv1.PodFailed
				terminal.Status.Message = "terminated during Workflow persistence"
				podClient := fake.NewClientset(terminal)
				client := controller.wfclientset.(*fakewfclientset.Clientset)
				denyWrite := true
				client.PrependReactor("update", "workflows", func(clienttesting.Action) (bool, runtime.Object, error) {
					if denyWrite {
						return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("persistence delayed"))
					}
					return false, nil, nil
				})
				woc.persistUpdates(ctx)
				require.True(t, woc.reapplyFailed)
				captureBridgeStartup(ctx, t, controller, podClient, false)
				captureBridgePod(ctx, t, podClient, terminal, false)

				denyWrite = false
				persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
				require.NoError(t, err)
				retry := newWorkflowOperationCtx(ctx, persisted, controller)
				retry.wf.Status.Nodes[wf.Name] = selected
				retry.updated = true
				retry.persistUpdates(ctx)
				require.False(t, retry.reapplyFailed)
				released := phase == apiv1.PodPending
				captureBridgeStartup(ctx, t, controller, podClient, released)
				captureBridgePod(ctx, t, podClient, terminal, released)
				if !released {
					persisted, err = client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
					require.NoError(t, err)
					retry = newWorkflowOperationCtx(ctx, persisted, controller)
					retry.wf.Status.MarkTaskResultComplete(ctx, wf.Name)
					retry.updated = true
					retry.persistUpdates(ctx)
					captureBridgeStartup(ctx, t, controller, podClient, true)
					captureBridgePod(ctx, t, podClient, terminal, true)
				}
				persisted, err = client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, selected.Phase, persisted.Status.Nodes[wf.Name].Phase)
				assert.Equal(t, selected.Message, persisted.Status.Nodes[wf.Name].Message)
				assert.Equal(t, selected.CapturedPodUID, persisted.Status.Nodes[wf.Name].CapturedPodUID)
			})
		}
	}
}
