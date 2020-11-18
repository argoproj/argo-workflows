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
				Value: wfv1.AnyStringPtr("test"),
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
				Value: wfv1.AnyStringPtr("test"),
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
		woc.operate()
		assert.Equal(t, wf.Spec.Arguments.Parameters, woc.wf.Spec.Arguments.Parameters)
	})

}

var wftWithArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: artifact-test-1
  namespace: test-namespace
spec:
  entrypoint: main
  arguments:
    artifacts:
    - name: binary-file
      http:
        url: https://a.server.io/file
    - name: data-file
      http:
        url: https://b.server.io/data

  templates:
    - name: main
      steps:
        - - name: process-data
            template: process

    - name: process
      inputs:
        artifacts:
          - name: binary-file
            path: /usr/local/bin/binfile
            mode: 0755
          - name: data-file
            path: /tmp/data
            mode: 0755
      container:
        image: busybox
        command: [sh, -c]
        args: ["binary-file /tmp/data"]
`

const wfWithTemplateWithArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-from-artifact-test-1-
  namespace: test-namespace
spec:
  arguments:
    artifacts:
    - name: own-file
      http:
        url: https://local/blob
  workflowTemplateRef:
    name: artifact-test-1
`

func TestWorkflowTemplateRefGetArtifactsFromTemplate(t *testing.T) {
	wf := unmarshalWF(wfWithTemplateWithArtifact)
	wftmpl := unmarshalWFTmpl(wftWithArtifact)

	t.Run("CheckArtifactArgumentFromWF", func(t *testing.T) {
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
		assert.Len(t, woc.execWf.Spec.Arguments.Artifacts, 3)

		assert.Equal(t, "own-file", woc.execWf.Spec.Arguments.Artifacts[0].Name)
		assert.Equal(t, "binary-file", woc.execWf.Spec.Arguments.Artifacts[1].Name)
		assert.Equal(t, "data-file", woc.execWf.Spec.Arguments.Artifacts[2].Name)
	})
}
