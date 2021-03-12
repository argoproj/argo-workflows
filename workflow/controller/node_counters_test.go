package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func getWfOperationCtx() *wfOperationCtx {
	return &wfOperationCtx{
		wf: &v1alpha1.Workflow{
			Status: v1alpha1.WorkflowStatus{
				Nodes: map[string]v1alpha1.NodeStatus{
					"1":  {Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeSucceeded, BoundaryID: "1"},
					"2":  {Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed, BoundaryID: "1"},
					"3":  {Type: v1alpha1.NodeTypeSteps, Phase: v1alpha1.NodeFailed, BoundaryID: "1"},
					"4":  {Type: v1alpha1.NodeTypeDAG, Phase: v1alpha1.NodeError, BoundaryID: "1"},
					"5":  {Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeRunning, BoundaryID: "1"},
					"5a": {Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeRunning, BoundaryID: "1", SynchronizationStatus: &v1alpha1.NodeSynchronizationStatus{Waiting: "yes"}},
					"6":  {Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodePending, BoundaryID: "1"},
					"7":  {Type: v1alpha1.NodeTypeSteps, Phase: v1alpha1.NodeRunning, BoundaryID: "1"},
					"8":  {Type: v1alpha1.NodeTypeDAG, Phase: v1alpha1.NodePending, BoundaryID: "1"},

					"9":  {Type: v1alpha1.NodeTypeSteps, Phase: v1alpha1.NodeFailed, BoundaryID: "2"},
					"10": {Type: v1alpha1.NodeTypeDAG, Phase: v1alpha1.NodeError, BoundaryID: "2"},
					"11": {Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeRunning, BoundaryID: "2"},
					"12": {Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodePending, BoundaryID: "2"},
				},
			},
		},
	}
}

func TestCounters(t *testing.T) {
	woc := getWfOperationCtx()
	assert.Equal(t, int64(2), woc.getActivePods("1"))
	// No BoundaryID requested
	assert.Equal(t, int64(4), woc.getActivePods(""))
	assert.Equal(t, int64(5), woc.getActiveChildren("1"))
	assert.Equal(t, int64(3), woc.getUnsuccessfulChildren("1"))
	assert.Equal(t, int64(2), woc.getActivePods("2"))
	assert.Equal(t, int64(2), woc.getActiveChildren("2"))
	assert.Equal(t, int64(2), woc.getUnsuccessfulChildren("2"))
}
