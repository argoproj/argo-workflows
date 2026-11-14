package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

const retryOnTransientErrorWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: lost-node
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    retryStrategy:
      limit: 2
      retryPolicy: OnTransientError
    container:
      image: busybox
      command: [sleep, "3600"]
`

// lostNodePod runs the workflow until its pod is Running, then makes the pod Failed while being deleted with no
// container status, as when its node is lost and the pod force-deleted, and operates once.
func lostNodePod(ctx context.Context, t *testing.T) (*WorkflowController, *wfOperationCtx) {
	t.Helper()
	wf := wfv1.MustUnmarshalWorkflow(retryOnTransientErrorWf)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	updatePod(ctx, t, woc, func(pod *apiv1.Pod) {
		now := metav1.Now()
		pod.DeletionTimestamp = &now
		pod.Finalizers = []string{common.FinalizerPodStatus}
		pod.Status = apiv1.PodStatus{Phase: apiv1.PodFailed}
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	return controller, woc
}

// TestPodFailedWhileDeletingWithoutContainerStatuses checks that a pod force-deleted from a lost node, observed Failed
// before the kubelet has reported any container status, has its retry policy evaluated on the status the kubelet
// reports, not on the "can't find failed message" placeholder.
func TestPodFailedWhileDeletingWithoutContainerStatuses(t *testing.T) {
	t.Setenv("TRANSIENT_ERROR_PATTERN", `ContainerStatusUnknown \(exit code 137\)`)
	ctx := logging.TestContext(t.Context())
	controller, woc := lostNodePod(ctx, t)

	retryNode, err := woc.wf.GetNodeByName("lost-node")
	require.NoError(t, err)
	assert.False(t, retryNode.Fulfilled(), "retry node closed on a placeholder: %s", retryNode.Message)

	// The kubelet of the recreated node reports the containers it could not find.
	updatePod(ctx, t, woc, func(pod *apiv1.Pod) {
		unknown := apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{
			Reason:   "ContainerStatusUnknown",
			ExitCode: 137,
			Message:  "The container could not be located when the pod was terminated",
		}}
		pod.Status.InitContainerStatuses = []apiv1.ContainerStatus{{Name: common.InitContainerName, State: unknown}}
		pod.Status.ContainerStatuses = []apiv1.ContainerStatus{
			{Name: common.WaitContainerName, State: unknown},
			{Name: common.MainContainerName, State: unknown},
		}
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	retryNode, err = woc.wf.GetNodeByName("lost-node")
	require.NoError(t, err)
	assert.Len(t, retryNode.Children, 2, "expected a second attempt, retry node is %s: %s", retryNode.Phase, retryNode.Message)
}

// TestPodFailedWhileDeletingWithoutContainerStatusesThenGone checks that when no kubelet reports the pod's statuses,
// the pod goes away and its node is recorded as "pod deleted", which the retry policy then decides on.
func TestPodFailedWhileDeletingWithoutContainerStatusesThenGone(t *testing.T) {
	t.Setenv("TRANSIENT_ERROR_PATTERN", "pod deleted")
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
	ctx := logging.TestContext(t.Context())
	controller, woc := lostNodePod(ctx, t)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	require.NoError(t, woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}))
	require.NoError(t, kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		_, exists, getErr := controller.PodController.TestingPodInformer().GetStore().Get(&pod)
		return !exists, getErr
	}))
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	podNode, err := woc.wf.GetNodeByName("lost-node(0)")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, podNode.Phase)
	assert.Equal(t, "pod deleted", podNode.Message)
	retryNode, err := woc.wf.GetNodeByName("lost-node")
	require.NoError(t, err)
	assert.Len(t, retryNode.Children, 2, "expected a second attempt, retry node is %s: %s", retryNode.Phase, retryNode.Message)
}

// updatePod applies mutate to the workflow's only pod and waits for the pod informer to observe the change.
func updatePod(ctx context.Context, t *testing.T, woc *wfOperationCtx, mutate func(pod *apiv1.Pod)) {
	t.Helper()
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace)
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	mutate(&pod)
	updated, err := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, woc.controller.PodController.TestingPodInformer(), updated, func(obj any) bool {
		stored := obj.(*apiv1.Pod)
		return stored.DeletionTimestamp.Equal(updated.DeletionTimestamp) && apiequality.Semantic.DeepEqual(stored.Status, updated.Status)
	})
}
