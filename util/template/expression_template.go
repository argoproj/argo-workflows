package template

import (
	"fmt"
	"io"
	"os"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/file"
	"github.com/antonmedv/expr/parser/lexer"
)

func init() {
	if os.Getenv("EXPRESSION_TEMPLATES") != "false" {
		registerKind(kindExpression)
	}
}

func expressionReplace(w io.Writer, expression string, env map[string]interface{}, allowUnresolved bool) (int, error) {
	if _, ok := env["retries"]; !ok && hasRetries(expression) && allowUnresolved {
		// this is to make sure expressions like `sprig.int(retries)` don't get resolved to 0 when `retries` don't exist in the env
		// See https://github.com/argoproj/argo-workflows/issues/5388
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	result, err := expr.Eval(expression, env)
	if (err != nil || result == nil) && allowUnresolved { //  <nil> result is also un-resolved, and any error can be unresolved
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	if err != nil {
		return 0, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	if result == nil {
		return 0, fmt.Errorf("failed to evaluate expression %q", expression)
	}
	return w.Write([]byte(fmt.Sprintf("%v", result)))
}

func envMap(replaceMap map[string]string) map[string]interface{} {
	envMap := make(map[string]interface{})
	for k, v := range replaceMap {
		envMap[k] = v
	}
	return envMap
}

// hasRetries checks if the variable `retries` exists in the expression template
func hasRetries(expression string) bool {
	tokens, err := lexer.Lex(file.NewSource(expression))
	if err != nil {
		return false
	}
	for _, token := range tokens {
		if token.Kind == lexer.Identifier && token.Value == "retries" {
			return true
		}
	}
	return false
}
