package template

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func Test_hasVarInEnv(t *testing.T) {
	t.Run("direct key present", func(t *testing.T) {
		env := map[string]any{"foo": "bar"}
		assert.True(t, hasVarInEnv(env, "foo", maxKeyParts(env)))
	})

	t.Run("direct key absent", func(t *testing.T) {
		env := map[string]any{"foo": "bar"}
		assert.False(t, hasVarInEnv(env, "baz", maxKeyParts(env)))
	})

	t.Run("dotted key present as flat entry", func(t *testing.T) {
		env := map[string]any{"workflow.status": "Succeeded"}
		assert.True(t, hasVarInEnv(env, "workflow.status", maxKeyParts(env)))
	})

	t.Run("nested map traversal", func(t *testing.T) {
		env := map[string]any{
			"workflow": map[string]any{
				"status": "Succeeded",
			},
		}
		assert.True(t, hasVarInEnv(env, "workflow.status", maxKeyParts(env)))
	})

	t.Run("nested map key absent", func(t *testing.T) {
		env := map[string]any{
			"workflow": map[string]any{
				"status": "Succeeded",
			},
		}
		assert.False(t, hasVarInEnv(env, "workflow.name", maxKeyParts(env)))
	})

	t.Run("deeply nested map traversal", func(t *testing.T) {
		env := map[string]any{
			"a": map[string]any{
				"b": map[string]any{
					"c": 42,
				},
			},
		}
		assert.True(t, hasVarInEnv(env, "a.b.c", maxKeyParts(env)))
		assert.False(t, hasVarInEnv(env, "a.b.d", maxKeyParts(env)))
	})

	t.Run("struct field traversal", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		env := map[string]any{
			"obj": Inner{Value: "hello"},
		}
		assert.True(t, hasVarInEnv(env, "obj.Value", maxKeyParts(env)))
		assert.False(t, hasVarInEnv(env, "obj.Missing", maxKeyParts(env)))
	})

	t.Run("pointer to struct field traversal", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		inner := &Inner{Value: "hello"}
		env := map[string]any{
			"obj": inner,
		}
		assert.True(t, hasVarInEnv(env, "obj.Value", maxKeyParts(env)))
	})

	t.Run("nil pointer in env value", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		var inner *Inner
		env := map[string]any{
			"obj": inner,
		}
		assert.False(t, hasVarInEnv(env, "obj.Value", maxKeyParts(env)))
	})

	t.Run("empty env", func(t *testing.T) {
		env := map[string]any{}
		assert.False(t, hasVarInEnv(env, "foo", maxKeyParts(env)))
		assert.False(t, hasVarInEnv(env, "foo.bar", maxKeyParts(env)))
	})

	t.Run("non-traversable leaf value", func(t *testing.T) {
		// "foo" resolves to a string, can't traverse further
		env := map[string]any{"foo": "bar"}
		assert.False(t, hasVarInEnv(env, "foo.bar", maxKeyParts(env)))
	})
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

func Test_anyVarNotInEnv(t *testing.T) {
	emptyEnv := map[string]any{}
	missing := func(expression string) *string { return anyVarNotInEnv(expression, emptyEnv) }

	t.Run("late-binding variables detected when absent", func(t *testing.T) {
		for _, expression := range []string{
			"retries",
			"retries + 1",
			"sprig(retries)",
			"sprig(retries + 1) * 64",
			"item",
			"sprig.upper(item)",
			`workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"`,
		} {
			assert.NotNil(t, missing(expression), expression)
		}
	})

	t.Run("unrelated identifiers do not match", func(t *testing.T) {
		for _, expression := range []string{
			"foo",
			"retriesCustom + 1",
		} {
			assert.Nil(t, missing(expression), expression)
		}
	})

	t.Run("string literal is not an identifier", func(t *testing.T) {
		assert.Nil(t, missing(`"workflow.status" == "Succeeded" ? "SUCCESSFUL" : "FAILED"`))
	})

	t.Run("present in env is not reported", func(t *testing.T) {
		env := map[string]any{"retries": 1}
		assert.Nil(t, anyVarNotInEnv("retries + 1", env))
	})
}

func Test_missingVarsInEnv(t *testing.T) {
	t.Run("nil leaf counts as present", func(t *testing.T) {
		env := map[string]any{"tasks": map[string]any{"a": map[string]any{"out": nil}}}
		missing, err := missingVarsInEnv("tasks.a.out ?? 'x'", env)
		require.NoError(t, err)
		assert.Empty(t, missing)
	})
	t.Run("missing identifiers reported", func(t *testing.T) {
		missing, err := missingVarsInEnv("foo + bar", map[string]any{"foo": "1"})
		require.NoError(t, err)
		assert.Equal(t, []string{"bar"}, missing)
	})
	t.Run("unparseable errors", func(t *testing.T) {
		_, err := missingVarsInEnv("foo +", map[string]any{})
		require.Error(t, err)
	})
	t.Run("each prefix of a member path is checked against dotted keys", func(t *testing.T) {
		env := map[string]any{"a.b": map[string]any{"b": "x"}}
		missing, err := missingVarsInEnv("a.b.b.b", env)
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "a.b.b.b"}, missing)
	})
	t.Run("deep member path", func(t *testing.T) {
		expression := "a" + strings.Repeat(".b", 500)
		want := []string{"a"}
		for i := 1; i <= 500; i++ {
			want = append(want, "a"+strings.Repeat(".b", i))
		}
		got, err := getIdentifiers(expression)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		missing, err := missingVarsInEnv(expression, map[string]any{"a.b.b": map[string]any{"b": map[string]any{}}})
		require.NoError(t, err)
		assert.Equal(t, append([]string{"a", "a.b"}, want[4:]...), missing)
	})
}

func BenchmarkReplaceStrictDeepMemberPath(b *testing.B) {
	replaceMap := map[string]any{
		"workflow.name":          "wf",
		"tasks.a.outputs.result": "x",
	}
	for _, depth := range []int{1, 100, 1000, 4000} {
		b.Run(strconv.Itoa(depth), func(b *testing.B) {
			ctx := logging.TestContext(b.Context())
			tmpl, err := NewTemplate("{{=tasks" + strings.Repeat(".a", depth) + "}}")
			require.NoError(b, err)
			for b.Loop() {
				_, _ = tmpl.ReplaceStrict(ctx, replaceMap, []string{"tasks"})
			}
		})
	}
}
