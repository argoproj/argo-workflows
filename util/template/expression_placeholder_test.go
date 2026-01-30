package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestExpressionReplaceCore_PlaceholderBehavior(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Setup: A variable 'foo' holding an internal placeholder
	env := map[string]interface{}{
		"foo": "__argo__internal__placeholder-1",
	}
	expression := "foo"

	t.Run("AllowUnresolved=true", func(t *testing.T) {
		// New Core Logic
		var bCore strings.Builder
		_, errCore := expressionReplaceCore(ctx, &bCore, expression, env, true)

		// Old Helper Logic
		var bHelper strings.Builder
		errHelper := expressionReplaceHelper(ctx, &bHelper, expression, env, true)

		// Assertions
		// With allowUnresolved=true, we might expect it to return the placeholder string literally,
		// OR if the expression engine sees it as a valid string, it just returns it.
		// The 'expressionReplaceCore' generally returns the evaluated result.
		// Since "foo" resolves to the string "__argo__internal__placeholder-1",
		// and that string is not 'nil' or an error, it should just be written out.

		t.Logf("Core Result: %q, Error: %v", bCore.String(), errCore)
		t.Logf("Helper Result: %q, Error: %v", bHelper.String(), errHelper)

		assert.Equal(t, "__argo__internal__placeholder-1", bCore.String())
		require.NoError(t, errCore)

		assert.Equal(t, "__argo__internal__placeholder-1", bHelper.String())
		assert.NoError(t, errHelper)
	})

	t.Run("AllowUnresolved=false", func(t *testing.T) {
		// New Core Logic
		var bCore strings.Builder
		_, errCore := expressionReplaceCore(ctx, &bCore, expression, env, false)

		// Old Helper Logic
		var bHelper strings.Builder
		errHelper := expressionReplaceHelper(ctx, &bHelper, expression, env, false)

		t.Logf("Core Result: %q, Error: %v", bCore.String(), errCore)
		t.Logf("Helper Result: %q, Error: %v", bHelper.String(), errHelper)

		// Both should succeed and print the placeholder string,
		// because resolving "foo" to a string is a valid operation.
		assert.Equal(t, "__argo__internal__placeholder-1", bCore.String())
		require.NoError(t, errCore)

		assert.Equal(t, "__argo__internal__placeholder-1", bHelper.String())
		assert.NoError(t, errHelper)
	})
}
