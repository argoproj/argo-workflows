package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopLevelWFTmplRef(t *testing.T) {
	controller := newController()
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)

	_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	woc.operate()
}
