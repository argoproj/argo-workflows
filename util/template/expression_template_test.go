package template

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

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

func Test_CompareExpressionReplace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	replaceMap := map[string]any{"foo": "bar", "tasks": map[string]any{"A": "success"}}

	tests := []struct {
		expression      string
		allowUnresolved bool
	}{
		{`foo`, false},
		{`foo`, true},
		{`missing`, false},
		{`missing`, true},
		{`tasks.A`, false},
		{`tasks.A + foo`, false},
		{`tasks.A + missing`, true},
		{`tasks.A + missing`, false},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s_%v", tc.expression, tc.allowUnresolved), func(t *testing.T) {
			// Helper (Old logic)
			var b1 strings.Builder
			err1 := expressionReplaceHelper(ctx, &b1, tc.expression, replaceMap, tc.allowUnresolved)
			res1 := b1.String()

			// New logic
			var b2 strings.Builder
			_, err2 := expressionReplace(ctx, &b2, tc.expression, replaceMap, tc.allowUnresolved)
			res2 := b2.String()

			switch {
			case err1 != nil:
				// Old (Helper) returns error even if suppressed (to signal suppression).
				switch {
				case err2 == nil:
					// New returned success (nil error).
					// If Old suppressed an error, New should also suppress it (return unresolved tag).
					if strings.Contains(res2, "{{=") {
						return
					}
					t.Errorf("Old suppressed error (%v) but New resolved it to %q", err1, res2)
				case strings.Contains(err1.Error(), "expr run error:"):
					// Both returned error.
					// If Old suppressed a RUNTIME error (expr run error), but New failed hard.
					// This is the expected divergence when variables are present.
					return
				case strings.Contains(err1.Error(), "variable not in env"):
					// If Old suppressed "variable not in env", New should also suppress it?
					// New `expressionReplaceStrict` detects missing vars and sets allowUnresolved=true.
					// So New should return nil error (unresolved tag).
					t.Errorf("Old suppressed missing var (%v), but New errored: %v", err1, err2)
				}
			case err2 != nil:
				t.Errorf("Old succeeded (res: %s) but New errored (%v)", res1, err2)
			case res1 != res2:
				t.Errorf("Results differ: Old=%q, New=%q", res1, res2)
			}
		})
	}
}

func expressionReplaceHelper(ctx context.Context, w io.Writer, expression string, env map[string]any, allowUnresolved bool) error {
	log := logging.RequireLoggerFromContext(ctx)
	// The template is JSON-marshaled. This JSON-unmarshals the expression to undo any character escapes.
	var unmarshalledExpression string
	err := json.Unmarshal(fmt.Appendf(nil, `"%s"`, expression), &unmarshalledExpression)
	if err != nil && allowUnresolved {
		log.WithError(err).Debug(ctx, "unresolved is allowed")
		fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
		return fmt.Errorf("json unmarshal error: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to unmarshall JSON expression: %w", err)
	}

	varNameNotInEnv := anyVarNotInEnv(unmarshalledExpression, env)
	if varNameNotInEnv != nil && allowUnresolved {
		// this is to make sure expressions don't get resolved to nil or an empty string when certain variables
		// don't exist in the env during the "global" replacement.
		// See https://github.com/argoproj/argo-workflows/issues/5388, https://github.com/argoproj/argo-workflows/issues/15008,
		// https://github.com/argoproj/argo-workflows/issues/10393, https://github.com/expr-lang/expr/issues/330
		log.WithField("variable", *varNameNotInEnv).Debug(ctx, "variable not in env but unresolved is allowed")
		fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
		return fmt.Errorf("variable not in env: %s", *varNameNotInEnv)
	}

	program, err := expr.Compile(unmarshalledExpression, expr.Env(env))
	// This allowUnresolved check is not great
	// it allows for errors that are obviously
	// not failed reference checks to also pass
	if err != nil && !allowUnresolved {
		return fmt.Errorf("failed to evaluate expression: %w", err)
	}
	result, err := expr.Run(program, env)
	if (err != nil || result == nil) && allowUnresolved {
		//  <nil> result is also un-resolved, and any error can be unresolved
		log.WithError(err).Debug(ctx, "Result and error are unresolved")
		fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
		if err != nil {
			return fmt.Errorf("expr run error: %w", err)
		}
		return fmt.Errorf("expr result nil")
	}
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %w", err)
	}
	if result == nil {
		return fmt.Errorf("failed to evaluate expression %q", expression)
	}
	resultMarshaled, err := json.Marshal(result)
	if (err != nil || resultMarshaled == nil) && allowUnresolved {
		log.WithError(err).Debug(ctx, "resultMarshaled is nil and unresolved is allowed ")
		fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
		if err != nil {
			return fmt.Errorf("json marshal error: %w", err)
		}
		return fmt.Errorf("json marshal result nil")
	}
	if err != nil {
		return fmt.Errorf("failed to marshal evaluated expression: %w", err)
	}
	if resultMarshaled == nil {
		return fmt.Errorf("failed to marshal evaluated marshaled expression %q", expression)
	}
	marshaledLength := len(resultMarshaled)

	// Trim leading and trailing quotes. The value is being inserted into something that's already a string.
	if len(resultMarshaled) > 1 && resultMarshaled[0] == '"' && resultMarshaled[marshaledLength-1] == '"' {
		_, err := w.Write(resultMarshaled[1 : marshaledLength-1])
		return err
	}

	resultQuoted := []byte(strconv.Quote(string(resultMarshaled)))
	_, err = w.Write(resultQuoted[1 : len(resultQuoted)-1])
	return err
}
