package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestExpressionReplaceCore_AsIntPlaceholder(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Get a base env with asInt
	e := env.GetFuncMap(map[string]any{
		"foo": "__argo__internal__placeholder-1",
	})

	expression := "asInt(foo)"

	t.Run("AllowUnresolved=true", func(t *testing.T) {
		var b strings.Builder
		_, err := expressionReplaceCore(ctx, &b, expression, e, true)

		t.Logf("Result: %q, Error: %v", b.String(), err)
		// Expected: asInt("...") fails, so it should return {{=asInt(foo)}}
		require.NoError(t, err)
		assert.Equal(t, "{{=asInt(foo)}}", b.String())
	})

	t.Run("AllowUnresolved=false", func(t *testing.T) {
		var b strings.Builder
		_, err := expressionReplaceCore(ctx, &b, expression, e, false)

		t.Logf("Core Result: %q, Error: %v", b.String(), err)

		// New behavior: even with allowUnresolved=false, if placeholders are present, it allows unresolved.
		require.NoError(t, err)
		assert.Equal(t, "{{=asInt(foo)}}", b.String())

		// Old behavior (Helper): fails because it doesn't check for placeholders
		var bHelper strings.Builder
		errHelper := expressionReplaceHelper(ctx, &bHelper, expression, e, false)
		t.Logf("Helper Result: %q, Error: %v", bHelper.String(), errHelper)

		require.Error(t, errHelper)
		assert.Contains(t, errHelper.Error(), "failed to evaluate expression")
	})
}
