package template

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/expr/env"
)

func TestExpressionReplaceCore_AsIntPlaceholder(t *testing.T) {
	e := env.GetFuncMap(map[string]any{
		"foo": "__argo__internal__placeholder-1",
	})

	expression := "asInt(foo)"

	for _, allowUnresolved := range []bool{true, false} {
		t.Run(fmt.Sprintf("AllowUnresolved=%v", allowUnresolved), func(t *testing.T) {
			var b strings.Builder
			_, err := expressionReplaceCore(&b, expression, e, allowUnresolved)

			t.Logf("Result: %q, Error: %v", b.String(), err)
			// Even with allowUnresolved=false, placeholders cause it to allow unresolved.
			require.NoError(t, err)
			assert.Equal(t, "{{=asInt(foo)}}", b.String())
		})
	}
}
