package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ResolveVar(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			v, err := ResolveVar("{{foo}}", map[string]any{"foo": "bar"})
			require.NoError(t, err)
			assert.Equal(t, "bar", v)
		})
		t.Run("Whitespace", func(t *testing.T) {
			v, err := ResolveVar("{{ foo }}", map[string]any{"foo": "bar"})
			require.NoError(t, err)
			assert.Equal(t, "bar", v)
		})
		t.Run("Unresolved", func(t *testing.T) {
			_, err := ResolveVar("{{foo}}", nil)
			require.EqualError(t, err, "Unable to resolve: \"foo\"")
		})
	})
	t.Run("Expression", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			v, err := ResolveVar("{{=foo}}", map[string]any{"foo": "bar"})
			require.NoError(t, err)
			assert.Equal(t, "bar", v)
		})
		t.Run("Unresolved", func(t *testing.T) {
			_, err := ResolveVar("{{=foo}}", nil)
			require.EqualError(t, err, "Unable to compile: \"foo\"")
		})
		t.Run("Error", func(t *testing.T) {
			_, err := ResolveVar("{{=!}}", nil)
			require.EqualError(t, err, "Unable to compile: \"!\"")
		})
	})
}
