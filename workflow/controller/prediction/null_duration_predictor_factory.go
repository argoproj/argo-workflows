package prediction

import wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

type nullDurationPredictorFactory struct{}

func (n nullDurationPredictorFactory) NewDurationPredictor(wf *wfv1.Workflow) (*DurationPredictor, error) {
	return NullDurationPredictor, nil
}

var NullDurationPredictorFactory DurationPredictorFactory = nullDurationPredictorFactory{}
