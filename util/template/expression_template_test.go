package template

import (
	"testing"

	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"
	"github.com/stretchr/testify/assert"
)

func Test_hasVarInEnv(t *testing.T) {
	t.Run("direct key present", func(t *testing.T) {
		env := map[string]any{"foo": "bar"}
		assert.True(t, hasVarInEnv(env, "foo"))
	})

	t.Run("direct key absent", func(t *testing.T) {
		env := map[string]any{"foo": "bar"}
		assert.False(t, hasVarInEnv(env, "baz"))
	})

	t.Run("dotted key present as flat entry", func(t *testing.T) {
		env := map[string]any{"workflow.status": "Succeeded"}
		assert.True(t, hasVarInEnv(env, "workflow.status"))
	})

	t.Run("nested map traversal", func(t *testing.T) {
		env := map[string]any{
			"workflow": map[string]any{
				"status": "Succeeded",
			},
		}
		assert.True(t, hasVarInEnv(env, "workflow.status"))
	})

	t.Run("nested map key absent", func(t *testing.T) {
		env := map[string]any{
			"workflow": map[string]any{
				"status": "Succeeded",
			},
		}
		assert.False(t, hasVarInEnv(env, "workflow.name"))
	})

	t.Run("deeply nested map traversal", func(t *testing.T) {
		env := map[string]any{
			"a": map[string]any{
				"b": map[string]any{
					"c": 42,
				},
			},
		}
		assert.True(t, hasVarInEnv(env, "a.b.c"))
		assert.False(t, hasVarInEnv(env, "a.b.d"))
	})

	t.Run("struct field traversal", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		env := map[string]any{
			"obj": Inner{Value: "hello"},
		}
		assert.True(t, hasVarInEnv(env, "obj.Value"))
		assert.False(t, hasVarInEnv(env, "obj.Missing"))
	})

	t.Run("pointer to struct field traversal", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		inner := &Inner{Value: "hello"}
		env := map[string]any{
			"obj": inner,
		}
		assert.True(t, hasVarInEnv(env, "obj.Value"))
	})

	t.Run("nil pointer in env value", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		var inner *Inner
		env := map[string]any{
			"obj": inner,
		}
		assert.False(t, hasVarInEnv(env, "obj.Value"))
	})

	t.Run("empty env", func(t *testing.T) {
		env := map[string]any{}
		assert.False(t, hasVarInEnv(env, "foo"))
		assert.False(t, hasVarInEnv(env, "foo.bar"))
	})

	t.Run("non-traversable leaf value", func(t *testing.T) {
		// "foo" resolves to a string, can't traverse further
		env := map[string]any{"foo": "bar"}
		assert.False(t, hasVarInEnv(env, "foo.bar"))
	})
}

func Test_hasVariableInExpression(t *testing.T) {
	assert.True(t, hasVariableInExpression("retries", "retries"))
	assert.True(t, hasVariableInExpression("retries + 1", "retries"))
	assert.True(t, hasVariableInExpression("sprig(retries)", "retries"))
	assert.True(t, hasVariableInExpression("sprig(retries + 1) * 64", "retries"))
	assert.False(t, hasVariableInExpression("foo", "retries"))
	assert.False(t, hasVariableInExpression("retriesCustom + 1", "retries"))
	assert.True(t, hasVariableInExpression("item", "item"))
	assert.False(t, hasVariableInExpression("foo", "item"))
	assert.True(t, hasVariableInExpression("sprig.upper(item)", "item"))
}

func Test_getIdentifiers(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		want       []string
	}{
		{
			name:       "plain member path is required",
			expression: "item.requiredKey",
			want:       []string{"item", "item.requiredKey"},
		},
		{
			name:       "nil-coalescing guards the member path but keeps the base variable",
			expression: "item.optionalKey ?? 'fallback'",
			want:       []string{"item"},
		},
		{
			name:       "nil-coalescing with bracket notation is guarded",
			expression: "item['optionalKey'] ?? ''",
			want:       []string{"item"},
		},
		{
			name:       "optional chaining is guarded",
			expression: "item?.optionalKey",
			want:       []string{"item"},
		},
		{
			name:       "optional chaining also guards the optional receiver",
			expression: "item.a?.b",
			want:       []string{"item"},
		},
		{
			name:       "only the guarded member path is skipped",
			expression: "item.requiredKey + (item.optionalKey ?? 'x')",
			want:       []string{"item", "item.requiredKey"},
		},
		{
			name:       "base variable stays required so requeue still works",
			expression: "tasks.a.outputs.result ?? 'default'",
			want:       []string{"tasks"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getIdentifiers(tt.expression)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func Test_hasWorkflowParameters(t *testing.T) {
	expression := `workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"`
	exprToks, _ := lexer.Lex(file.NewSource(expression))
	variableToks, _ := lexer.Lex(file.NewSource("workflow.status"))
	variableToks = filterEOF(variableToks)
	assert.True(t, searchTokens(exprToks, variableToks))
	assert.True(t, hasVariableInExpression(expression, "workflow.status"))

	assert.False(t, hasVariableInExpression(expression, "workflow status"))
	assert.False(t, hasVariableInExpression(expression, "workflow .status"))

	expression = `"workflow.status" == "Succeeded" ? "SUCCESSFUL" : "FAILED"`
	assert.False(t, hasVariableInExpression(expression, "workflow .status"))
}
