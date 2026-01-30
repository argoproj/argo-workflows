package template

import (
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func FuzzSimpleReplace(f *testing.F) {
	f.Add("foo", true)
	f.Add("foo", false)
	f.Add("nested-{{foo}}", true)
	f.Add("nested-{{foo}}", false)
	f.Add("steps.step.outputs.artifacts.art", false)

	f.Fuzz(func(t *testing.T, tag string, allowUnresolved bool) {
		ctx := logging.TestContext(t.Context())
		replaceMap := map[string]interface{}{
			"foo":                              "bar",
			"baz":                              "qux",
			"nested":                           "value",
			"int":                              1,
			"steps.step.outputs.artifacts.art": "path/to/art",
		}

		var b1 strings.Builder
		err1 := simpleReplaceHelper(ctx, &b1, tag, replaceMap, allowUnresolved)
		res1 := b1.String()

		var b2 strings.Builder
		_, err2 := simpleReplace(ctx, &b2, tag, replaceMap, allowUnresolved)
		res2 := b2.String()

		if (err1 == nil) != (err2 == nil) {
			// Deviation for artifacts when allowUnresolved is false
			if !allowUnresolved && strings.Contains(tag, ".outputs.artifacts.") {
				// Old errors, New succeeds (returns tag).
				if err1 != nil && err2 == nil {
					return
				}
			}
			t.Fatalf("Error mismatch for tag=%q allow=%v: OldErr=%v, NewErr=%v", tag, allowUnresolved, err1, err2)
		}

		if err1 == nil && res1 != res2 {
			t.Fatalf("Result mismatch for tag=%q allow=%v: Old=%q, New=%q", tag, allowUnresolved, res1, res2)
		}
	})
}

func FuzzExpressionReplace(f *testing.F) {
	f.Add("foo", true)
	f.Add("foo", false)
	f.Add("foo + 1", true)
	f.Add("tasks.A", false)
	f.Add("tasks.A.outputs.result", false)

	f.Fuzz(func(t *testing.T, expression string, allowUnresolved bool) {
		ctx := logging.TestContext(t.Context())
		env := map[string]interface{}{
			"foo": "bar",
			"val": 1,
			"tasks": map[string]interface{}{
				"A": map[string]interface{}{
					"outputs": map[string]interface{}{
						"result": "success",
					},
				},
			},
			"inputs": map[string]interface{}{
				"parameters": map[string]interface{}{
					"param": "value",
				},
			},
		}

		var b1 strings.Builder
		err1 := expressionReplaceHelper(ctx, &b1, expression, env, allowUnresolved)
		res1 := b1.String()

		var b2 strings.Builder
		_, err2 := expressionReplace(ctx, &b2, expression, env, allowUnresolved)
		res2 := b2.String()

		if err1 != nil {
			// Old (Helper) returned error.

			if err2 == nil {
				// New returned success.
				if strings.Contains(res2, "{{=") {
					return // Both suppressed.
				}
				t.Fatalf("Old suppressed error (%v) but New resolved it to %q", err1, res2)
			} else {
				// Both errored.
				// If Old suppressed "expr run error", and New failed hard.
				if strings.Contains(err1.Error(), "expr run error") {
					return
				}
				// If Old suppressed "variable not in env", New should NOT error (it should suppress).
				if strings.Contains(err1.Error(), "variable not in env") && allowUnresolved {
					t.Fatalf("Old suppressed missing var (%v), but New errored: %v", err1, err2)
				}
			}
		} else {
			// Old succeeded.
			if err2 != nil {
				t.Fatalf("Old succeeded (res: %s) but New errored (%v)", res1, err2)
			}
			if res1 != res2 {
				t.Fatalf("Result mismatch for expr=%q allow=%v: Old=%q, New=%q", expression, allowUnresolved, res1, res2)
			}
		}
	})
}
