package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
)

func TestWorkflowTemplateRef(t *testing.T) {
	cancel, controller := newController(unmarshalWF(wfWithTmplRef), unmarshalWFTmpl(wfTmpl))
	defer cancel()
	woc := newWorkflowOperationCtx(unmarshalWF(wfWithTmplRef), controller)
	woc.operate()
	assert.Equal(t, unmarshalWFTmpl(wfTmpl).Spec.WorkflowSpec.Templates, woc.wfSpec.Templates)
	assert.Equal(t, woc.wf.Spec.Entrypoint, woc.wfSpec.Entrypoint)
	// verify we copy these values
	assert.Len(t, woc.volumes, 1, "volumes from workflow template")
	// and these
	assert.Equal(t, "my-sa", woc.globalParams["workflow.serviceAccountName"])
	assert.Equal(t, "77", woc.globalParams["workflow.priority"])
}

func TestWorkflowTemplateRefWithArgs(t *testing.T) {
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	t.Run("CheckArgumentPassing", func(t *testing.T) {
		value := "test"
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: &value,
			},
		}
		wf.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])
	})

}
func TestWorkflowTemplateRefWithWorkflowTemplateArgs(t *testing.T) {
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	t.Run("CheckArgumentFromWFT", func(t *testing.T) {
		value := "test"
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: &value,
			},
		}
		wftmpl.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
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
		cancel, controller := newController(wf)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		_, execArgs, err := woc.loadExecutionSpec()
		assert.NoError(t, err)

		assert.Equal(t, "test", *execArgs.Parameters[0].Value)
		assert.Equal(t, "hello", *execArgs.Parameters[1].Value)
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
		cancel, controller := newController(wf)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		_, _, err := woc.loadExecutionSpec()
		assert.Error(t, err)
		woc.operate()
		assert.Equal(t, wfv1.NodeError, woc.wf.Status.Phase)
	})
}
