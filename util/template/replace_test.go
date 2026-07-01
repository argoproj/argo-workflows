package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toJSONString(v interface{}) string {
	jsonString, _ := json.Marshal(v)
	return string(jsonString)
}

func Test_Replace(t *testing.T) {
	t.Run("InvalidTemplate", func(t *testing.T) {
		_, err := Replace(toJSONString("{{"), nil, false)
		require.Error(t, err)
	})
	t.Run("Simple", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace(toJSONString("{{foo}}"), map[string]string{"foo": "bar"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString("bar"), r)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace(toJSONString("{{foo}}"), nil, true)
				require.NoError(t, err)
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace(toJSONString("{{foo}}"), nil, false)
				require.EqualError(t, err, "failed to resolve {{foo}}")
			})
		})
	})
	t.Run("Expression", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace(toJSONString("{{=foo}}"), map[string]string{"foo": "bar"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString("bar"), r)
		})
		t.Run("Valid With Variadic Sprig Expression", func(t *testing.T) {
			r, err := Replace(toJSONString("{{=sprig.dig('status', nil, workflow)}}"), map[string]string{"workflow.status": "Succeeded"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString("Succeeded"), r)
		})
		t.Run("Valid WorkflowStatus", func(t *testing.T) {
			replaced, err := Replace(toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.status": "Succeeded"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`SUCCESSFUL`), replaced)
			replaced, err = Replace(toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.status": "Failed"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`FAILED`), replaced)
		})
		t.Run("Valid WorkflowFailures", func(t *testing.T) {
			replaced, err := Replace(toJSONString(`{{=workflow.failures == "{\"foo\":\"bar\"}" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.failures": `{"foo":"bar"}`}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`SUCCESSFUL`), replaced)
			replaced, err = Replace(toJSONString(`{{=workflow.failures == "{\"foo\":\"bar\"}" ? "SUCCESSFUL" : "FAILED"}}`), map[string]string{"workflow.failures": `{"foo":"barr"}`}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`FAILED`), replaced)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace(toJSONString("{{=foo}}"), nil, true)
				require.NoError(t, err)
			})
			t.Run("AllowedRetries", func(t *testing.T) {
				replaced, err := Replace(toJSONString("{{=sprig.int(retries)}}"), nil, true)
				require.NoError(t, err)
				assert.Equal(t, toJSONString("{{=sprig.int(retries)}}"), replaced)
			})
			t.Run("AllowedWorkflowStatus", func(t *testing.T) {
				replaced, err := Replace(toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, true)
				require.NoError(t, err)
				assert.Equal(t, toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), replaced)
			})
			t.Run("AllowedWorkflowFailures", func(t *testing.T) {
				replaced, err := Replace(toJSONString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, true)
				require.NoError(t, err)
				assert.Equal(t, toJSONString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), replaced)
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace(toJSONString("{{=foo}}"), nil, false)
				require.EqualError(t, err, "failed to evaluate expression: foo is missing")
			})
			t.Run("DisallowedWorkflowStatus", func(t *testing.T) {
				_, err := Replace(toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, false)
				require.ErrorContains(t, err, "failed to evaluate expression")
			})
			t.Run("DisallowedWorkflowFailures", func(t *testing.T) {
				_, err := Replace(toJSONString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, false)
				require.ErrorContains(t, err, "failed to evaluate expression")
			})
		})
		t.Run("Error", func(t *testing.T) {
			_, err := Replace(toJSONString("{{=!}}"), nil, false)
			require.ErrorContains(t, err, "failed to evaluate expression")
		})
	})
}

func Test_ReplaceStrict_NilCoalescing(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	itemPresent := map[string]any{"item": map[string]any{"name": "a", "optionalKey": "value"}}
	itemMissing := map[string]any{"item": map[string]any{"name": "b"}}

	tests := []struct {
		name        string
		expression  string
		wantPresent string
		wantMissing string
	}{
		{
			name:        "nil-coalescing",
			expression:  `{{= item.optionalKey ?? 'fallback' }}`,
			wantPresent: "value",
			wantMissing: "fallback",
		},
		{
			name:        "nil-coalescing with bracket notation",
			expression:  `{{= item['optionalKey'] ?? 'fallback' }}`,
			wantPresent: "value",
			wantMissing: "fallback",
		},
		{
			name:        "optional chaining with nil-coalescing",
			expression:  `{{= item?.optionalKey ?? 'fallback' }}`,
			wantPresent: "value",
			wantMissing: "fallback",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := NewTemplate(tt.expression)
			require.NoError(t, err)

			out, err := tmpl.ReplaceStrict(ctx, itemPresent, []string{"item"})
			require.NoError(t, err)
			assert.Equal(t, tt.wantPresent, out)

			out, err = tmpl.ReplaceStrict(ctx, itemMissing, []string{"item"})
			require.NoError(t, err)
			assert.Equal(t, tt.wantMissing, out)
		})
	}

	t.Run("missing base variable still fails strict check", func(t *testing.T) {
		tmpl, err := NewTemplate(`{{= item.optionalKey ?? 'fallback' }}`)
		require.NoError(t, err)
		_, err = tmpl.ReplaceStrict(ctx, map[string]any{}, []string{"item"})
		require.EqualError(t, err, "failed to evaluate expression: item is missing")
	})
}

func TestNestedReplaceString(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := toJSONString(`{{- with secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`)
	replacement, err := Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with {{ secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with {{ secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ {{ }} secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with {{ {{ }} secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ {{ }} secret "{{does-not-exist}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, test, replacement)
}

func TestReplaceStringWithWhiteSpace(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := toJSONString(`{{ inputs.parameters.message }}`)
	replacement, err := Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("hello world"), replacement)
}

func TestReplaceStringWithExpression(t *testing.T) {
	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := toJSONString(`test {{= sprig.trunc(5, inputs.parameters.message) }}`)
	replacement, err := Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("test hello"), replacement)

	test = toJSONString(`test {{= sprig.trunc(-5, inputs.parameters.message) }}`)
	replacement, err = Replace(test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("test world"), replacement)
}
