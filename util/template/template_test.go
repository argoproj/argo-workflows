package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Template_Replace(t *testing.T) {
	t.Run("ExpressionWithEscapedCharacters", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			template, err := NewTemplate("{{='test'}}")
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			assert.Equal(t, "test", r)
		})
		t.Run("Valid", func(t *testing.T) {
			template, err := NewTemplate(`{{=\"test\"}}`)
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			assert.Equal(t, "test", r)
		})
		t.Run("Valid", func(t *testing.T) {
			template, err := NewTemplate(`{{='some\\path\\with\\backslashes'}}`)
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			assert.Equal(t, `some\path\with\backslashes`, r)
		})
	})
}
