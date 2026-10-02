package controller

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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
)

func captureWriterFixture() (*wfv1.Workflow, *apiv1.Pod) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "default", UID: "workflow-uid", ResourceVersion: "1"},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "main", Templates: []wfv1.Template{{Name: "main", Container: &apiv1.Container{Image: "busybox"}}}},
		Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, StartedAt: metav1.NewTime(time.Unix(1000, 0)), Nodes: wfv1.Nodes{
			"capture": {ID: "capture", Name: "capture", Type: wfv1.NodeTypePod, Phase: wfv1.NodeRunning, TemplateName: "main", StartedAt: metav1.NewTime(time.Unix(1000, 0)), TaskResultSynced: new(true)},
		}},
	}
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capture-pod", Namespace: wf.Namespace, UID: "pod-uid", ResourceVersion: "1",
			Labels:          map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyCompleted: "false"},
			Annotations:     map[string]string{common.AnnotationKeyNodeID: wf.Name, common.AnnotationKeyNodeName: wf.Name},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind("Workflow"))},
		},
		Status: apiv1.PodStatus{Phase: apiv1.PodSucceeded, ContainerStatuses: []apiv1.ContainerStatus{
			{Name: common.MainContainerName, State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0, StartedAt: metav1.NewTime(time.Unix(1000, 0)), FinishedAt: metav1.NewTime(time.Unix(1001, 0))}}},
			{Name: common.WaitContainerName, State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0, StartedAt: metav1.NewTime(time.Unix(1000, 0)), FinishedAt: metav1.NewTime(time.Unix(1002, 0))}}},
		}},
	}
	return wf, pod
}

func TestCapturedPodUIDSuccessfulAssessment(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	old := wf.Status.Nodes[wf.Name]
	captured := woc.assessNodeStatus(ctx, pod, &old)
	require.NotNil(t, captured)
	assert.Equal(t, string(pod.UID), captured.CapturedPodUID)
	assert.Equal(t, wfv1.NodeSucceeded, captured.Phase)
	assert.Equal(t, "0", *captured.Outputs.ExitCode)
	assert.Nil(t, woc.assessNodeStatus(ctx, pod, captured), "same successful observation is idempotent")

	legacy := captured.DeepCopy()
	legacy.CapturedPodUID = ""
	captured = woc.assessNodeStatus(ctx, pod, legacy)
	require.NotNil(t, captured, "capturing an unchanged result must itself create a persistent delta")
	assert.Equal(t, string(pod.UID), captured.CapturedPodUID)

	replacement := pod.DeepCopy()
	replacement.UID = "replacement"
	captured = woc.assessNodeStatus(ctx, replacement, captured)
	require.NotNil(t, captured)
	assert.Equal(t, "replacement", captured.CapturedPodUID)

	foreign := replacement.DeepCopy()
	foreign.OwnerReferences[0].UID = "another-workflow"
	captured = woc.assessNodeStatus(ctx, foreign, captured)
	require.NotNil(t, captured)
	assert.Empty(t, captured.CapturedPodUID, "a label/name match is not ownership")
}

func TestCapturedPodUIDDaemonObservationSurvivesTeardown(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	wf.Spec.Templates[0].Daemon = new(true)
	pod.Status.Phase = apiv1.PodRunning
	pod.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: common.MainContainerName, Ready: true, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.NewTime(time.Unix(1000, 0))}}}}
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	old := wf.Status.Nodes[wf.Name]
	captured := woc.assessNodeStatus(ctx, pod, &old)
	require.NotNil(t, captured)
	assert.True(t, captured.IsDaemoned())
	assert.Equal(t, string(pod.UID), captured.CapturedPodUID)
	woc.wf.Status.Nodes[wf.Name] = *captured
	woc.killDaemonedChildren(ctx, "")
	stopped := woc.wf.Status.Nodes[wf.Name]
	assert.Equal(t, wfv1.NodeSucceeded, stopped.Phase)
	assert.Equal(t, string(pod.UID), stopped.CapturedPodUID)
}

func TestCapturedPodUIDTemplateFailureDoesNotReuseObservation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	old := wf.Status.Nodes[wf.Name]
	old.CapturedPodUID = string(pod.UID)
	old.TemplateScope = "cluster/missing"
	old.TemplateRef = &wfv1.TemplateRef{Name: "missing", Template: "main", ClusterScope: true}
	wf.Status.Nodes[wf.Name] = old
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.Nil(t, woc.assessNodeStatus(ctx, pod, &old))
	assert.Empty(t, woc.wf.Status.Nodes[wf.Name].CapturedPodUID)
	assert.True(t, woc.updated)
}

type captureFailingMemoization struct{}

func (captureFailingMemoization) GetCache(controllercache.Type, string) controllercache.MemoizationCache {
	return captureFailingMemoization{}
}
func (captureFailingMemoization) Load(context.Context, string) (*controllercache.Entry, error) {
	return nil, nil
}
func (captureFailingMemoization) Save(context.Context, string, string, *wfv1.Outputs) error {
	return fmt.Errorf("memoization write unavailable")
}

func TestCapturedPodUIDPersistsFinalMemoizationError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	node := wf.Status.Nodes[wf.Name]
	node.MemoizationStatus = &wfv1.MemoizationStatus{CacheName: "cache", Key: "key"}
	wf.Status.Nodes[wf.Name] = node
	cancel, controller := newController(ctx, wf)
	defer cancel()
	controller.cacheFactory = captureFailingMemoization{}
	require.NoError(t, controller.PodController.TestingPodInformer().GetIndexer().Add(pod))
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	complete, err := woc.podReconciliation(ctx)
	require.NoError(t, err)
	assert.True(t, complete)
	woc.persistUpdates(ctx)
	persisted, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	got := persisted.Status.Nodes[wf.Name]
	assert.Equal(t, wfv1.NodeError, got.Phase)
	assert.Equal(t, "memoization write unavailable", got.Message)
	assert.Equal(t, string(pod.UID), got.CapturedPodUID)
	assert.Equal(t, "0", *got.Outputs.ExitCode)
}

func TestCapturedPodUIDPersistenceFailuresDoNotPublishCleanup(t *testing.T) {
	for _, failure := range []string{"forbidden", "conflict", "timeout", "committedLostReply"} {
		t.Run(failure, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureWriterFixture()
			cancel, controller := newController(ctx, wf)
			defer cancel()
			require.NoError(t, controller.PodController.TestingPodInformer().GetIndexer().Add(pod))
			client := controller.wfclientset.(*fakewfclientset.Clientset)
			client.PrependReactor("update", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
				switch failure {
				case "forbidden":
					return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("denied"))
				case "conflict":
					return true, nil, apierr.NewConflict(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("still conflicting"))
				case "committedLostReply":
					update := action.(k8stesting.UpdateAction).GetObject().(*wfv1.Workflow)
					require.NoError(t, client.Tracker().Update(wfv1.SchemeGroupVersion.WithResource("workflows"), update.DeepCopy(), wf.Namespace))
				}
				return true, nil, apierr.NewTimeoutError("reply lost", 1)
			})
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			old := wf.Status.Nodes[wf.Name]
			captured := woc.assessNodeStatus(ctx, pod, &old)
			require.NotNil(t, captured)
			woc.wf.Status.Nodes[wf.Name] = *captured
			woc.updated = true
			woc.persistUpdates(ctx)
			assert.True(t, woc.reapplyFailed)
			for _, action := range []string{"labelPodCompleted", "removeFinalizer", "deletePod", "deletePodByUID"} {
				key := fmt.Sprintf("%s/%s/%s/%s", pod.Namespace, pod.Name, action, pod.UID)
				assert.Zero(t, controller.PodController.TestingQueueNumRequeues(key), "an unknown/failed write must not publish ordinary cleanup")
			}
			persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			if failure == "committedLostReply" {
				assert.Equal(t, string(pod.UID), persisted.Status.Nodes[wf.Name].CapturedPodUID, "reader recovery must distinguish the actual committed result")
			} else {
				assert.Empty(t, persisted.Status.Nodes[wf.Name].CapturedPodUID)
			}
		})
	}
}

func TestCapturedPodUIDReapplyRetainsMergedNodes(t *testing.T) {
	for _, offload := range []bool{false, true} {
		t.Run(fmt.Sprintf("offload=%v", offload), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			original, pod := captureWriterFixture()
			current := original.DeepCopy()
			current.ResourceVersion = "2"
			current.Status.Nodes["other"] = wfv1.NodeStatus{ID: "other", Name: "other", Phase: wfv1.NodeSucceeded, CapturedPodUID: "other-pod"}
			cancel, controller := newController(ctx, current)
			defer cancel()
			if offload {
				controller.hydrator = hydratorfake.Always
			}
			woc := newWorkflowOperationCtx(ctx, original, controller)
			old := original.Status.Nodes[original.Name]
			captured := woc.assessNodeStatus(ctx, pod, &old)
			require.NotNil(t, captured)
			woc.wf.Status.Nodes[original.Name] = *captured
			got, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(original.Namespace), woc.wf.Status.Nodes)
			require.NoError(t, err)
			assert.Equal(t, string(pod.UID), got.Status.Nodes[original.Name].CapturedPodUID)
			assert.Equal(t, "other-pod", got.Status.Nodes["other"].CapturedPodUID, "return the merged representation, not the old operation's nodes")
			persisted, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(original.Namespace).Get(ctx, original.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.NoError(t, controller.hydrator.Hydrate(ctx, persisted))
			assert.Equal(t, got.Status.Nodes, persisted.Status.Nodes)
		})
	}
}

func TestCapturedPodUIDReapplyRejectsDifferentIncarnation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	original, _ := captureWriterFixture()
	old := original.Status.Nodes[original.Name]
	old.CapturedPodUID = "old-pod"
	original.Status.Nodes[original.Name] = old
	current := original.DeepCopy()
	newer := old
	newer.CapturedPodUID = "replacement"
	current.Status.Nodes[current.Name] = newer
	current.ResourceVersion = "2"
	cancel, controller := newController(ctx, current)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, original, controller)
	stale := old
	stale.Message = "old incarnation result"
	woc.wf.Status.Nodes[original.Name] = stale
	_, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(original.Namespace), woc.wf.Status.Nodes)
	require.EqualError(t, err, "captured pod changed for node capture")
	persisted, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(original.Namespace).Get(ctx, original.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "replacement", persisted.Status.Nodes[original.Name].CapturedPodUID)
	assert.Empty(t, persisted.Status.Nodes[original.Name].Message)
}

func replaceCapturePodUID(pod *apiv1.Pod, uid string) *apiv1.Pod {
	replacement := pod.DeepCopy()
	replacement.UID = types.UID(uid)
	return replacement
}

func TestCapturedPodUIDReapplyRejectsConcurrentRetirementChange(t *testing.T) {
	for _, completedRestart := range []bool{false, true} {
		t.Run(fmt.Sprintf("completedRestart=%v", completedRestart), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			original, pod := captureWriterFixture()
			old := original.Status.Nodes[original.Name]
			old.CapturedPodUID = "previous"
			if completedRestart {
				old.CapturedPodUID = ""
				old.Phase = wfv1.NodePending
				old.RestartingPodUID = "previous"
			}
			original.Status.Nodes[original.Name] = old
			current := original.DeepCopy()
			current.ResourceVersion = "2"
			newer := old
			newer.CapturedPodUID = ""
			if completedRestart {
				newer.Phase = wfv1.NodeRunning
				newer.RestartingPodUID = ""
			} else {
				newer.Phase = wfv1.NodePending
				newer.RestartingPodUID = "previous"
			}
			current.Status.Nodes[current.Name] = newer
			cancel, controller := newController(ctx, current)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, original, controller)
			stale := woc.assessNodeStatus(ctx, pod, &old)
			require.NotNil(t, stale)
			woc.wf.Status.Nodes[original.Name] = *stale
			_, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(original.Namespace), woc.wf.Status.Nodes)
			require.EqualError(t, err, "captured pod changed for node capture")
			persisted, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(original.Namespace).Get(ctx, original.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, newer, persisted.Status.Nodes[original.Name])
		})
	}
}

func TestCapturedPodUIDSuccessfulConflictReapplyPublishesCommittedResult(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	original, pod := captureWriterFixture()
	current := original.DeepCopy()
	current.ResourceVersion = "2"
	current.Status.Nodes["other"] = wfv1.NodeStatus{ID: "other", Name: "other", Phase: wfv1.NodeSucceeded, CapturedPodUID: "other-pod"}
	cancel, controller := newController(ctx, current)
	defer cancel()
	require.NoError(t, controller.PodController.TestingPodInformer().GetIndexer().Add(pod))
	controller.hydrator = hydratorfake.Always
	client := controller.wfclientset.(*fakewfclientset.Clientset)
	updates := 0
	key := fmt.Sprintf("%s/%s/labelPodCompleted/%s", pod.Namespace, pod.Name, pod.UID)
	client.PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
		updates++
		assert.Zero(t, controller.PodController.TestingQueueNumRequeues(key), "no cleanup intent before the successful Update returns")
		if updates == 1 {
			return true, nil, apierr.NewConflict(schema.GroupResource{Resource: "workflows"}, original.Name, fmt.Errorf("changed"))
		}
		return false, nil, nil
	})
	woc := newWorkflowOperationCtx(ctx, original, controller)
	old := original.Status.Nodes[original.Name]
	captured := woc.assessNodeStatus(ctx, pod, &old)
	require.NotNil(t, captured)
	woc.wf.Status.Nodes[original.Name] = *captured
	woc.updated = true
	woc.persistUpdates(ctx)
	require.False(t, woc.reapplyFailed)
	assert.Equal(t, 2, updates)
	assert.Equal(t, "other-pod", woc.wf.Status.Nodes["other"].CapturedPodUID)
	persisted, err := client.ArgoprojV1alpha1().Workflows(original.Namespace).Get(ctx, original.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NoError(t, controller.hydrator.Hydrate(ctx, persisted))
	assert.Equal(t, persisted.Status.Nodes, woc.wf.Status.Nodes)
	assert.Equal(t, string(pod.UID), persisted.Status.Nodes[original.Name].CapturedPodUID)
	assert.Equal(t, 1, controller.PodController.TestingQueueNumRequeues(key))
}
