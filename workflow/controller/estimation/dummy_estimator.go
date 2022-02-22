package estimation

import (
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type dummyEstimator struct{}

func (e *dummyEstimator) EstimateWorkflowDuration() wfv1.EstimatedDuration {
	return wfv1.NewEstimatedDuration(time.Second)
}

func (e *dummyEstimator) EstimateNodeDuration(string) wfv1.EstimatedDuration {
	return wfv1.NewEstimatedDuration(time.Second)
}
