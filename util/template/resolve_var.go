package template

import (
	"strings"

	"github.com/expr-lang/expr"

	"github.com/argoproj/argo-workflows/v4/errors"
)

// ResolveVar resolves s against m and additionally returns the unwrapped tag
// (s with the surrounding "{{" / "}}" and whitespace stripped) so callers can reuse
// it without re-parsing.
func ResolveVar(s string, m map[string]any) (string, any, error) {
	tag := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, prefix), suffix))
	kind, expression := parseTag(tag)
	switch kind {
	case kindExpression:
		program, err := expr.Compile(expression, expr.Env(m))
		if err != nil {
			return tag, nil, errors.Errorf(errors.CodeBadRequest, "Unable to compile: %q", expression)
		}
		result, err := expr.Run(program, m)
		if err != nil {
			return tag, nil, errors.Errorf(errors.CodeBadRequest, "Invalid expression: %q", expression)
		}
		if result == nil {
			return tag, nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: %q", tag)
		}
		return tag, result, nil
	default:
		v, ok := m[tag]
		if !ok {
			return tag, nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: %q", tag)
		}
		return tag, v, nil
	}
}
