package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Replace(t *testing.T) {
	t.Run("InvailedTemplate", func(t *testing.T) {
		_, err := Replace("{{", nil, false)
		assert.Error(t, err)
	})
	t.Run("Simple", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace("{{foo}}", map[string]string{"foo": "bar"}, false)
			assert.NoError(t, err)
			assert.Equal(t, "bar", r)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace("{{foo}}", nil, true)
				assert.NoError(t, err)
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace("{{foo}}", nil, false)
				assert.EqualError(t, err, "failed to resolve {{foo}}")
			})
		})
	})
	t.Run("Expression", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace("{{=foo}}", map[string]string{"foo": "bar"}, false)
			assert.NoError(t, err)
			assert.Equal(t, "bar", r)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace("{{=foo}}", nil, true)
				assert.NoError(t, err)
			})
			t.Run("AllowedRetries", func(t *testing.T) {
				replaced, err := Replace("{{=sprig.int(retries)}}", nil, true)
				assert.NoError(t, err)
				assert.Equal(t, replaced, "{{=sprig.int(retries)}}")
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace("{{=foo}}", nil, false)
				assert.EqualError(t, err, "failed to evaluate expression \"foo\"")
			})
		})
		t.Run("Error", func(t *testing.T) {
			_, err := Replace("{{=!}}", nil, false)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "failed to evaluate expression")
			}
		})
	})
}

func TestNestedReplaceString(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := `{{- with secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`
	replacement, err := Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
	}

	test = `{{- with {{ secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with {{ secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
	}

	test = `{{- with {{ secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with {{ secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
	}

	test = `{{- with secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
	}

	test = `{{- with {{ {{ }} secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with {{ {{ }} secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
	}

	test = `{{- with {{ {{ }} secret "{{does-not-exist}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, test, replacement)
	}
}

func TestReplaceStringWithWhiteSpace(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := `{{ inputs.parameters.message }}`
	replacement, err := Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "hello world", replacement)
	}
}
