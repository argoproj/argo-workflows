package estimation

import (
	"strings"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	oldNodeID := e.baselineWF.NodeID(strings.Replace(nodeName, e.wf.Name, e.baselineWF.Name, 1))
	node, err := e.baselineWF.Status.Nodes.Get(oldNodeID)
	if err != nil {
		log.Errorf("was unable to obtain node for %s", oldNodeID)
		// inacurate but not going to break anything
		return 0
	}
	return wfv1.NewEstimatedDuration(node.GetDuration())
}
