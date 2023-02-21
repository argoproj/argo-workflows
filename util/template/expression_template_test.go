package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_hasRetries(t *testing.T) {
	t.Run("hasRetiresInExpression", func(t *testing.T) {
		assert.True(t, hasRetries("retries"))
		assert.True(t, hasRetries("retries + 1"))
		assert.True(t, hasRetries("sprig(retries)"))
		assert.True(t, hasRetries("sprig(retries + 1) * 64"))
		assert.False(t, hasRetries("foo"))
		assert.False(t, hasRetries("retriesCustom + 1"))
	})
}

func Test_hasWorkflowParameters(t *testing.T) {
	t.Run("hasWorkflowStatusInExpression", func(t *testing.T) {
		assert.True(t, hasWorkflowStatus("workflow.status"))
		assert.True(t, hasWorkflowStatus(`workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		assert.False(t, hasWorkflowStatus(`"workflow.status" == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		assert.False(t, hasWorkflowStatus("workflow status"))
		assert.False(t, hasWorkflowStatus("workflow .status"))
	})

	t.Run("hasWorkflowFailuresInExpression", func(t *testing.T) {
		assert.True(t, hasWorkflowFailures("workflow.failures"))
		assert.True(t, hasWorkflowFailures(`workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		assert.False(t, hasWorkflowFailures(`"workflow.failures" == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
		assert.False(t, hasWorkflowFailures("workflow failures"))
		assert.False(t, hasWorkflowFailures("workflow .failures"))
	})
}
