package expr

import (
	"fmt"
	"strings"

	"github.com/antonmedv/expr"

	"github.com/argoproj/argo/v3/util/json"
)

func Eval(expression string, env map[string]interface{}) (interface{}, error) {
	fmt.Println(expression)
	variables := GetVariable(expression)
	for _, v := range variables {
		if _, exist := env[v]; !exist {
			env[v] = nil
		}
	}
	ufmap, _ := json.Unflatten(env)
	result, err := expr.Eval(strings.TrimSpace(expression), ufmap)
	if err != nil {
		return nil, err
	}
	return result, nil
}
