package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestSubstituteJSON: values are substituted in the JSON form, so quotes
// and backslashes arrive intact; a missing task or step reference is
// ErrRequeue, and other unresolved tags are left for later passes.
func TestSubstituteJSON(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	params := map[string]any{"tasks.a.outputs.result": `C:\temp "q"`}

	got, err := substituteJSON(ctx, "'{{tasks.a.outputs.result}}' == x", params)
	require.NoError(t, err)
	assert.Equal(t, `'C:\temp "q"' == x`, got)

	body, err := substituteJSON(ctx, wfv1.DAGTask{Name: "b", WithParam: "{{tasks.a.outputs.result}}", When: "{{item}} == {{inputs.parameters.p}}"}, params)
	require.NoError(t, err)
	assert.Equal(t, `C:\temp "q"`, body.WithParam)
	assert.Equal(t, "{{item}} == {{inputs.parameters.p}}", body.When)

	_, err = substituteJSON(ctx, "{{steps.missing.outputs.result}}", params)
	require.ErrorIs(t, err, ErrRequeue, "got %v", err)
}

// TestStepAdapterResolve: a resolved step stays a WorkflowStep named by the
// step name, in the same group with the same dependencies.
func TestStepAdapterResolve(t *testing.T) {
	step := &StepAdapter{step: &wfv1.WorkflowStep{Name: "x", Template: "t", When: "{{steps.a.status}} == Succeeded"}, dependencies: []string{"[0].a"}, groupIndex: 1}
	resolved, err := step.Resolve(func(body wfv1.DAGTask) (wfv1.DAGTask, error) {
		assert.Equal(t, "x", body.Name)
		body.When = "Succeeded == Succeeded"
		return body, nil
	})
	require.NoError(t, err)
	r, ok := resolved.(*StepAdapter)
	require.True(t, ok, "resolved step is a %T", resolved)
	assert.Equal(t, "[1].x", r.GetName())
	assert.Equal(t, "x", r.GetDisplayName())
	assert.Equal(t, []string{"[0].a"}, r.GetDependencies())
	assert.Equal(t, "Succeeded == Succeeded", r.GetWhen())
	assert.True(t, r.GetTemplateReferenceHolder().IsWorkflowStep())
	assert.Equal(t, "{{steps.a.status}} == Succeeded", step.GetWhen())
}
