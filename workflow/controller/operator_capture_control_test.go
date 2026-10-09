package controller

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestCapturedPodControlDisposition(t *testing.T) {
	for _, phase := range []apiv1.PodPhase{apiv1.PodPending, apiv1.PodRunning} {
		for _, strategy := range []string{"Stop", "Terminate", "deadline"} {
			t.Run(string(phase)+"/"+strategy, func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, observed := captureWriterFixture()
				node := wf.Status.Nodes[wf.Name]
				node.Phase = wfv1.NodePhase(phase)
				node.TaskResultSynced = new(false)
				wf.Status.Nodes[wf.Name] = node
				observed.Status.Phase = phase
				observed.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: common.MainContainerName, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.NewTime(time.Unix(1000, 0))}}}}
				cancel, controller := newController(ctx, wf)
				defer cancel()
				woc := newWorkflowOperationCtx(ctx, wf, controller)
				if strategy == "deadline" {
					expired := time.Now().Add(-time.Minute)
					woc.workflowDeadline = &expired
				} else {
					woc.execWf.Spec.Shutdown = wfv1.ShutdownStrategy(strategy)
				}
				woc.applyExecutionControl(ctx, observed, &sync.RWMutex{})
				selected := woc.wf.Status.Nodes[wf.Name]
				assert.Equal(t, wfv1.NodeFailed, selected.Phase)
				assert.Equal(t, string(observed.UID), selected.CapturedPodUID)
				assert.Equal(t, phase == apiv1.PodPending, *selected.TaskResultSynced, "preserve existing pending/running task-result semantics")

				// A Pod terminal transition does not change this already selected
				// UID-bound controller stop result while persistence is in flight.
				terminal := observed.DeepCopy()
				terminal.Status.Phase = apiv1.PodFailed
				woc.persistUpdates(ctx)
				persisted, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, string(terminal.UID), persisted.Status.Nodes[wf.Name].CapturedPodUID)
				assert.Equal(t, wfv1.NodeFailed, persisted.Status.Nodes[wf.Name].Phase)
			})
		}
	}
}

func TestCapturedPodOrdinaryRunningDoesNotAuthorizeLaterError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	pod.Status.Phase = apiv1.PodRunning
	pod.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: common.MainContainerName, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.NewTime(time.Unix(1000, 0))}}}}
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	old := wf.Status.Nodes[wf.Name]
	if observed := woc.assessNodeStatus(ctx, pod, &old); observed != nil {
		woc.wf.Status.Nodes[wf.Name] = *observed
	}
	assert.Empty(t, woc.wf.Status.Nodes[wf.Name].CapturedPodUID)
	failed := woc.markNodeError(ctx, wf.Name, assert.AnError)
	assert.Empty(t, failed.CapturedPodUID)
}

func TestCapturedPodLegacyStoppedDaemonDoesNotInventIdentity(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	wf.Spec.Templates[0].Daemon = new(true)
	pod.Status.Phase = apiv1.PodFailed
	old := wf.Status.Nodes[wf.Name]
	old.Phase = wfv1.NodeSucceeded
	wf.Status.Nodes[wf.Name] = old
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	updated := woc.assessNodeStatus(ctx, pod, &old)
	require.NotNil(t, updated)
	assert.Equal(t, wfv1.NodeSucceeded, updated.Phase, "retain the controller's stopped-daemon interpretation")
	assert.Empty(t, updated.CapturedPodUID, "old fulfilled daemon/name alone does not identify the stopped incarnation")
}

func TestCapturedPodControlRejectsWrongOwner(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := captureWriterFixture()
	pod.OwnerReferences[0].UID = "another-workflow"
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.handleExecutionControlError(ctx, wf.Name, &sync.RWMutex{}, "stopped", pod)
	assert.Empty(t, woc.wf.Status.Nodes[wf.Name].CapturedPodUID)
}

func TestCapturedPodForcedTaskResultCompletion(t *testing.T) {
	for _, reason := range []string{"auxFailure", "eviction"} {
		for _, legacy := range []bool{false, true} {
			t.Run(reason+"/legacy="+fmt.Sprint(legacy), func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, pod := captureWriterFixture()
				old := wf.Status.Nodes[wf.Name]
				old.TaskResultSynced = new(false)
				if legacy {
					old.TaskResultSynced = nil
					wf.Status.TaskResultsCompletionStatus = map[string]bool{wf.Name: false}
				}
				wf.Status.Nodes[wf.Name] = old
				pod.Status.Phase = apiv1.PodFailed
				if reason == "eviction" {
					pod.Status.Reason = "Evicted"
					pod.Status.ContainerStatuses = pod.Status.ContainerStatuses[:1]
					pod.Status.ContainerStatuses[0].State.Terminated.ExitCode = 1
				} else {
					pod.Status.ContainerStatuses[1].State.Terminated.ExitCode = 1
				}
				cancel, controller := newController(ctx, wf)
				defer cancel()
				woc := newWorkflowOperationCtx(ctx, wf, controller)
				updated := woc.assessNodeStatus(ctx, pod, &old)
				require.NotNil(t, updated)
				woc.wf.Status.Nodes[wf.Name] = *updated // actual caller installs the returned candidate
				assert.False(t, woc.wf.Status.IsTaskResultIncomplete(wf.Name))
				assert.True(t, updated.Phase.Fulfilled(updated.TaskResultSynced))
				assert.Equal(t, string(pod.UID), updated.CapturedPodUID)
				if legacy {
					assert.Nil(t, updated.TaskResultSynced)
				} else {
					require.NotNil(t, updated.TaskResultSynced)
					assert.True(t, *updated.TaskResultSynced)
				}
			})
		}
	}
}

func TestCapturedPodFinalErrorRetainsAssessedIncarnation(t *testing.T) {
	for _, daemon := range []bool{false, true} {
		t.Run(fmt.Sprintf("daemon=%v", daemon), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := captureWriterFixture()
			if daemon {
				wf.Spec.Templates[0].Daemon = new(true)
				pod.Status.Phase = apiv1.PodRunning
				pod.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: common.MainContainerName, Ready: true, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.NewTime(time.Unix(1000, 0))}}}}
			}
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			old := wf.Status.Nodes[wf.Name]
			captured := woc.assessNodeStatus(ctx, pod, &old)
			require.NotNil(t, captured)
			require.Equal(t, string(pod.UID), captured.CapturedPodUID)
			woc.wf.Status.Nodes[wf.Name] = *captured
			woc.updated = true
			// Persistence fallback and other final controller interpretations
			// must retain provenance of a successful assessment.
			woc.markWorkflowError(ctx, assert.AnError)
			assert.Equal(t, string(pod.UID), woc.wf.Status.Nodes[wf.Name].CapturedPodUID)
			failed := woc.markNodeError(ctx, wf.Name, assert.AnError)
			assert.Equal(t, string(pod.UID), failed.CapturedPodUID)
		})
	}
}
