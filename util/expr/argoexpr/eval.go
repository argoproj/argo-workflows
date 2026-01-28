package argoexpr

import (
	"fmt"

	"github.com/expr-lang/expr"
)

func EvalBool(input string, env interface{}) (bool, error) {
	program, err := expr.Compile(input, expr.Env(env))
	if err != nil {
		return false, err
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("unable to evaluate expression '%s': %w", input, err)
	}
	resultBool, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("unable to cast expression result '%s' to bool", result)
	}
	return resultBool, nil
}
