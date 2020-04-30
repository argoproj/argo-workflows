package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
)

func TestTopLevelWFTmplRef(t *testing.T) {
	//_, controller := newController()
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	t.Run("ExecuteWorkflowWithTmplRef", func(t *testing.T) {
		_, controller := newController(wf, wftmpl)
		woc := newWorkflowOperationCtx(wf, controller)
		err := woc.setWorkflowSpecAndEntrypoint()
		assert.NoError(t, err)
		woc.operate()
		assert.Equal(t, &wftmpl.Spec.WorkflowSpec, woc.wfSpec)
	})
}

func TestTopLevelWFTmplRefWithArgs(t *testing.T) {
	//_, controller := newController()
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
		_, controller := newController(wf, wftmpl)
		woc := newWorkflowOperationCtx(wf, controller)
		err := woc.setWorkflowSpecAndEntrypoint()
		assert.NoError(t, err)
		woc.operate()
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])
	})

}
func TestTopLevelWFTmplRefWithWFTArgs(t *testing.T) {
	//_, controller := newController()
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
		_, controller := newController(wf, wftmpl)
		woc := newWorkflowOperationCtx(wf, controller)
		err := woc.setWorkflowSpecAndEntrypoint()
		assert.NoError(t, err)
		woc.operate()
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])

	})
}