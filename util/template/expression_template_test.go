package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_hasRetries(t *testing.T) {
	t.Run("hasRetiresInExpression", func(t *testing.T) {
		require.True(t, hasRetries("retries"))
		require.True(t, hasRetries("retries + 1"))
		require.True(t, hasRetries("sprig(retries)"))
		require.True(t, hasRetries("sprig(retries + 1) * 64"))
		require.False(t, hasRetries("foo"))
		require.False(t, hasRetries("retriesCustom + 1"))
	})
}

func Test_hasWorkflowParameters(t *testing.T) {
	t.Run("hasWorkflowStatusInExpression", func(t *testing.T) {
		require.True(t, hasWorkflowStatus("workflow.status"))
		require.True(t, hasWorkflowStatus(`workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		require.False(t, hasWorkflowStatus(`"workflow.status" == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		require.False(t, hasWorkflowStatus("workflow status"))
		require.False(t, hasWorkflowStatus("workflow .status"))
	})

	t.Run("hasWorkflowFailuresInExpression", func(t *testing.T) {
		require.True(t, hasWorkflowFailures("workflow.failures"))
		require.True(t, hasWorkflowFailures(`workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		require.False(t, hasWorkflowFailures(`"workflow.failures" == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		require.False(t, hasWorkflowFailures("workflow failures"))
		require.False(t, hasWorkflowFailures("workflow .failures"))
	})
}
