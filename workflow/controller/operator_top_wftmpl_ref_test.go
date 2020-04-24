package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const wft1 =`
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
spec:
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`
const wf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  entrypoint: whalesay-template

  workflowTemplateRef:
    name: workflow-template-whalesay-template
`



func TestTopLevelWFTmplRef(t *testing.T){
	controller := newController()
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	woc.operate()
}
