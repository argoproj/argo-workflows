package estimation

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type dummyEstimatorFactory struct{}

func (d dummyEstimatorFactory) NewEstimator(*wfv1.Workflow) (Estimator, error) {
	return &dummyEstimator{}, nil
}

var DummyEstimatorFactory EstimatorFactory = &dummyEstimatorFactory{}
