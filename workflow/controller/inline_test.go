package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestInlineDAG(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: inline-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            inline:
              container:
                image: argoproj/argosay:v2
                args:
                  - echo
                  - "{{inputs.parameters.foo}}"
              inputs:
                parameters:
                  - name: foo
                    value: bar
`)
	cancel, wfc := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, wfc)
	woc.operate(context.Background())
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestInlineSteps(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-inline-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: a
            inline:
              inputs:
                parameters:
                  - name: message
                    value: foo
              container:
                image: docker/whalesay:latest
                command:
                  - cowsay
                args:
                  - '{{inputs.parameters.message}}'
`)
	cancel, wfc := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, wfc)
	woc.operate(context.Background())
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	node := woc.wf.Status.Nodes.FindByDisplayName("a")
	assert.Equal(t, "message", node.Inputs.Parameters[0].Name)
	assert.Equal(t, "foo", node.Inputs.Parameters[0].Value.String())
}

var workflowCallTemplateWithInline = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-call-inline-iterated
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: process
            templateRef:
              name: test-inline-iterated
              template: main`

var workflowTemplateWithInlineSteps = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-inline-iterated
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: iterated
            template: steps-inline
            arguments:
              parameters:
                - name: arg
                  value: "{{ item }}"
            withItems:
              - foo
              - bar

    - name: steps-inline
      inputs:
        parameters:
          - name: arg
      steps:
        - - name: inline-a
            arguments:
              parameters:
                - name: arg
                  value: "{{ inputs.parameters.arg }}"
            inline:
              inputs:
                parameters:
                  - name: arg
              container:
                image: docker/whalesay
                command: [echo]
                args:
                  - "{{ inputs.parameters.arg }} a"
              outputs:
                parameters:
                  - name: arg-out
                    value: "{{ inputs.parameters.arg }}"
          - name: inline-b
            arguments:
              parameters:
                - name: arg
                  value: "{{ inputs.parameters.arg }}"
            inline:
              inputs:
                parameters:
                  - name: arg
              container:
                image: docker/whalesay
                command: [echo]
                args:
                  - "{{ inputs.parameters.arg }} b"
              outputs:
                parameters:
                  - name: arg-out
                    value: "{{ inputs.parameters.arg }}"
`

func TestCallTemplateWithInlineSteps(t *testing.T) {
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(workflowTemplateWithInlineSteps)
	wf := wfv1.MustUnmarshalWorkflow(workflowCallTemplateWithInline)
	cancel, controller := newController(wf, wftmpl)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 4)
	count := 0
	for _, pod := range pods.Items {
		nodeName := pod.Annotations["workflows.argoproj.io/node-name"]
		if strings.Contains(nodeName, "foo") {
			count++
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "foo")
		}
		if strings.Contains(nodeName, "bar") {
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "bar")
		}
	}
	assert.Equal(t, 2, count)
	for name, storedTemplate := range woc.wf.Status.StoredTemplates {
		if strings.Contains(name, "inline-a") {
			assert.Equal(t, "{{ inputs.parameters.arg }} a", storedTemplate.Container.Args[0])
		}
		if strings.Contains(name, "inline-b") {
			assert.Equal(t, "{{ inputs.parameters.arg }} b", storedTemplate.Container.Args[0])
		}
	}
}

var workflowTemplateWithInlineDAG = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-inline-iterated
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: iterated
            template: dag-inline
            arguments:
              parameters:
                - name: arg
                  value: "{{ item }}"
            withItems:
              - foo
              - bar

    - name: dag-inline
      inputs:
        parameters:
          - name: arg
      dag:
        tasks:
          - name: inline-a
            arguments:
              parameters:
              - name: arg
                value: '{{ inputs.parameters.arg }}'
            inline:
              container:
                args:
                - '{{ inputs.parameters.arg }} a'
                command:
                - echo
                image: docker/whalesay
              inputs:
                parameters:
                - name: arg
              outputs:
                parameters:
                - name: arg-out
                  value: '{{ inputs.parameters.arg }}'

          - name: inline-b
            arguments:
              parameters:
              - name: arg
                value: '{{ inputs.parameters.arg }}'
            inline:
              container:
                args:
                - '{{ inputs.parameters.arg }} b'
                command:
                - echo
                image: docker/whalesay
              inputs:
                parameters:
                - name: arg
              outputs:
                parameters:
                - name: arg-out
                  value: '{{ inputs.parameters.arg }}'
`

func TestCallTemplateWithInlineDAG(t *testing.T) {
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(workflowTemplateWithInlineDAG)
	wf := wfv1.MustUnmarshalWorkflow(workflowCallTemplateWithInline)
	cancel, controller := newController(wf, wftmpl)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 4)
	count := 0
	for _, pod := range pods.Items {
		nodeName := pod.Annotations["workflows.argoproj.io/node-name"]
		if strings.Contains(nodeName, "foo") {
			count++
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "foo")
		}
		if strings.Contains(nodeName, "bar") {
			assert.Contains(t, pod.Spec.Containers[1].Args[0], "bar")
		}
	}
	assert.Equal(t, 2, count)
	for name, storedTemplate := range woc.wf.Status.StoredTemplates {
		if strings.Contains(name, "inline-a") {
			assert.Equal(t, "{{ inputs.parameters.arg }} a", storedTemplate.Container.Args[0])
		}
		if strings.Contains(name, "inline-b") {
			assert.Equal(t, "{{ inputs.parameters.arg }} b", storedTemplate.Container.Args[0])
		}
	}
}
