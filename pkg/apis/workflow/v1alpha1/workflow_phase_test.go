package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowPhase_Completed(t *testing.T) {
	require.False(t, WorkflowUnknown.Completed())
	require.False(t, WorkflowPending.Completed())
	require.False(t, WorkflowRunning.Completed())
	require.True(t, WorkflowSucceeded.Completed())
	require.True(t, WorkflowFailed.Completed())
	require.True(t, WorkflowError.Completed())
}
