package template

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/argoproj/argo-workflows/v3/util/mapper"
	"strings"
)

func Eval(x any, env any) (any, error) {
	return mapper.Map(x, func(g any) (any, error) {
		s, ok := g.(string)
		if ok {
			return eval(s, env)
		}
		return g, nil
	})
}

func eval(s string, env any) (string, error) {
	const prefix = "ƛ"
	if !strings.HasPrefix(s, prefix) {
		return s, nil
	}
	input := strings.TrimPrefix(s, prefix)
	output, err := expr.Eval(input, env)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate %s: %w", s, err)
	}
	result, ok := output.(string)
	if !ok {
		return "", fmt.Errorf("failed to evaluate %s: %w", s, fmt.Errorf("expected result to be a string, but got %T", output))
	}
	return result, nil
}
