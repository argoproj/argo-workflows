package expr

import (
	"strings"

	"github.com/antonmedv/expr"
	expr1 "github.com/argoproj/pkg/expr"
	"github.com/doublerebel/bellows"
)

func Eval(expression string, env map[string]interface{}) (interface{}, error) {
	exprEnv := expr1.GetExprEnvFunctionMap()
	for k, v := range env {
		exprEnv[k] = v
	}
	expandEnv := bellows.Expand(exprEnv)
	return expr.Eval(strings.TrimSpace(expression), expandEnv)
}
