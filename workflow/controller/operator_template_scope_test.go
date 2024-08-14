package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var testTemplateScopeWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    steps:
      - - name: step
          templateRef:
            name: test-template-scope-1
            template: steps
`

var testTemplateScopeWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-1
  namespace: default
spec:
  templates:
  - name: steps
    steps:
    - - name: hello
        template: hello
      - name: other-wftmpl
        templateRef:
          name: test-template-scope-2
          template: steps
  - name: hello
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        print("hello world")
`

var testTemplateScopeWorkflowTemplateYaml2 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-2
  namespace: default
spec:
  templates:
  - name: steps
    steps:
    - - name: hello
        template: hello
  - name: hello
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        print("hello world")
`

func TestTemplateScope(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testTemplateScopeWorkflowYaml)
	wftmpl1 := wfv1.MustUnmarshalWorkflowTemplate(testTemplateScopeWorkflowTemplateYaml1)
	wftmpl2 := wfv1.MustUnmarshalWorkflowTemplate(testTemplateScopeWorkflowTemplateYaml2)

	cancel, controller := newController(wf, wftmpl1, wftmpl2)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	ctx := context.Background()
	woc.operate(ctx)

	wf = woc.wf

	node := findNodeByName(wf.Status.Nodes, "test-template-scope[0].step")
	if require.NotNil(t, node, "Node %s not found", "test-templte-scope") {
		require.Equal(t, wfv1.NodeTypeSteps, node.Type)
		require.Equal(t, "local/test-template-scope", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0]")
	if require.NotNil(t, node, "Node %s not found", "test-templte-scope[0]") {
		require.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		require.Equal(t, "namespaced/test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].hello")
	if require.NotNil(t, node, "Node %s not found", "test-templte-scope[0].hello") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl") {
		require.Equal(t, wfv1.NodeTypeSteps, node.Type)
		require.Equal(t, "namespaced/test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0]")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0]") {
		require.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		require.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0].hello")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0].hello") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)
	}
}

var testTemplateScopeWithParamWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope-with-param
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: step
            templateRef:
              name: test-template-scope-with-param-1
              template: main
`

var testTemplateScopeWithParamWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-with-param-1
  namespace: default
spec:
  templates:
    - name: main
      steps:
        - - name: print-string
            template: print-string
            arguments:
              parameters:
               - name: letter
                 value: '{{item}}'
            withParam: '["x", "y", "z"]'
    - name: print-string
      inputs:
        parameters:
         - name: letter
      container:
        image: alpine:3.6
        command: [sh, -c]
        args: ["echo {{inputs.parameters.letter}}"]
`

func TestTemplateScopeWithParam(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testTemplateScopeWithParamWorkflowYaml)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(testTemplateScopeWithParamWorkflowTemplateYaml1)

	cancel, controller := newController(wf, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param") {
		require.Equal(t, wfv1.NodeTypeSteps, node.Type)
		require.Equal(t, "local/test-template-scope-with-param", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0]")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0]") {
		require.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		require.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0].print-string(0:x)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(0:x)") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0].print-string(1:y)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(1:y)") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].step[0].print-string(2:z)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(2:z)") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-with-param-1", node.TemplateScope)
	}
}

var testTemplateScopeNestedStepsWithParamsWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope-nested-steps-with-params
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: step
            templateRef:
              name: test-template-scope-nested-steps-with-params-1
              template: main
`

var testTemplateScopeNestedStepsWithParamsWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-nested-steps-with-params-1
  namespace: default
spec:
  templates:
    - name: main
      steps:
        - - name: main
            template: sub
    - name: sub
      steps:
        - - name: print-string
            template: print-string
            arguments:
              parameters:
               - name: letter
                 value: '{{item}}'
            withParam: '["x", "y", "z"]'
    - name: print-string
      inputs:
        parameters:
         - name: letter
      container:
        image: alpine:3.6
        command: [sh, -c]
        args: ["echo {{inputs.parameters.letter}}"]
`

func TestTemplateScopeNestedStepsWithParams(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testTemplateScopeNestedStepsWithParamsWorkflowYaml)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(testTemplateScopeNestedStepsWithParamsWorkflowTemplateYaml1)

	cancel, controller := newController(wf, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param") {
		require.Equal(t, wfv1.NodeTypeSteps, node.Type)
		require.Equal(t, "local/test-template-scope-nested-steps-with-params", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0]")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0]") {
		require.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		require.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main") {
		require.Equal(t, wfv1.NodeTypeSteps, node.Type)
		require.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0]")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0]") {
		require.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		require.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0].print-string(0:x)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(0:x)") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0].print-string(1:y)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(1:y)") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].step[0].main[0].print-string(2:z)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(2:z)") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}
}

var testTemplateScopeDAGWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope-dag
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: step
            templateRef:
              name: test-template-scope-dag-1
              template: main
`

var testTemplateScopeDAGWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-dag-1
  namespace: default
spec:
  templates:
    - name: main
      dag:
        tasks:
        - name: A
          template: print-string
          arguments:
            parameters:
            - name: letter
              value: 'A'
        - name: B
          template: print-string
          arguments:
            parameters:
            - name: letter
              value: '{{item}}'
          withParam: '["x", "y", "z"]'
    - name: print-string
      inputs:
        parameters:
         - name: letter
      container:
        image: alpine:3.6
        command: [sh, -c]
        args: ["echo {{inputs.parameters.letter}}"]
`

func TestTemplateScopeDAG(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testTemplateScopeDAGWorkflowYaml)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(testTemplateScopeDAGWorkflowTemplateYaml1)

	cancel, controller := newController(wf, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-dag") {
		require.Equal(t, wfv1.NodeTypeDAG, node.Type)
		require.Equal(t, "local/test-template-scope-dag", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.A")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.A") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B") {
		require.Equal(t, wfv1.NodeTypeTaskGroup, node.Type)
		require.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B(0:x)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B(1:y)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag[0].step.B(2:z)")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-dag-1", node.TemplateScope)
	}
}

func findNodeByName(nodes map[string]wfv1.NodeStatus, name string) *wfv1.NodeStatus {
	for _, node := range nodes {
		if node.Name == name {
			return &node
		}
	}
	return nil
}

var testTemplateClusterScopeWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope
  namespace: default
spec:
  entrypoint: entry
  templates:
  - name: entry
    steps:
      - - name: step
          templateRef:
            name: test-template-scope-1
            template: steps
            clusterScope: true
`

var testTemplateClusterScopeWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: test-template-scope-1
spec:
  templates:
  - name: steps
    steps:
    - - name: hello
        template: hello
      - name: other-wftmpl
        templateRef:
          name: test-template-scope-2
          template: steps
  - name: hello
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        print("hello world")
`

func TestTemplateClusterScope(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testTemplateClusterScopeWorkflowYaml)
	cwftmpl := wfv1.MustUnmarshalClusterWorkflowTemplate(testTemplateClusterScopeWorkflowTemplateYaml1)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(testTemplateScopeWorkflowTemplateYaml2)

	cancel, controller := newController(wf, cwftmpl, wftmpl)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("default")

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope[0].step")
	if require.NotNil(t, node, "Node %s not found", "test-templte-scope") {
		require.Equal(t, wfv1.NodeTypeSteps, node.Type)
		require.Equal(t, "local/test-template-scope", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0]")
	if require.NotNil(t, node, "Node %s not found", "test-templte-scope[0]") {
		require.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		require.Equal(t, "cluster/test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].hello")
	if require.NotNil(t, node, "Node %s not found", "test-templte-scope[0].hello") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "cluster/test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl") {
		require.Equal(t, wfv1.NodeTypeSteps, node.Type)
		require.Equal(t, "cluster/test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0]")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0]") {
		require.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		require.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].step[0].other-wftmpl[0].hello")
	if require.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0].hello") {
		require.Equal(t, wfv1.NodeTypePod, node.Type)
		require.Equal(t, "namespaced/test-template-scope-2", node.TemplateScope)
	}
}
