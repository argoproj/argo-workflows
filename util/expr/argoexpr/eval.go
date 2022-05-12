package argoexpr

import (
	"fmt"

	"github.com/antonmedv/expr"
)

func EvalBool(input string, env interface{}) (bool, error) {
	result, err := expr.Eval(input, env)
	if err != nil {
		return false, fmt.Errorf("unable to evaluate expression '%s': %s", input, err)
	}
	resultBool, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("unable to cast expression result '%s' to bool", result)
	}
	return resultBool, nil
}
