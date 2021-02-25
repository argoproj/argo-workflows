package expr

import (
	"strings"

	"github.com/antonmedv/expr"
	"github.com/doublerebel/bellows"
)

func Eval(expression string, env map[string]interface{}) (interface{}, error) {
	addExprFunctions(env)
	expandEnv := bellows.Expand(env)

	return expr.Eval(strings.TrimSpace(expression), expandEnv)
}
