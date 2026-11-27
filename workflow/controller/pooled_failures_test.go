package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
)

// TestWorkflowFailuresNamePooledNode checks that `{{workflow.failures}}` gives
// a failed pooled node its task name, and as its pod the claimant worker pod,
// or no pod when no worker claimed it.
func TestWorkflowFailuresNamePooledNode(t *testing.T) {
	tests := []struct {
		name        string
		state       wfv1.PooledTaskState
		wantPodName string
	}{
		{name: "claimed", state: wfv1.PooledTaskClaimed, wantPodName: "work-0123456789-abcde"},
		{name: "interrupted", state: wfv1.PooledTaskInterrupted, wantPodName: "work-0123456789-abcde"},
		{name: "never claimed", state: wfv1.PooledTaskEnqueued},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow(pooledNodesWf)
			wf.Status.Phase = wfv1.WorkflowRunning
			ledger := pooledTask(tt.state)
			now := metav1.Now()
			root := wfv1.NodeStatus{
				ID:           wf.NodeID(wf.Name),
				Name:         wf.Name,
				DisplayName:  wf.Name,
				Type:         wfv1.NodeTypeSteps,
				TemplateName: "main",
				Phase:        wfv1.NodeFailed,
				StartedAt:    now,
				FinishedAt:   now,
			}
			child := wfv1.NodeStatus{
				ID:               wf.NodeID(pooledChild),
				Name:             pooledChild,
				DisplayName:      "a",
				Type:             wfv1.NodeTypePod,
				TemplateName:     "work",
				BoundaryID:       root.ID,
				Phase:            wfv1.NodeFailed,
				Message:          "failed",
				StartedAt:        now,
				FinishedAt:       now,
				TaskResultSynced: new(true),
				PooledTask:       ledger,
			}
			root.Children = []string{child.ID}
			wf.Status.Nodes = wfv1.Nodes{root.ID: root, child.ID: child}
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)

			woc.operate(ctx)

			failures := woc.globalParams()[varkeys.WorkflowFailures.Template()]
			assert.Contains(t, failures, `\"displayName\":\"a\",\"message\":\"failed\",\"templateName\":\"work\",\"phase\":\"Failed\",\"podName\":\"`+tt.wantPodName+`\",\"taskName\":\"task\"`)
		})
	}
}
