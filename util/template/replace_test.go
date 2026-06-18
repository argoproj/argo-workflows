package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func toJSONString(v interface{}) string {
	jsonString, _ := json.Marshal(v)
	return string(jsonString)
}

func Test_Replace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	t.Run("InvalidTemplate", func(t *testing.T) {
		_, err := Replace(ctx, toJSONString("{{"), nil, false)
		require.Error(t, err)
	})
	t.Run("Simple", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace(ctx, toJSONString("{{foo}}"), map[string]any{"foo": "bar"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString("bar"), r)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace(ctx, toJSONString("{{foo}}"), nil, true)
				require.NoError(t, err)
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace(ctx, toJSONString("{{foo}}"), nil, false)
				require.EqualError(t, err, "failed to resolve {{foo}}")
			})
		})
	})
	t.Run("Expression", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			r, err := Replace(ctx, toJSONString("{{=foo}}"), map[string]any{"foo": "bar"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString("bar"), r)
		})
		t.Run("Valid With Variadic Sprig Expression", func(t *testing.T) {
			r, err := Replace(ctx, toJSONString("{{=sprig.dig('status', nil, workflow)}}"), map[string]any{"workflow.status": "Succeeded"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString("Succeeded"), r)
		})
		t.Run("Valid WorkflowStatus", func(t *testing.T) {
			replaced, err := Replace(ctx, toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), map[string]any{"workflow.status": "Succeeded"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`SUCCESSFUL`), replaced)
			replaced, err = Replace(ctx, toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), map[string]any{"workflow.status": "Failed"}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`FAILED`), replaced)
		})
		t.Run("Valid WorkflowFailures", func(t *testing.T) {
			replaced, err := Replace(ctx, toJSONString(`{{=workflow.failures == "{\"foo\":\"bar\"}" ? "SUCCESSFUL" : "FAILED"}}`), map[string]any{"workflow.failures": `{"foo":"bar"}`}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`SUCCESSFUL`), replaced)
			replaced, err = Replace(ctx, toJSONString(`{{=workflow.failures == "{\"foo\":\"bar\"}" ? "SUCCESSFUL" : "FAILED"}}`), map[string]any{"workflow.failures": `{"foo":"barr"}`}, false)
			require.NoError(t, err)
			assert.Equal(t, toJSONString(`FAILED`), replaced)
		})
		t.Run("Unresolved", func(t *testing.T) {
			t.Run("Allowed", func(t *testing.T) {
				_, err := Replace(ctx, toJSONString("{{=foo}}"), nil, true)
				require.NoError(t, err)
			})
			t.Run("AllowedRetries", func(t *testing.T) {
				replaced, err := Replace(ctx, toJSONString("{{=sprig.int(retries)}}"), nil, true)
				require.NoError(t, err)
				assert.Equal(t, toJSONString("{{=sprig.int(retries)}}"), replaced)
			})
			t.Run("AllowedWorkflowStatus", func(t *testing.T) {
				replaced, err := Replace(ctx, toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, true)
				require.NoError(t, err)
				assert.Equal(t, toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), replaced)
			})
			t.Run("AllowedWorkflowFailures", func(t *testing.T) {
				replaced, err := Replace(ctx, toJSONString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, true)
				require.NoError(t, err)
				assert.Equal(t, toJSONString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), replaced)
			})
			t.Run("Disallowed", func(t *testing.T) {
				_, err := Replace(ctx, toJSONString("{{=foo}}"), nil, false)
				require.EqualError(t, err, "failed to evaluate expression: foo is missing")
			})
			t.Run("DisallowedWorkflowStatus", func(t *testing.T) {
				_, err := Replace(ctx, toJSONString(`{{=workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, false)
				require.ErrorContains(t, err, "failed to evaluate expression")
			})
			t.Run("DisallowedWorkflowFailures", func(t *testing.T) {
				_, err := Replace(ctx, toJSONString(`{{=workflow.failures == "Succeeded" ? "SUCCESSFUL" : "FAILED"}}`), nil, false)
				require.ErrorContains(t, err, "failed to evaluate expression")
			})
		})
		t.Run("Error", func(t *testing.T) {
			_, err := Replace(ctx, toJSONString("{{=!}}"), nil, false)
			require.ErrorContains(t, err, "failed to evaluate expression")
		})
	})
}

func TestNestedReplaceString(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	replaceMap := map[string]any{"inputs.parameters.message": "hello world"}

	test := toJSONString(`{{- with secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`)
	replacement, err := Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with {{ secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with {{ secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ {{ }} secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("{{- with {{ {{ }} secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}"), replacement)

	test = toJSONString(`{{- with {{ {{ }} secret "{{does-not-exist}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`)

	replacement, err = Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, test, replacement)
}

func TestReplaceStringWithWhiteSpace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	replaceMap := map[string]any{"inputs.parameters.message": "hello world"}

	test := toJSONString(`{{ inputs.parameters.message }}`)
	replacement, err := Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("hello world"), replacement)
}

func TestReplaceStringWithExpression(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	replaceMap := map[string]any{"inputs.parameters.message": "hello world"}

	test := toJSONString(`test {{= sprig.trunc(5, inputs.parameters.message) }}`)
	replacement, err := Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("test hello"), replacement)

	test = toJSONString(`test {{= sprig.trunc(-5, inputs.parameters.message) }}`)
	replacement, err = Replace(ctx, test, replaceMap, true)
	require.NoError(t, err)
	assert.Equal(t, toJSONString("test world"), replacement)
}

// TestReplaceStrictAnyNilValues verifies the absent-optional (nil) semantics: expression tags can
// distinguish a present-but-nil value (skipped step output with no default) from an empty string
// via ??, and a simple tag resolving to nil is a terminal error (not a missing-variable requeue).
func TestReplaceStrictAnyNilValues(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	replaceMap := map[string]any{
		"tasks.producer.outputs.parameters.msg": nil,
		"tasks.other.outputs.parameters.msg":    "real",
	}

	t.Run("ExpressionFallbackFires", func(t *testing.T) {
		r, err := ReplaceStrictAny(ctx, toJSONString(`{{= tasks.producer.outputs.parameters.msg ?? 'fallback'}}`), replaceMap, []string{"tasks", "steps"})
		require.NoError(t, err)
		assert.Equal(t, toJSONString("fallback"), r)
	})
	t.Run("ExpressionFallbackNotFiredForEmptyString", func(t *testing.T) {
		emptyValueMap := map[string]any{"tasks.producer.outputs.parameters.msg": ""}
		r, err := ReplaceStrictAny(ctx, toJSONString(`{{= tasks.producer.outputs.parameters.msg ?? 'fallback'}}`), emptyValueMap, []string{"tasks", "steps"})
		require.NoError(t, err)
		assert.Equal(t, toJSONString(""), r)
	})
	t.Run("BareExpressionRefNilErrors", func(t *testing.T) {
		_, err := ReplaceStrictAny(ctx, toJSONString(`{{= tasks.producer.outputs.parameters.msg}}`), replaceMap, []string{"tasks", "steps"})
		require.Error(t, err)
	})
	t.Run("SimpleTagAbsentValueErrors", func(t *testing.T) {
		_, err := ReplaceStrictAny(ctx, toJSONString(`pre-{{tasks.producer.outputs.parameters.msg}}-post`), replaceMap, []string{"tasks", "steps"})
		require.ErrorContains(t, err, "absent optional")
		assert.False(t, IsMissingVariableErr(err), "absent optional must be terminal, not a requeue")
	})
	t.Run("SimpleTagEmptyStringStillSubstitutes", func(t *testing.T) {
		emptyValueMap := map[string]any{"tasks.producer.outputs.parameters.msg": ""}
		r, err := ReplaceStrictAny(ctx, toJSONString(`pre-{{tasks.producer.outputs.parameters.msg}}-post`), emptyValueMap, []string{"tasks", "steps"})
		require.NoError(t, err)
		assert.Equal(t, toJSONString("pre--post"), r)
	})
	t.Run("RealValueStillWins", func(t *testing.T) {
		r, err := ReplaceStrictAny(ctx, toJSONString(`{{= tasks.other.outputs.parameters.msg ?? 'fallback'}}`), replaceMap, []string{"tasks", "steps"})
		require.NoError(t, err)
		assert.Equal(t, toJSONString("real"), r)
	})
	t.Run("MissingStrictIdentifierStillErrors", func(t *testing.T) {
		_, err := ReplaceStrictAny(ctx, toJSONString(`{{= tasks.unknown.outputs.parameters.msg}}`), replaceMap, []string{"tasks", "steps"})
		require.Error(t, err)
	})
}

// TestReplaceStrictAnyNilNestedTag verifies that a nested tag whose value is a present-but-nil
// absent optional (a skipped step's defaultless output) is a terminal resolution error: the
// composite outer tag cannot meaningfully resolve from an absent value, and the error must not be
// classified as a missing-variable error (which would requeue forever). A real empty string nested
// value still collapses as before.
func TestReplaceStrictAnyNilNestedTag(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	const input = `{{outer-{{steps.x.outputs.parameters.key}}}}`

	t.Run("AbsentNestedValueErrors", func(t *testing.T) {
		replaceMap := map[string]any{
			"steps.x.outputs.parameters.key": nil, // skipped, no producer default
			"outer-":                         "resolved-outer",
		}
		_, err := ReplaceStrictAny(ctx, toJSONString(input), replaceMap, []string{"tasks", "steps"})
		require.Error(t, err)
		assert.False(t, IsMissingVariableErr(err), "absent nested value must be terminal, not a requeue")
	})

	t.Run("EmptyNestedValueStillCollapses", func(t *testing.T) {
		replaceMap := map[string]any{
			"steps.x.outputs.parameters.key": "", // produced a real empty string
			"outer-":                         "resolved-outer",
		}
		pass1, err := ReplaceStrictAny(ctx, toJSONString(input), replaceMap, []string{"tasks", "steps"})
		require.NoError(t, err)
		assert.Equal(t, toJSONString("{{outer-}}"), pass1)

		pass2, err := ReplaceStrictAny(ctx, pass1, replaceMap, []string{"tasks", "steps"})
		require.NoError(t, err)
		assert.Equal(t, toJSONString("resolved-outer"), pass2)
	})
}
