package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowPhase_Completed(t *testing.T) {
	assert.False(t, WorkflowUnknown.Completed())
	assert.False(t, WorkflowPending.Completed())
	assert.False(t, WorkflowRunning.Completed())
	assert.True(t, WorkflowSucceeded.Completed())
	assert.True(t, WorkflowFailed.Completed())
	assert.True(t, WorkflowError.Completed())
}
