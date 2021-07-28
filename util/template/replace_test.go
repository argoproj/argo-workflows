package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Replace(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			obj := "{{foo}}"
			err := Replace(&obj, map[string]string{"foo": "bar"}, false)
			assert.NoError(t, err)
			assert.Equal(t, "bar", obj)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				obj := "{{foo}}"
				err := Replace(&obj, nil, true)
				assert.NoError(t, err)
			})
			t.Run("Disallowed", func(t *testing.T) {
				obj := "{{foo}}"
				err := Replace(&obj, nil, false)
				assert.EqualError(t, err, "failed to resolve {{foo}}")
			})
		})
	})
	t.Run("Expression", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			obj := "{{=foo}}"
			err := Replace(&obj, map[string]string{"foo": "bar"}, false)
			assert.NoError(t, err)
			assert.Equal(t, "bar", obj)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				obj := "{{=foo}}"
				err := Replace(&obj, nil, true)
				assert.NoError(t, err)
			})
			t.Run("AllowedRetries", func(t *testing.T) {
				obj := "{{=sprig.int(retries)}}"
				err := Replace(&obj, nil, true)
				assert.NoError(t, err)
				assert.Equal(t, obj, "{{=sprig.int(retries)}}")
			})
			t.Run("Disallowed", func(t *testing.T) {
				obj := "{{=foo}}"
				err := Replace(&obj, nil, false)
				assert.EqualError(t, err, "failed to evaluate expression \"foo\"")
			})
		})
		t.Run("Error", func(t *testing.T) {
			obj := "{{=!}}"
			err := Replace(&obj, nil, false)
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
	err := Replace(&test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", test)
	}

	test = `{{- with {{ secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	err = Replace(&test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with {{ secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", test)
	}

	test = `{{- with {{ secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	err = Replace(&test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with {{ secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", test)
	}

	test = `{{- with secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	err = Replace(&test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", test)
	}

	test = `{{- with {{ {{ }} secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	err = Replace(&test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "{{- with {{ {{ }} secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", test)
	}

	test = `{{- with {{ {{ }} secret "{{does-not-exist}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`

	err = Replace(&test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, test, test)
	}
}

func TestReplaceStringWithWhiteSpace(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := `{{ inputs.parameters.message }}`
	err := Replace(&test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "hello world", test)
	}
}
