package expr

import (
	"fmt"

	"github.com/antonmedv/expr/file"
	"github.com/antonmedv/expr/parser/lexer"
)

func GetVariable(expr string) []string {
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
