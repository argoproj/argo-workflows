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

func TestHasVarInEnv(t *testing.T) {
	t.Run("parameterExistsInEnv", func(t *testing.T) {
		env := map[string]interface{}{
			"workflow": map[string]interface{}{
				"status": "Succeeded",
			},
		}
		assert.True(t, hasVarInEnv(env, "workflow.status"))
	})

	t.Run("parameterDoesNotExistInEnv", func(t *testing.T) {
		env := map[string]interface{}{
			"workflow": map[string]interface{}{
				"status": "Succeeded",
			},
		}
		assert.False(t, hasVarInEnv(env, "workflow.failures"))
	})

	t.Run("emptyEnv", func(t *testing.T) {
		env := map[string]interface{}{}
		assert.False(t, hasVarInEnv(env, "workflow.status"))
	})

	t.Run("nestedParameterExistsInEnv", func(t *testing.T) {
		env := map[string]interface{}{
			"workflow": map[string]interface{}{
				"details": map[string]interface{}{
					"status": "Succeeded",
				},
			},
		}
		assert.True(t, hasVarInEnv(env, "workflow.details.status"))
	})

	t.Run("nestedParameterDoesNotExistInEnv", func(t *testing.T) {
		env := map[string]interface{}{
			"workflow": map[string]interface{}{
				"details": map[string]interface{}{
					"status": "Succeeded",
				},
			},
		}
		assert.False(t, hasVarInEnv(env, "workflow.details.failures"))
	})
}
