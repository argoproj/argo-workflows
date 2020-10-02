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
	assert.Equal(t, unmarshalWFTmpl(wfTmpl).Spec.WorkflowSpec.Templates, woc.execWf.Spec.Templates)
	assert.Equal(t, woc.wf.Spec.Entrypoint, woc.execWf.Spec.Entrypoint)
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
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: wfv1.Int64OrStringPtr("test"),
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
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: wfv1.Int64OrStringPtr("test"),
			},
		}
		wftmpl.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])

	})

	t.Run("CheckMergingWFDefaults", func(t *testing.T) {
		wfDefaultActiveS := int64(5)
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		controller.Config.WorkflowDefaults = &wfv1.Workflow{Spec: wfv1.WorkflowSpec{
			ActiveDeadlineSeconds: &wfDefaultActiveS,
		},
		}
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, wfDefaultActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)

	})
	t.Run("CheckMergingWFTandWF", func(t *testing.T) {
		wfActiveS := int64(10)
		wftActiveS := int64(10)
		wfDefaultActiveS := int64(5)

		wftmpl.Spec.ActiveDeadlineSeconds = &wftActiveS
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		controller.Config.WorkflowDefaults = &wfv1.Workflow{Spec: wfv1.WorkflowSpec{
			ActiveDeadlineSeconds: &wfDefaultActiveS,
		},
		}
		wf.Spec.ActiveDeadlineSeconds = &wfActiveS
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, wfActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)

		wf.Spec.ActiveDeadlineSeconds = nil
		woc = newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Equal(t, wftActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)
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
		cancel, controller := newController(wf)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		_, _, err := woc.loadExecutionSpec()
		assert.Error(t, err)
		woc.operate()
		assert.Equal(t, wfv1.NodeError, woc.wf.Status.Phase)
	})
}

var wftWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: params-test-1
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: a-a
        value: "10"
      - name: b
        value: ""
      - name: c-c
        value: "0"
      - name: d
        value: ""
      - name: e-e
        value: "10"
      - name: f
        value: ""
      - name: g-g
        value: "1"
      - name: h
        value: ""
      - name: i-i
        value: "{}"
      - name: things
        value: "[]"

  templates:
    - name: main
      steps:
        - - name: echoitems
            template: echo

    - name: echo
      container:
        image: busybox
        command: [echo]
        args: ["{{workflows.parameters.a-a}} = {{workflows.parameters.g-g}}"]
`
var wfWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: params-test-1-grx2n
  namespace: default
spec:
  arguments:
    parameters:
    - name: f
      value: f
    - name: g-g
      value: 2
    - name: h
      value: h
    - name: i-i
      value: '{}'
    - name: things
      value: '[{"a":"1","nested":{"B":"3"}},{"a":"2"}]'
    - name: a-a
      value: 5
  workflowTemplateRef:
    name: params-test-1
`

func TestWorkflowTemplateRefParamMerge(t *testing.T) {
	wf := unmarshalWF(wfWithParam)
	wftmpl := unmarshalWFTmpl(wftWithParam)

	t.Run("CheckArgumentFromWF", func(t *testing.T) {
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		_, _, err := woc.loadExecutionSpec()
		assert.NoError(t, err)
		assert.Equal(t, wf.Spec.Arguments.Parameters, woc.wf.Spec.Arguments.Parameters)
	})

}
