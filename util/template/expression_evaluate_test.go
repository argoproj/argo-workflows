package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestEvaluateExpression(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("ResolvedPreservesType", func(t *testing.T) {
		value, resolved, err := EvaluateExpression(ctx, "1 + 2", nil, true)
		require.NoError(t, err)
		assert.True(t, resolved)
		assert.Equal(t, 3, value)
	})

	t.Run("ResolvedUsesArgoEnvironment", func(t *testing.T) {
		value, resolved, err := EvaluateExpression(ctx, "sprig.upper(workflow.parameters.message)", map[string]any{
			"workflow": map[string]any{
				"parameters": map[string]any{"message": "hello"},
			},
		}, true)
		require.NoError(t, err)
		assert.True(t, resolved)
		assert.Equal(t, "HELLO", value)
	})

	t.Run("AllowedUnresolved", func(t *testing.T) {
		value, resolved, err := EvaluateExpression(ctx, "workflow.parameters.missing", nil, true)
		require.NoError(t, err)
		assert.False(t, resolved)
		assert.Nil(t, value)
	})

	t.Run("InvalidSyntax", func(t *testing.T) {
		_, _, err := EvaluateExpression(ctx, "workflow.parameters.", nil, true)
		require.ErrorContains(t, err, "failed to evaluate expression")
	})
}
