package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Template_Replace(t *testing.T) {
	t.Run("ExpressionWithEscapedCharacters", func(t *testing.T) {
		t.Run("SingleQuotes", func(t *testing.T) {
			template, err := NewTemplate("{{='test'}}")
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			assert.Equal(t, "test", r)
		})
		t.Run("DoubleQuotes", func(t *testing.T) {
			template, err := NewTemplate(`{{=\"test\"}}`)
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			assert.Equal(t, "test", r)
		})
		t.Run("EscapedBackslashes", func(t *testing.T) {
			// In YAML, this would look like {{='some\\path\\with\\backslashes'}}, making it valid expr.
			template, err := NewTemplate(`{{='some\\\\path\\\\with\\\\backslashes'}}`)
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			assert.Equal(t, `some\path\with\backslashes`, r)
		})
		t.Run("Newline", func(t *testing.T) {
			template, err := NewTemplate(`{{=1 +\n1}}`)
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			assert.Equal(t, "2", r)
		})
		t.Run("StringAsJson", func(t *testing.T) {
			template, err := NewTemplate(`{{=toJson('test')}}`)
			assert.NoError(t, err)
			r, err := template.Replace(map[string]string{}, true)
			assert.NoError(t, err)
			// Output should be escaped since it will be embedded in stringified JSON before unmarshaling.
			assert.Equal(t,`\"test\"`, r)
		})
	})
}
