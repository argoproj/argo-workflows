package template

import (
	"strings"

	"github.com/expr-lang/expr"

	"github.com/argoproj/argo-workflows/v4/errors"
)

func ResolveVar(s string, m map[string]interface{}) (interface{}, error) {
	tag := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, prefix), suffix))
	kind, expression := parseTag(tag)
	switch kind {
	case kindExpression:
		program, err := expr.Compile(expression, expr.Env(m))
		if err != nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "Unable to compile: %q", expression)
		}
		result, err := expr.Run(program, m)
		if err != nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "Invalid expression: %q", expression)
		}
		if result == nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: %q", tag)
		}
		return result, nil
	default:
		v, ok := m[tag]
		if !ok {
			return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: %q", tag)
		}
		return v, nil
	}
}
