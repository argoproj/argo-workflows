package estimation

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// DurationEstimator return estimations for how long workflows and nodes will take
type DurationEstimator struct {
	wf         *wfv1.Workflow
	baselineWF *wfv1.Workflow
}

func (woc DurationEstimator) EstimateWorkflowDuration() wfv1.EstimatedDuration {
	if woc.baselineWF == nil {
		return 0
	}
	return wfv1.NewEstimatedDuration(woc.baselineWF.Status.GetDuration())
}

func (woc DurationEstimator) EstimateNodeDuration(nodeName string) wfv1.EstimatedDuration {
	if woc.baselineWF == nil {
		return 0
	}
	// special case for root node
	if nodeName == woc.wf.Name {
		nodeName = woc.baselineWF.Name
	}
	oldNodeID := woc.baselineWF.NodeID(nodeName)
	return wfv1.NewEstimatedDuration(woc.baselineWF.Status.Nodes[oldNodeID].GetDuration())
}
