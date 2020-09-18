package prediction

import (
	"testing"

	"gotest.tools/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNullDurationPredictor(t *testing.T) {
	assert.Equal(t, wfv1.EstimatedDuration(0), NullDurationPredictor.EstimateWorkflowDuration())
	assert.Equal(t, wfv1.EstimatedDuration(0), NullDurationPredictor.EstimateNodeDuration(""))
}
