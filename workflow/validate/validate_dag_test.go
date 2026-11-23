package validate

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestDAGCycle(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-cycle.yaml")
	require.ErrorContains(t, err, "cycle")
}

func TestAnyWithoutExpandingTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-any-without-expanding-task.yaml")
	require.ErrorContains(t, err, "does not contain any items")
}

func TestDAGUndefinedTemplate(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-undefined-template.yaml")
	require.ErrorContains(t, err, "undefined")
}

func TestDAGVariableResolution(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, "@testdata/dag/dag-unresolved-var.yaml")
	require.ErrorContains(t, err, "failed to resolve {{tasks.A.outputs.parameters.unresolvable}}")

	err = validate(ctx, "@testdata/dag/dag-resolved-var.yaml")
	require.NoError(t, err)

	err = validate(ctx, "@testdata/dag/dag-resolved-var-not-ancestor.yaml")
	require.ErrorContains(t, err, "templates.unresolved.tasks.C missing dependency 'B' for parameter 'message'")

	err = validate(ctx, "@testdata/dag/dag-resolved-global-var.yaml")
	require.NoError(t, err)
	err = validate(ctx, "@testdata/dag/dag-resolved-global-var-reversed.yaml")
	require.NoError(t, err)
}

func TestDAGArtifactResolution(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-resolved-art.yaml")
	require.NoError(t, err)
}

func TestDAGStatusReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, "@testdata/dag/dag-status-reference.yaml")
	require.NoError(t, err)

	err = validate(ctx, "@testdata/dag/dag-status-no-future-reference-simple.yaml")
	// Can't reference the status of steps that have not run yet
	require.ErrorContains(t, err, "failed to resolve {{tasks.B.status}}")

	err = validate(ctx, "@testdata/dag/dag-status-no-future-reference-when-future-reference-has-child.yaml")
	// Can't reference the status of steps that have not run yet, even if the referenced steps have children
	require.ErrorContains(t, err, "failed to resolve {{tasks.B.status}}")

	err = validate(ctx, "@testdata/dag/dag-status-past-reference-chain.yaml")
	require.NoError(t, err)

	err = validate(ctx, "@testdata/dag/dag-status-only-direct-ancestors.yaml")
	// Can't reference steps that are not direct ancestors of node
	// Here Node E references the status of Node B, even though it is not its descendent
	require.ErrorContains(t, err, "failed to resolve {{tasks.B.status}}")
}

func TestDAGNonExistantTarget(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-nonexistent-target.yaml")
	require.ErrorContains(t, err, "target 'DOESNTEXIST' is not defined")
}

func TestDAGTargetSubstitution(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-target-substitution.yaml")
	require.NoError(t, err)
}

func TestDAGTargetMissingInputParam(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-target-missing-input-param.yaml")
	require.Error(t, err)
}

func TestDependsAndDependencies(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-depends-and-dependencies.yaml")
	require.ErrorContains(t, err, "templates.dag-target cannot use both 'depends' and 'dependencies' in the same DAG template")
}

func TestDependsAndContinueOn(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-depends-and-continue-on.yaml")
	require.ErrorContains(t, err, "templates.dag-target cannot use 'continueOn' when using 'depends'. Instead use 'dep-task.Failed'/'dep-task.Errored'")
}

func TestDAGDependsDigit(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-depends-digit.yaml")
	require.ErrorContains(t, err, "templates.diamond.tasks.5A name cannot begin with a digit when using either 'depends' or 'dependencies'")
}

func TestDAGDependenciesDigit(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-dependencies-digit.yaml")
	require.ErrorContains(t, err, "templates.diamond.tasks.5A name cannot begin with a digit when using either 'depends' or 'dependencies'")
}

func TestDAGWithDigitNameNoDepends(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-with-digit-no-depends.yaml")
	require.NoError(t, err)
}

func TestDAGOutputsResolveTaskAggregatedOutputs(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-outputs-resolve-task-aggregated-outputs.yaml")
	require.NoError(t, err)
}

func TestDAGMissingParamValueInTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-missing-param-value-in-task.yaml")
	require.ErrorContains(t, err, ".valueFrom only allows: default, configMapKeyRef and supplied")
}

func TestDAGArgParamValueFromConfigMapInTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/dag-arg-param-value-from-config-map-in-task.yaml")
	require.NoError(t, err)
}

func TestFailDAGArgParamValueFromPathInTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), "@testdata/dag/fail-dag-arg-param-value-from-path-in-task.yaml")
	require.ErrorContains(t, err, "valueFrom only allows: default, configMapKeyRef and supplied")
}

func TestDagWithItemTemplateRefTmpl(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-with-item-template-ref-wf.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/dag/dag-with-item-template-ref-tmpl.yaml")

	err := createWorkflowTemplate(ctx, wftmpl)
	require.NoError(t, err)

	err = Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.NoError(t, err)

	_ = deleteWorkflowTemplate(ctx, wftmpl.Name)
}
