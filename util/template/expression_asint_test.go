package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestExpressionReplaceCore_AsIntPlaceholder(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	e := env.GetFuncMap(map[string]any{
		"foo": "__argo__internal__placeholder-1",
	})

	expression := "asInt(foo)"

	t.Run("AllowUnresolved=true", func(t *testing.T) {
		var b strings.Builder
		_, err := expressionReplaceCore(ctx, &b, expression, e, true)

		t.Logf("Result: %q, Error: %v", b.String(), err)
		require.NoError(t, err)
		assert.Equal(t, "{{=asInt(foo)}}", b.String())
	})

	t.Run("AllowUnresolved=false", func(t *testing.T) {
		var b strings.Builder
		_, err := expressionReplaceCore(ctx, &b, expression, e, false)

		t.Logf("Result: %q, Error: %v", b.String(), err)
		// Even with allowUnresolved=false, placeholders cause it to allow unresolved.
		require.NoError(t, err)
		assert.Equal(t, "{{=asInt(foo)}}", b.String())
	})
}
