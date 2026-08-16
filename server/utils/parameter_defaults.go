package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	templateutil "github.com/argoproj/argo-workflows/v4/util/template"
)

// EvaluateParameterDefaults evaluates expression defaults in two passes: workflow parameters first,
// followed by template input parameters with the resolved workflow parameters in scope.
func EvaluateParameterDefaults(ctx context.Context, spec *v1alpha1.WorkflowSpec) error {
	globalParameters := make(map[string]any)
	for i := range spec.Arguments.Parameters {
		parameter := &spec.Arguments.Parameters[i]
		value, resolved, expression, err := evaluateParameterDefault(ctx, parameter.Default, nil)
		if err != nil {
			return fmt.Errorf("failed to evaluate default for workflow parameter %q: %w", parameter.Name, err)
		}
		if expression {
			if resolved {
				parameter.Value = v1alpha1.AnyStringPtr(value)
				globalParameters[parameter.Name] = value
			}
		} else if parameter.Default != nil {
			globalParameters[parameter.Name] = parameter.Default.String()
		}
	}

	replaceMap := map[string]any{
		"workflow": map[string]any{
			"parameters": globalParameters,
		},
	}
	for i := range spec.Templates {
		for j := range spec.Templates[i].Inputs.Parameters {
			parameter := &spec.Templates[i].Inputs.Parameters[j]
			value, resolved, _, err := evaluateParameterDefault(ctx, parameter.Default, replaceMap)
			if err != nil {
				return fmt.Errorf("failed to evaluate default for template %q input parameter %q: %w", spec.Templates[i].Name, parameter.Name, err)
			}
			if resolved {
				parameter.Value = v1alpha1.AnyStringPtr(value)
			}
		}
	}

	return nil
}

func evaluateParameterDefault(ctx context.Context, defaultValue *v1alpha1.AnyString, replaceMap map[string]any) (value any, resolved bool, expression bool, err error) {
	if defaultValue == nil {
		return nil, false, false, nil
	}
	valueString := strings.TrimSpace(defaultValue.String())
	if !strings.HasPrefix(valueString, "{{=") || !strings.HasSuffix(valueString, "}}") {
		return nil, false, false, nil
	}

	expressionBody := strings.TrimSpace(valueString[3 : len(valueString)-2])
	value, resolved, err = templateutil.EvaluateExpression(ctx, expressionBody, replaceMap, true)
	return value, resolved, true, err
}
