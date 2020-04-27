package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"

	//v1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
)

const wftWithVol = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: volumes-emptydir-template
spec:
  entrypoint: volumes-emptydir-example
  volumes:
  - name: workdir
    emptyDir: {}
  templates:
  - name: volumes-emptydir-example
    container:
      image: debian:latest
      command: ["/bin/bash", "-c"]
      args: ["sleep 30"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol
`
const wfRefVolWFT = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
  namespace: default
spec:
  entrypoint: volumes-emptydir-example
  arguments:
    parameters:
    - name: message
      value: "test"
  workflowTemplateRef:
    name: volumes-emptydir-template
`

func TestTopLevelWFTmplRef(t *testing.T) {
	controller := newController()
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)
	t.Run("ExecuteWorkflowWithTmplRef", func(t *testing.T) {
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.checkAndInitWorkflowTmplRef()
		woc.operate()
		assert.True(t, woc.hasTopLevelWFTmplRef)
		assert.NotNil(t, woc.topLevelWFTmplRef)
		assert.Equal(t, wftmpl.Name, woc.topLevelWFTmplRef.GetName())
	})

	t.Run("CheckArgumentPassing", func(t *testing.T) {
		value := "test"
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: &value,
			},
		}
		wf.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
		assert.NoError(t, err)
		_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.checkAndInitWorkflowTmplRef()
		woc.operate()
		assert.True(t, woc.hasTopLevelWFTmplRef)
		assert.NotNil(t, woc.topLevelWFTmplRef)
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])
		assert.Equal(t, wftmpl.Name, woc.topLevelWFTmplRef.GetName())
	})
	t.Run("CheckArgumentFromWFT", func(t *testing.T) {
		value := "test"
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: &value,
			},
		}
		wftmpl.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
		assert.NoError(t, err)
		_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.checkAndInitWorkflowTmplRef()
		woc.operate()
		assert.True(t, woc.hasTopLevelWFTmplRef)
		assert.NotNil(t, woc.topLevelWFTmplRef)
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])
		assert.Equal(t, wftmpl.Name, woc.topLevelWFTmplRef.GetName())
	})
}

func TestTopLevelWFTmplRefWithVol(t *testing.T) {
	controller := newController()
	wf := unmarshalWF(wfRefVolWFT)
	wftmpl := unmarshalWFTmpl(wftWithVol)
	t.Run("ExecuteWorkflowWithTmplRef", func(t *testing.T) {
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		woc.checkAndInitWorkflowTmplRef()
		woc.operate()
		assert.True(t, woc.hasTopLevelWFTmplRef)
		assert.NotNil(t, woc.topLevelWFTmplRef)
		assert.Equal(t, wftmpl.Name, woc.topLevelWFTmplRef.GetName())
		assert.Equal(t, wftmpl.Spec.Volumes[0].Name, woc.volumes[0].Name)
	})
}