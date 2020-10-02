package estimation

import (
	wfv1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
)

// Estimator return estimations for how long workflows and nodes will take
type Estimator interface {
	EstimateWorkflowDuration() wfv1.EstimatedDuration
	EstimateNodeDuration(nodeName string) wfv1.EstimatedDuration
}

type estimator struct {
	wf         *wfv1.Workflow
	baselineWF *wfv1.Workflow
}

func (e *estimator) EstimateWorkflowDuration() wfv1.EstimatedDuration {
	if e.baselineWF == nil {
		return 0
	}
	return wfv1.NewEstimatedDuration(e.baselineWF.Status.GetDuration())
}

func (e *estimator) EstimateNodeDuration(nodeName string) wfv1.EstimatedDuration {
	if e.baselineWF == nil {
		return 0
	}
	// special case for root node
	if nodeName == e.wf.Name {
		nodeName = e.baselineWF.Name
	}
	oldNodeID := e.baselineWF.NodeID(nodeName)
	return wfv1.NewEstimatedDuration(e.baselineWF.Status.Nodes[oldNodeID].GetDuration())
}
