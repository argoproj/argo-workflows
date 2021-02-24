package expr

import (
	"fmt"
	"strings"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/file"
	"github.com/antonmedv/expr/parser/lexer"

	"github.com/argoproj/argo-workflows/v3/util/flatten"
)

func Eval(expression string, env map[string]interface{}) (interface{}, error) {
	variables := GetExprIdentifers(expression)
	for _, v := range variables {
		if _, exist := env[v]; !exist {
			env[v] = nil
		}
	}
	expandEnv := flatten.Expand(env)
	result, err := expr.Eval(strings.TrimSpace(expression), expandEnv)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetExprIdentifers get all variables from expression
// Current Expr has bug in AllowUndefinedVariables.
// https://github.com/antonmedv/expr/issues/167
func GetExprIdentifers(expr string) []string {
	source := file.NewSource(expr)
	token, _ := lexer.Lex(source)
	var variables []string
	variable := ""
	for _, item := range token {
		if item.Kind == lexer.Identifier {
			if variable == "" {
				variable = item.Value
			} else {
				variable = fmt.Sprintf("%s.%s", variable, item.Value)
			}
		}
		if item.Kind == lexer.Operator && item.Value != "." {
			if variable != "" {
				variables = append(variables, variable)
			}
			variable = ""
		}
	}
	if variable != "" {
		variables = append(variables, variable)
	}
	return variables
}
