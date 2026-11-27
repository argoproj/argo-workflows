package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestTemplateScope(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_template_scope/workflow.yaml")
	wftmpl1 := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_template_scope/workflow-template-1.yaml")
	wftmpl2 := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_template_scope/workflow-template-2.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl1, wftmpl2)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	wf = woc.wf

	node := findNodeByName(wf.Status.Nodes, "test-template-scope[0].step")
	require.NotNil(t, node, "Node %s not found", "test-templte-scope")
	assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
	assert.Equal(t, "local/test-template-scope", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0]")
	require.NotNil(t, node, "Node %s not found", "test-templte-scope[0]")
	assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].hello")
	require.NotNil(t, node, "Node %s not found", "test-templte-scope[0].hello")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl")
	require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl")
	assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0]")
	require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0]")
	assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0].hello")
	require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0].hello")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)
}

func TestTemplateScopeWithParam(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_template_scope/with-param-workflow.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_template_scope/with-param-workflow-template-1.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param")
	assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
	assert.Equal(t, "local/test-template-scope-with-param", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0]")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0]")
	assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0].print-string(0:x)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(0:x)")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0].print-string(1:y)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(1:y)")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0].print-string(2:z)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(2:z)")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)
}

func TestTemplateScopeNestedStepsWithParams(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_template_scope/nested-steps-with-params-workflow.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_template_scope/nested-steps-with-params-workflow-template-1.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param")
	assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
	assert.Equal(t, "local/test-template-scope-nested-steps-with-params", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0]")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0]")
	assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main")
	assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0]")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0]")
	assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0].print-string(0:x)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(0:x)")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0].print-string(1:y)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(1:y)")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0].print-string(2:z)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(2:z)")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)
}

func TestTemplateScopeDAG(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_template_scope/dag-workflow.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_template_scope/dag-workflow-template-1.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-dag")
	assert.Equal(t, wfv1.NodeTypeDAG, node.Type)
	assert.Equal(t, "local/test-template-scope-dag", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.A")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.A")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B")
	assert.Equal(t, wfv1.NodeTypeTaskGroup, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B(0:x)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B(1:y)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B(2:z)")
	require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)
}

func findNodeByName(nodes map[string]wfv1.NodeStatus, name string) *wfv1.NodeStatus {
	for _, node := range nodes {
		if node.Name == name {
			return &node
		}
	}
	return nil
}

func TestTemplateClusterScope(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator_template_scope/cluster-scope-workflow.yaml")
	cwftmpl := wfv1.MustUnmarshalClusterWorkflowTemplate("@testdata/operator_template_scope/cluster-scope-workflow-template-1.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator_template_scope/workflow-template-2.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf, cwftmpl, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope[0].step")
	require.NotNil(t, node, "Node %s not found", "test-templte-scope")
	assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
	assert.Equal(t, "local/test-template-scope", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0]")
	require.NotNil(t, node, "Node %s not found", "test-templte-scope[0]")
	assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
	assert.Equal(t, "cluster/test-template-scope-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].hello")
	require.NotNil(t, node, "Node %s not found", "test-templte-scope[0].hello")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "cluster/test-template-scope-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl")
	require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl")
	assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
	assert.Equal(t, "cluster/test-template-scope-1", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0]")
	require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0]")
	assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0].hello")
	require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0].hello")
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	assert.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)
}
