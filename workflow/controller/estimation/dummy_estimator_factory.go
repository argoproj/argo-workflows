package estimation

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type dummyEstimatorFactory struct{}

func (d dummyEstimatorFactory) NewEstimator(context.Context, *wfv1.Workflow) (Estimator, error) {
	return &dummyEstimator{}, nil
}

var DummyEstimatorFactory EstimatorFactory = &dummyEstimatorFactory{}
