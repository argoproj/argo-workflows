package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// A failed downstream publish is connected to a completed nested DAG through
// its outbound loop item. Retrying publish must not reopen that predecessor.
func TestRetryPreservesCompletedPredecessorBoundary(t *testing.T) {
	for _, leafPhase := range []wfv1.NodePhase{wfv1.NodeSucceeded, wfv1.NodeSkipped} {
		t.Run(string(leafPhase), func(t *testing.T) {
			leafType := wfv1.NodeTypePod
			if leafPhase == wfv1.NodeSkipped {
				leafType = wfv1.NodeTypeSkipped
			}
			wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "test", Labels: map[string]string{}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowFailed, Nodes: wfv1.Nodes{
				"wf":          {ID: "wf", Name: "wf", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeFailed, Children: []string{"media"}},
				"media":       {ID: "media", Name: "wf.media", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeSucceeded, BoundaryID: "wf", Children: []string{"loop"}, OutboundNodes: []string{"item"}},
				"loop":        {ID: "loop", Name: "wf.media.loop", Type: wfv1.NodeTypeTaskGroup, Phase: wfv1.NodeSucceeded, BoundaryID: "media", Children: []string{"item"}},
				"item":        {ID: "item", Name: "wf.media.loop(0)", Type: leafType, Phase: leafPhase, BoundaryID: "media", Children: []string{"publish"}},
				"publish":     {ID: "publish", Name: "wf.publish", Type: wfv1.NodeTypeRetry, Phase: wfv1.NodeFailed, BoundaryID: "wf", Children: []string{"publish-pod"}},
				"publish-pod": {ID: "publish-pod", Name: "wf.publish(0)", Type: wfv1.NodeTypePod, Phase: wfv1.NodeFailed, BoundaryID: "wf"},
			}}}
			completedAt := metav1.Now()
			media := wf.Status.Nodes["media"]
			media.FinishedAt = completedAt
			media.Outputs = &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "artifact", Value: wfv1.AnyStringPtr("existing-result")}}}
			wf.Status.Nodes["media"] = media
			retried, deleted, err := FormulateRetryWorkflow(logging.TestContext(t.Context()), wf, false, "", nil)
			require.NoError(t, err)
			require.Len(t, deleted, 1)
			require.Equal(t, wfv1.NodeRunning, retried.Status.Nodes["wf"].Phase)
			for _, id := range []string{"media", "loop", "item"} {
				require.Equal(t, wf.Status.Nodes[id].Phase, retried.Status.Nodes[id].Phase, "predecessor %s must not reset", id)
			}
			require.Equal(t, completedAt, retried.Status.Nodes["media"].FinishedAt)
			require.Equal(t, media.Outputs, retried.Status.Nodes["media"].Outputs)
			require.NotContains(t, retried.Status.Nodes, "publish-pod")
		})
	}
}

// Retrying inside media must still invalidate publish, even when publish had
// succeeded. Preserving predecessors must not preserve stale downstream work.
func TestRetryInsideNestedBoundaryInvalidatesDownstream(t *testing.T) {
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "test", Labels: map[string]string{}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded, Nodes: wfv1.Nodes{
		"wf":          {ID: "wf", Name: "wf", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeSucceeded, Children: []string{"media"}},
		"media":       {ID: "media", Name: "wf.media", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeSucceeded, BoundaryID: "wf", Children: []string{"loop"}},
		"loop":        {ID: "loop", Name: "wf.media.loop", Type: wfv1.NodeTypeTaskGroup, Phase: wfv1.NodeSucceeded, BoundaryID: "media", Children: []string{"item"}},
		"item":        {ID: "item", Name: "wf.media.loop(0)", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, BoundaryID: "media", Children: []string{"publish"}},
		"publish":     {ID: "publish", Name: "wf.publish", Type: wfv1.NodeTypeRetry, Phase: wfv1.NodeSucceeded, BoundaryID: "wf", Children: []string{"publish-pod"}},
		"publish-pod": {ID: "publish-pod", Name: "wf.publish(0)", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, BoundaryID: "wf"},
	}}}
	retried, deleted, err := FormulateRetryWorkflow(logging.TestContext(t.Context()), wf, true, "id=item", nil)
	require.NoError(t, err)
	require.Len(t, deleted, 2)
	for _, id := range []string{"wf", "media", "loop"} {
		require.Equal(t, wfv1.NodeRunning, retried.Status.Nodes[id].Phase, id)
	}
	for _, id := range []string{"item", "publish", "publish-pod"} {
		require.NotContains(t, retried.Status.Nodes, id)
	}
}
