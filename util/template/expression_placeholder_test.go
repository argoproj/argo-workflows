package template

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionReplaceCore_PlaceholderBehavior(t *testing.T) {
	env := map[string]any{
		"foo": "__argo__internal__placeholder-1",
	}
	expression := "foo"

	for _, allowUnresolved := range []bool{true, false} {
		t.Run(fmt.Sprintf("AllowUnresolved=%v", allowUnresolved), func(t *testing.T) {
			var b strings.Builder
			_, err := expressionReplaceCore(&b, expression, env, allowUnresolved)

			t.Logf("Result: %q, Error: %v", b.String(), err)
			assert.Equal(t, "__argo__internal__placeholder-1", b.String())
			require.NoError(t, err)
		})
	}
}
