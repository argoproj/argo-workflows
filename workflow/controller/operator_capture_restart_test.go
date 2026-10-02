package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestCapturedPodRestartWaitsForPersistence(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	pod.Status.Phase, pod.Status.Reason = apiv1.PodFailed, "Evicted"
	pod.Status.ContainerStatuses = nil
	old := wf.Status.Nodes[wf.Name]
	old.CapturedPodUID = string(pod.UID)
	wf.Status.Nodes[wf.Name] = old
	cancel, controller := newController(ctx, wf)
	defer cancel()
	controller.Config.FailedPodRestart = &config.FailedPodRestartConfig{Enabled: true}
	require.NoError(t, controller.PodController.TestingPodInformer().GetIndexer().Add(pod))
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	restarting := woc.assessNodeStatus(ctx, pod, &old)
	require.NotNil(t, restarting)
	assert.Equal(t, wfv1.NodePending, restarting.Phase)
	assert.Equal(t, string(pod.UID), restarting.RestartingPodUID)
	assert.Empty(t, restarting.CapturedPodUID)
	assert.EqualValues(t, 1, restarting.FailedPodRestarts)
	assert.Zero(t, controller.PodController.TestingQueueNumRequeues(fmt.Sprintf("%s/%s/deletePodByUID/%s", pod.Namespace, pod.Name, pod.UID)), "assessment must not enqueue pre-persistence deletion")
	woc.wf.Status.Nodes[wf.Name] = *restarting
	woc.updated = true
	client := controller.wfclientset.(*fakewfclientset.Clientset)
	client.PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("denied"))
	})
	woc.persistUpdates(ctx)
	assert.Zero(t, controller.PodController.TestingQueueNumRequeues(fmt.Sprintf("%s/%s/deletePodByUID/%s", pod.Namespace, pod.Name, pod.UID)), "failed Pending persistence must not publish retirement")
	persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Empty(t, persisted.Status.Nodes[wf.Name].RestartingPodUID)
}

func TestCapturedPodRestartRedeliversCommittedDisposition(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	pod.Status.Phase, pod.Status.Reason = apiv1.PodFailed, "Evicted"
	pod.Status.ContainerStatuses = nil
	pending := wf.Status.Nodes[wf.Name]
	pending.Phase, pending.RestartingPodUID, pending.FailedPodRestarts = wfv1.NodePending, string(pod.UID), 3
	pending.TemplateScope = "cluster/removed"
	pending.TemplateRef = &wfv1.TemplateRef{Name: "removed", Template: "main", ClusterScope: true}
	wf.Status.Nodes[wf.Name] = pending
	cancel, controller := newController(ctx, wf)
	defer cancel()
	controller.Config.FailedPodRestart = &config.FailedPodRestartConfig{Enabled: false, MaxRestarts: new(int32(1))}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.Nil(t, woc.assessNodeStatus(ctx, pod, &pending))
	assert.False(t, woc.updated)
	assert.Zero(t, controller.PodController.TestingQueueLen())
	woc.persistUpdates(ctx)
	assert.Equal(t, 1, controller.PodController.TestingQueueNumRequeues(fmt.Sprintf("%s/%s/deletePodByUID/%s", pod.Namespace, pod.Name, pod.UID)), "already-persisted retirement must re-drive without a new status delta")
	assert.EqualValues(t, 3, woc.wf.Status.Nodes[wf.Name].FailedPodRestarts)
	assert.Equal(t, string(pod.UID), woc.wf.Status.Nodes[wf.Name].RestartingPodUID)
}

func TestCapturedPodRestartReplacementAndFailedAfterMain(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	cancel, controller := newController(ctx, wf)
	defer cancel()
	controller.Config.FailedPodRestart = &config.FailedPodRestartConfig{Enabled: true}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	pending := wf.Status.Nodes[wf.Name]
	pending.Phase, pending.RestartingPodUID, pending.FailedPodRestarts = wfv1.NodePending, string(pod.UID), 1
	replacement := replaceCapturePodUID(pod, "replacement")
	replacement.Status.Phase = apiv1.PodRunning
	replacement.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: common.MainContainerName, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.NewTime(time.Unix(1003, 0))}}}}
	running := woc.assessNodeStatus(ctx, replacement, &pending)
	require.NotNil(t, running)
	assert.Empty(t, running.RestartingPodUID)
	assert.Empty(t, running.CapturedPodUID, "ordinary Running is not a terminal capture permit")
	assert.EqualValues(t, 1, running.FailedPodRestarts)

	failed := replacement.DeepCopy()
	failed.Status.Phase, failed.Status.Reason = apiv1.PodFailed, "Evicted"
	failed.Status.ContainerStatuses[0].State = apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1, StartedAt: metav1.NewTime(time.Unix(1003, 0)), FinishedAt: metav1.NewTime(time.Unix(1004, 0))}}
	terminal := woc.assessNodeStatus(ctx, failed, running)
	require.NotNil(t, terminal)
	assert.Equal(t, wfv1.NodeFailed, terminal.Phase)
	assert.Empty(t, terminal.RestartingPodUID)
	assert.Equal(t, "replacement", terminal.CapturedPodUID)
	assert.EqualValues(t, 1, terminal.FailedPodRestarts)
}

func TestCapturedPodRestartPublishesOnlyCommittedDisposition(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	pod.Status.Phase, pod.Status.Reason = apiv1.PodFailed, "Evicted"
	pod.Status.ContainerStatuses = nil
	cancel, controller := newController(ctx, wf)
	defer cancel()
	controller.Config.FailedPodRestart = &config.FailedPodRestartConfig{Enabled: true}
	require.NoError(t, controller.PodController.TestingPodInformer().GetIndexer().Add(pod))
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	old := wf.Status.Nodes[wf.Name]
	pending := woc.assessNodeStatus(ctx, pod, &old)
	require.NotNil(t, pending)
	woc.wf.Status.Nodes[wf.Name] = *pending
	woc.updated = true
	key := fmt.Sprintf("%s/%s/deletePodByUID/%s", pod.Namespace, pod.Name, pod.UID)
	client := controller.wfclientset.(*fakewfclientset.Clientset)
	client.PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
		assert.Zero(t, controller.PodController.TestingQueueNumRequeues(key), "no retirement intent before Workflow Update commits")
		return false, nil, nil
	})
	woc.persistUpdates(ctx)
	require.False(t, woc.reapplyFailed)
	persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodePending, persisted.Status.Nodes[wf.Name].Phase)
	assert.Equal(t, string(pod.UID), persisted.Status.Nodes[wf.Name].RestartingPodUID)
	assert.Empty(t, persisted.Status.Nodes[wf.Name].CapturedPodUID)
	assert.Equal(t, 1, controller.PodController.TestingQueueNumRequeues(key))
}
