package estimation

import wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

type nullDurationEstimatorFactory struct{}

func (n nullDurationEstimatorFactory) NewDurationEstimator(wf *wfv1.Workflow) (*DurationEstimator, error) {
	return NullDurationEstimator, nil
}

var NullDurationEstimatorFactory DurationEstimatorFactory = nullDurationEstimatorFactory{}
