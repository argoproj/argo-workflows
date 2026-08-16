package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestEvaluateParameterDefaults(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("EvaluatesInTwoPasses", func(t *testing.T) {
		spec := v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{
				{Name: "message", Default: v1alpha1.AnyStringPtr("hello")},
				{Name: "count", Default: v1alpha1.AnyStringPtr("{{=1 + 2}}")},
				{Name: "unresolved", Default: v1alpha1.AnyStringPtr("{{=workflow.parameters.missing}}")},
			}},
			Templates: []v1alpha1.Template{{
				Name: "main",
				Inputs: v1alpha1.Inputs{Parameters: []v1alpha1.Parameter{
					{Name: "upper", Default: v1alpha1.AnyStringPtr("{{=sprig.upper(workflow.parameters.message)}}")},
					{Name: "sum", Default: v1alpha1.AnyStringPtr("{{=workflow.parameters.count + 4}}")},
					{Name: "dependent-unresolved", Default: v1alpha1.AnyStringPtr("{{=workflow.parameters.unresolved}}")},
				}},
			}},
		}

		require.NoError(t, EvaluateParameterDefaults(ctx, &spec))

		assert.Nil(t, spec.Arguments.Parameters[0].Value)
		require.NotNil(t, spec.Arguments.Parameters[1].Value)
		assert.Equal(t, "3", spec.Arguments.Parameters[1].Value.String())
		assert.Equal(t, "{{=1 + 2}}", spec.Arguments.Parameters[1].Default.String())
		assert.Nil(t, spec.Arguments.Parameters[2].Value)
		assert.Equal(t, "{{=workflow.parameters.missing}}", spec.Arguments.Parameters[2].Default.String())

		require.NotNil(t, spec.Templates[0].Inputs.Parameters[0].Value)
		assert.Equal(t, "HELLO", spec.Templates[0].Inputs.Parameters[0].Value.String())
		require.NotNil(t, spec.Templates[0].Inputs.Parameters[1].Value)
		assert.Equal(t, "7", spec.Templates[0].Inputs.Parameters[1].Value.String())
		assert.Nil(t, spec.Templates[0].Inputs.Parameters[2].Value)
		assert.Equal(t, "{{=workflow.parameters.unresolved}}", spec.Templates[0].Inputs.Parameters[2].Default.String())
	})

	t.Run("ReturnsInvalidSyntax", func(t *testing.T) {
		spec := v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{{
				Name:    "invalid",
				Default: v1alpha1.AnyStringPtr("{{=1 + }}"),
			}}},
		}

		err := EvaluateParameterDefaults(ctx, &spec)
		require.ErrorContains(t, err, "failed to evaluate default for workflow parameter \"invalid\"")
		require.ErrorContains(t, err, "failed to evaluate expression")
	})
}
