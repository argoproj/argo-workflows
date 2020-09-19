package estimation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNullDurationEstimator(t *testing.T) {
	assert.Equal(t, wfv1.EstimatedDuration(0), NullDurationEstimator.EstimateWorkflowDuration())
	assert.Equal(t, wfv1.EstimatedDuration(0), NullDurationEstimator.EstimateNodeDuration(""))
}
