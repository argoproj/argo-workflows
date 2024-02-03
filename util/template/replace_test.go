package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func toJsonString(v interface{}) string {
	jsonString, _ := json.Marshal(v)
	return string(jsonString)
}

func Test_Replace(t *testing.T) {
	t.Run("InvalidTemplate", func(t *testing.T) {
		_, err := Replace(toJsonString("{{"), nil, false)
		assert.Error(t, err)
	})
	t.Run("Simple", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace(toJsonString("{{foo}}"), map[string]string{"foo": "bar"}, false)
			assert.NoError(t, err)
			assert.Equal(t, toJsonString("bar"), r)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace(toJsonString("{{foo}}"), nil, true)
				assert.NoError(t, err)
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace(toJsonString("{{foo}}"), nil, false)
				assert.EqualError(t, err, "failed to resolve {{foo}}")
			})
		})
	})
	t.Run("Expression", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace(toJsonString("{{=foo}}"), map[string]string{"foo": "bar"}, false)
			assert.NoError(t, err)
			assert.Equal(t, toJsonString("bar"), r)
		})
		t.Run("Valid WorkflowStatus", func(t *testing.T) {
			replaced, err := Replace(toJsonString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.status": "Succeeded"}, false)
			assert.NoError(t, err)
			assert.Equal(t, toJsonString(`SUCCESSFUL`), replaced)
			replaced, err = Replace(toJsonString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.status": "Failed"}, false)
			assert.NoError(t, err)
			assert.Equal(t, toJsonString(`FAILED`), replaced)
		})
		t.Run("Valid WorkflowFailures", func(t *testing.T) {
			replaced, err := Replace(toJsonString(`{{=workflow.failures == "{\"foo\":\"bar\"}" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.failures": `{"foo":"bar"}`}, false)
			assert.NoError(t, err)
			assert.Equal(t, toJsonString(`SUCCESSFUL`), replaced)
			replaced, err = Replace(toJsonString(`{{=workflow.failures == "{\"foo\":\"bar\"}" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.failures": `{"foo":"barr"}`}, false)
			assert.NoError(t, err)
			assert.Equal(t, toJsonString(`FAILED`), replaced)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace(toJsonString("{{=foo}}"), nil, true)
				assert.NoError(t, err)
			})
			t.Run("AllowedRetries", func(t *testing.T) {
				replaced, err := Replace(toJsonString("{{=sprig.int(retries)}}"), nil, true)
				assert.NoError(t, err)
				assert.Equal(t, toJsonString("{{=sprig.int(retries)}}"), replaced)
			})
			t.Run("AllowedWorkflowStatus", func(t *testing.T) {
				replaced, err := Replace(toJsonString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, true)
				assert.NoError(t, err)
				assert.Equal(t, toJsonString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), replaced)
			})
			t.Run("AllowedWorkflowFailures", func(t *testing.T) {
				replaced, err := Replace(toJsonString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, true)
				assert.NoError(t, err)
				assert.Equal(t, toJsonString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), replaced)
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace(toJsonString("{{=foo}}"), nil, false)
				assert.EqualError(t, err, "failed to evaluate expression: unknown name foo (1:1)\n | foo\n | ^")
			})
			t.Run("DisallowedWorkflowStatus", func(t *testing.T) {
				_, err := Replace(toJsonString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, false)
				assert.ErrorContains(t, err, "failed to evaluate expression")
			})
			t.Run("DisallowedWorkflowFailures", func(t *testing.T) {
				_, err := Replace(toJsonString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, false)
				assert.ErrorContains(t, err, "failed to evaluate expression")
			})
		})
		t.Run("Error", func(t *testing.T) {
			_, err := Replace(toJsonString("{{=!}}"), nil, false)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "failed to evaluate expression")
			}
		})
	})
}

func TestNestedReplaceString(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := toJsonString(`{{- with secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`)
	replacement, err := Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("{{- with secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)
	}

	test = toJsonString(`{{- with {{ secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("{{- with {{ secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)
	}

	test = toJsonString(`{{- with {{ secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("{{- with {{ secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)
	}

	test = toJsonString(`{{- with secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("{{- with secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)
	}

	test = toJsonString(`{{- with {{ {{ }} secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("{{- with {{ {{ }} secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)
	}

	test = toJsonString(`{{- with {{ {{ }} secret "{{does-not-exist}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, test, replacement)
	}
}

func TestReplaceStringWithWhiteSpace(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := toJsonString(`{{ inputs.parameters.message }}`)
	replacement, err := Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("hello world"), replacement)
	}
}

func TestReplaceStringWithExpression(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := toJsonString(`test {{= sprig.trunc(5, inputs.parameters.message) }}`)
	replacement, err := Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("test hello"), replacement)
	}

	test = toJsonString(`test {{= sprig.trunc(-5, inputs.parameters.message) }}`)
	replacement, err = Replace(test, replaceMap, true)
	if assert.NoError(t, err) {
		assert.Equal(t, toJsonString("test world"), replacement)
	}
}
