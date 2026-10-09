package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

func TestSynchronizationExistenceCheckReleasesOnlyCompletedExecution(t *testing.T) {
	for _, tc := range []struct {
		name       string
		phase      any
		completed  string
		artifactGC bool
		release    bool
	}{
		{name: "running", phase: string(wfv1.WorkflowRunning), completed: "false"},
		{name: "running-with-completed-label", phase: string(wfv1.WorkflowRunning), completed: "true"},
		{name: "error-without-completed-label", phase: string(wfv1.WorkflowError), completed: "false"},
		{name: "malformed-phase", phase: int64(1), completed: "true"},
		{name: "missing-phase", completed: "true"},
		{name: "completed-error-artifact-gc", phase: string(wfv1.WorkflowError), completed: "true", artifactGC: true, release: true},
		{name: "completed-success", phase: string(wfv1.WorkflowSucceeded), completed: "true", release: true},
		{name: "completed-success-artifact-gc", phase: string(wfv1.WorkflowSucceeded), completed: "true", artifactGC: true, release: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx)
			defer cancel()
			wf, _ := captureWriterFixture()
			wf.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "stale-completion"}}}
			wf.Labels = map[string]string{common.LabelKeyCompleted: tc.completed}
			if tc.artifactGC {
				wf.Finalizers = []string{common.FinalizerArtifactGC}
			}
			cached, err := util.ToUnstructured(wf)
			require.NoError(t, err)
			if tc.phase == nil {
				unstructured.RemoveNestedField(cached.Object, "status", "phase")
			} else {
				require.NoError(t, unstructured.SetNestedField(cached.Object, tc.phase, "status", "phase"))
			}
			// Populate the cache without an event. A stale hold is acquired
			// afterwards, so only the periodic existence check can release it.
			require.NoError(t, controller.wfInformer.GetIndexer().Add(cached))
			acquired, _, _, _, err := controller.syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired)
			contender := wf.DeepCopy()
			contender.Name, contender.UID = "contender", "contender-uid"
			contender.Status = wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning}
			contender.Labels = map[string]string{common.LabelKeyCompleted: "false"}
			pending, err := util.ToUnstructured(contender)
			require.NoError(t, err)
			require.NoError(t, controller.wfInformer.GetIndexer().Add(pending))
			acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
			require.NoError(t, err)
			require.False(t, acquired)

			controller.syncManager.CheckWorkflowExistence(ctx)
			acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
			require.NoError(t, err)
			assert.Equal(t, tc.release, acquired, "artifact GC does not extend the execution lock; malformed or incomplete completion evidence retains it")
		})
	}
}

func TestSynchronizationCompletionEventRechecksCurrentWorkflow(t *testing.T) {
	for _, tc := range []struct {
		name    string
		newUID  bool
		running bool
		release bool
	}{
		{name: "replacement-workflow", newUID: true, running: true},
		{name: "same-uid-retried-workflow", running: true},
		{name: "current-completed-workflow", release: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx)
			defer cancel()
			wf, _ := captureWriterFixture()
			wf.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "late-completion"}}}
			if tc.newUID {
				wf.UID = "replacement-uid"
			}
			if tc.running {
				wf.Status.Phase = wfv1.WorkflowRunning
				wf.Labels = map[string]string{common.LabelKeyCompleted: "false"}
			} else {
				wf.Status.Phase = wfv1.WorkflowError
				wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			}
			acquired, _, _, _, err := controller.syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired)
			current, err := util.ToUnstructured(wf)
			require.NoError(t, err)
			require.NoError(t, controller.wfInformer.GetIndexer().Add(current))
			contender := wf.DeepCopy()
			contender.Name, contender.UID = "contender", "contender-uid"
			contender.Status = wfv1.WorkflowStatus{}
			acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
			require.NoError(t, err)
			require.False(t, acquired)

			// A separate listener can still be delivering an older completion
			// after the worker has acquired the same named lock for a retry or
			// replacement. Its event alone must not release the current hold.
			previous := wf.DeepCopy()
			previous.UID = "workflow-uid"
			previous.Status.Phase = wfv1.WorkflowError
			previous.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			event, err := util.ToUnstructured(previous)
			require.NoError(t, err)
			controller.releaseCompletedWorkflowLocks(ctx, event)
			acquired, _, _, _, err = controller.syncManager.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
			require.NoError(t, err)
			assert.Equal(t, tc.release, acquired)
		})
	}
}
