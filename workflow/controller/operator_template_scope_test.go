package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var testTemplateScopeWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope
spec:
  entrypoint: entry
  templates:
  - name: entry
    templateRef:
      name: test-template-scope-1
      template: steps
`

var testTemplateScopeWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
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

var testTemplateScopeWorkflowTemplateYaml2 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-2
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
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wfctmplset := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("")

	wf := unmarshalWF(testTemplateScopeWorkflowYaml)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wftmpl := unmarshalWFTmpl(testTemplateScopeWorkflowTemplateYaml1)
	_, err = wfctmplset.Create(wftmpl)
	assert.NoError(t, err)
	wftmpl = unmarshalWFTmpl(testTemplateScopeWorkflowTemplateYaml2)
	_, err = wfctmplset.Create(wftmpl)
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope")
	if assert.NotNil(t, node, "Node %s not found", "test-templte-scope") {
		assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
		assert.Equal(t, "", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0]")
	if assert.NotNil(t, node, "Node %s not found", "test-templte-scope[0]") {
		assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		assert.Equal(t, "test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].hello")
	if assert.NotNil(t, node, "Node %s not found", "test-templte-scope[0].hello") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].other-wftmpl")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl") {
		assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
		assert.Equal(t, "test-template-scope-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].other-wftmpl[0]")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0]") {
		assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		assert.Equal(t, "test-template-scope-2", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope[0].other-wftmpl[0].hello")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope[0].other-wftmpl[0].hello") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-2", node.TemplateScope)
	}
}

var testTemplateScopeWithParamWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope-with-param
spec:
  entrypoint: main
  templates:
    - name: main
      templateRef:
        name: test-template-scope-with-param-1
        template: main
`

var testTemplateScopeWithParamWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-with-param-1
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
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wfctmplset := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("")

	wf := unmarshalWF(testTemplateScopeWithParamWorkflowYaml)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wftmpl := unmarshalWFTmpl(testTemplateScopeWithParamWorkflowTemplateYaml1)
	_, err = wfctmplset.Create(wftmpl)
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-with-param")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-with-param") {
		assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
		assert.Equal(t, "", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0]")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0]") {
		assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		assert.Equal(t, "test-template-scope-with-param-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].print-string(0:x)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(0:x)") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-with-param-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].print-string(1:y)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(1:y)") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-with-param-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-with-param[0].print-string(2:z)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0].print-string(2:z)") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-with-param-1", node.TemplateScope)
	}
}

var testTemplateScopeNestedStepsWithParamsWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope-nested-steps-with-params
spec:
  entrypoint: main
  templates:
    - name: main
      templateRef:
        name: test-template-scope-nested-steps-with-params-1
        template: main
`

var testTemplateScopeNestedStepsWithParamsWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-nested-steps-with-params-1
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
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wfctmplset := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("")

	wf := unmarshalWF(testTemplateScopeNestedStepsWithParamsWorkflowYaml)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wftmpl := unmarshalWFTmpl(testTemplateScopeNestedStepsWithParamsWorkflowTemplateYaml1)
	_, err = wfctmplset.Create(wftmpl)
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-with-param") {
		assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
		assert.Equal(t, "", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0]")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-with-param[0]") {
		assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		assert.Equal(t, "test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].main")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main") {
		assert.Equal(t, wfv1.NodeTypeSteps, node.Type)
		assert.Equal(t, "test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].main[0]")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0]") {
		assert.Equal(t, wfv1.NodeTypeStepGroup, node.Type)
		assert.Equal(t, "test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].main[0].print-string(0:x)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(0:x)") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].main[0].print-string(1:y)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(1:y)") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-nested-steps-with-params[0].main[0].print-string(2:z)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-nested-steps-with-params[0].main[0].print-string(2:z)") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-nested-steps-with-params-1", node.TemplateScope)
	}
}

var testTemplateScopeDAGWorkflowYaml = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-scope-dag
spec:
  entrypoint: main
  templates:
    - name: main
      templateRef:
        name: test-template-scope-dag-1
        template: main
`

var testTemplateScopeDAGWorkflowTemplateYaml1 = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template-scope-dag-1
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
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wfctmplset := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("")

	wf := unmarshalWF(testTemplateScopeDAGWorkflowYaml)
	_, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wftmpl := unmarshalWFTmpl(testTemplateScopeDAGWorkflowTemplateYaml1)
	_, err = wfctmplset.Create(wftmpl)
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	node := findNodeByName(wf.Status.Nodes, "test-template-scope-dag")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-dag") {
		assert.Equal(t, wfv1.NodeTypeDAG, node.Type)
		assert.Equal(t, "", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag.A")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-dag.A") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag.B")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B") {
		assert.Equal(t, wfv1.NodeTypeTaskGroup, node.Type)
		assert.Equal(t, "test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag.B(0:x)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag.B(1:y)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-dag-1", node.TemplateScope)
	}

	node = findNodeByName(wf.Status.Nodes, "test-template-scope-dag.B(2:z)")
	if assert.NotNil(t, node, "Node %s not found", "test-template-scope-dag.B(0:x") {
		assert.Equal(t, wfv1.NodeTypePod, node.Type)
		assert.Equal(t, "test-template-scope-dag-1", node.TemplateScope)
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
