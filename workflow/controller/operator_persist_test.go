package controller

import (
	"testing"

	"github.com/argoproj/argo/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func getMockDBCtx(expectedResullt interface{}, largeWfSupport bool, isInterfaceNil bool) sqldb.DBRepository {

	mockDBRepo := &mocks.DBRepository{}

	mockDBRepo.On("Save", mock.Anything).Return(expectedResullt)
	mockDBRepo.On("IsNodeStatusOffload").Return(largeWfSupport)
	mockDBRepo.On("IsInterfaceNil").Return(isInterfaceNil)
	return mockDBRepo
}

var helloWorldWfPersist = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metadata:
      annotations:
        annotationKey1: "annotationValue1"
        annotationKey2: "annotationValue2"
      labels:
        labelKey1: "labelValue1"
        labelKey2: "labelValue2"
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

// TestPersistWithoutLargeWfSupport verifies persistence with no largeWFsuppport
func TestPersistWithoutLargeWfSupport(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	if err != nil {

	}
	controller.wfDBctx = getMockDBCtx(sqldb.DBUpdateNoRowFoundError(nil, "test"), false, false)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.True(t, woc.wf.Status.Phase == wfv1.NodeRunning)

}

// TestPersistWithoutLargeWfSupport verifies persistence error with no largeWFsuppport
func TestPersistErrorWithoutLargeWfSupport(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	if err != nil {

	}
	var err1 error = errors.New("23324", "test")
	controller.wfDBctx = getMockDBCtx(sqldb.DBUpdateNoRowFoundError(err1, "test"), false, false)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	assert.True(t, woc.wf.Status.Phase == wfv1.NodeRunning)

}

// TestPersistWithoutLargeWfSupport verifies persistence with largeWFsuppport
func TestPersistWithLargeWfSupport(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	if err != nil {

	}
	controller.wfDBctx = getMockDBCtx(sqldb.DBUpdateNoRowFoundError(nil, "test"), true, true)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.True(t, woc.wf.Status.Phase == wfv1.NodeRunning)

}

// TestPersistWithoutLargeWfSupport verifies persistence error with largeWFsuppport
func TestPersistErrorWithLargeWfSupport(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	if err != nil {

	}
	var err1 error = errors.New("23324", "test")
	controller.wfDBctx = getMockDBCtx(sqldb.DBUpdateNoRowFoundError(err1, "test"), true, false)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	assert.True(t, woc.wf.Status.Phase == wfv1.NodeFailed)

}
