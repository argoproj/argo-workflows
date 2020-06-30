package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
)

func TestWorkflowTemplateRef(t *testing.T) {
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	t.Run("ExecuteWorkflowWithTmplRef", func(t *testing.T) {
		_, controller := newController(wf, wftmpl)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, &wftmpl.Spec.WorkflowSpec, woc.wfSpec)
	})
}

func TestWorkflowTemplateRefWithArgs(t *testing.T) {
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	t.Run("CheckArgumentPassing", func(t *testing.T) {
		value := intstr.Parse("test")
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: &value,
			},
		}
		wf.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		_, controller := newController(wf, wftmpl)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])
	})

}
func TestWorkflowTemplateRefWithWorkflowTemplateArgs(t *testing.T) {
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	t.Run("CheckArgumentFromWFT", func(t *testing.T) {
		value := intstr.Parse("test")
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: &value,
			},
		}
		wftmpl.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		_, controller := newController(wf, wftmpl)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])

	})
}

const wfWithStatus = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-whalesay-template-
  namespace: argo
spec:
  arguments:
    parameters:
    - name: param1
      value: test
  entrypoint: whalesay-template
  workflowTemplateRef:
    name: workflow-template-whalesay-template
status:
  startedAt: "2020-05-01T01:04:41Z"
  storedTemplates:
    namespaced/workflow-template-whalesay-template/whalesay-template:
      arguments: {}
      container:
        args:
        - '{{inputs.parameters.message}}'
        command:
        - cowsay
        image: docker/whalesay
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: message
      metadata: {}
      name: whalesay-template
      outputs: {}
  storedWorkflowTemplateSpec:
    arguments:
      parameters:
      - name: param2
        value: hello
    templates:
    - arguments: {}
      container:
        args:
        - '{{inputs.parameters.message}}'
        command:
        - cowsay
        image: docker/whalesay
        name: ""
        resources: {}
      inputs:
        parameters:
        - name: message
      metadata: {}
      name: whalesay-template
      outputs: {}
`

func TestWorkflowTemplateRefGetFromStored(t *testing.T) {
	wf := unmarshalWF(wfWithStatus)
	t.Run("ProcessWFWithStoredWFT", func(t *testing.T) {
		_, controller := newController(wf)
		woc := newWorkflowOperationCtx(wf, controller)
		_, execArgs, err := woc.loadExecutionSpec()
		assert.NoError(t, err)

		assert.Equal(t, "test", execArgs.Parameters[0].Value.String())
		assert.Equal(t, "hello", execArgs.Parameters[1].Value.String())
	})
}

const invalidWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ui-workflow-error
  namespace: argo
spec:
  entrypoint: main
  workflowTemplateRef:
    name: not-exists
`

func TestWorkflowTemplateRefInvalidWF(t *testing.T) {
	wf := unmarshalWF(invalidWF)
	t.Run("ProcessWFWithStoredWFT", func(t *testing.T) {
		_, controller := newController(wf)
		woc := newWorkflowOperationCtx(wf, controller)
		_, _, err := woc.loadExecutionSpec()
		assert.Error(t, err)
		woc.operate()
		assert.Equal(t, wfv1.NodeError, woc.wf.Status.Phase)
	})
}
