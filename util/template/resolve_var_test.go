package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ResolveVar(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			v, err := ResolveVar("{{foo}}", map[string]interface{}{"foo": "bar"})
			assert.NoError(t, err)
			assert.Equal(t, "bar", v)
		})
		t.Run("Whitespace", func(t *testing.T) {
			v, err := ResolveVar("{{ foo }}", map[string]interface{}{"foo": "bar"})
			assert.NoError(t, err)
			assert.Equal(t, "bar", v)
		})
		t.Run("Unresolved", func(t *testing.T) {
			_, err := ResolveVar("{{foo}}", nil)
			assert.EqualError(t, err, "Unable to resolve: \"foo\"")
		})
	})
	t.Run("Expression", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			v, err := ResolveVar("{{=foo}}", map[string]interface{}{"foo": "bar"})
			assert.NoError(t, err)
			assert.Equal(t, "bar", v)
		})
		t.Run("Unresolved", func(t *testing.T) {
			_, err := ResolveVar("{{=foo}}", nil)
			assert.EqualError(t, err, "Unable to resolve: \"=foo\"")
		})
		t.Run("Error", func(t *testing.T) {
			_, err := ResolveVar("{{=!}}", nil)
			assert.EqualError(t, err, "Invalid expression: \"!\"")
		})
	})
}
