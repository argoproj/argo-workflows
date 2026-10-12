package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNativeParameterReferences(t *testing.T) {
	tests := []struct {
		input       any
		scope, name string
		want        bool
	}{
		{`{{tasks.build.outputs.parameters.items}}`, "tasks", "build", true},
		{`{{=tasks['build'].outputs.parameters.items}}`, "tasks", "build", true},
		{`{{=tasks.buildExtra.outputs.parameters.items}}`, "tasks", "build", false},
		{`{{=tasks[inputs.parameters.task].outputs.parameters.items}}`, "tasks", "build", true},
		{"{{workflow.parameters.valueExtra}}", "workflow", "value", false},
		{"workflow.parameters.value", "workflow", "value", false},
		{`{{="workflow.parameters.value"}}`, "workflow", "value", false},
		{`{{=workflow.parameters['value']}}`, "workflow", "value", true},
		{`{{=workflow['parameters']['value']}}`, "workflow", "value", true},
		{`{{=jsonpath(workflow.parameters['valueExtra'], '$.value')}}`, "workflow", "value", false},
		{`{{=workflow.parameters[inputs.parameters.key]}}`, "workflow", "value", true},
		{`{{=workflow.parameters[inputs.parameters.key]}}`, "inputs", "key", true},
		{`{{=toJson(workflow.parameters)}}`, "workflow", "value", true},
		{`{{=sprig.toJson(workflow)}}`, "workflow", "value", true},
		{`{{=workflow.name}}`, "workflow", "value", false},
		{`{{=workflow?.parameters?.value}}`, "workflow", "value", true},
		{`{{workflow.parameters.json}}`, "workflow", "value", true},
		{"{{ workflow.parameters.value }}", "workflow", "value", true},
		{`{{=let p = workflow.parameters; p.value}}`, "workflow", "value", true},
		{map[string]any{"items": []string{"literal", "{{inputs.parameters.value}}"}}, "inputs", "value", true},
		{map[string]string{"{{workflow.parameters.value}}": "label"}, "workflow", "value", true},
	}
	for _, tc := range tests {
		t.Run(tc.name+"-"+tc.scope, func(t *testing.T) {
			r, err := References(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.want, r.Uses(tc.scope, []string{tc.name}))
		})
	}
	for _, input := range []string{"{{invalid", "{{=foo(}}", "{{=$env['workflow'].parameters.value}}"} {
		_, err := References(input)
		require.Error(t, err)
	}
}
