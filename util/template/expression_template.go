package template

import (
	"fmt"
	"io"
	"os"

	"github.com/antonmedv/expr"
)

func init() {
	if os.Getenv("EXPRESSION_TEMPLATES") != "false" {
		registerKind(kindExpression)
	}
}

func expressionReplace(w io.Writer, expression string, env map[string]interface{}, allowUnresolved bool) (int, error) {
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
