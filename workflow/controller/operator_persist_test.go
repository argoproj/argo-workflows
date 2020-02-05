package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/packer"
)

func getMockDBCtx(expectedError error, largeWfSupport bool) *mocks.OffloadNodeStatusRepo {
	mockDBRepo := &mocks.OffloadNodeStatusRepo{}
	mockDBRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return("my-offloaded-version", expectedError)
	mockDBRepo.On("IsEnabled").Return(largeWfSupport)
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
	defer makeMax()()
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	controller.offloadNodeStatusRepo = getMockDBCtx(fmt.Errorf("not found"), false)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, wf.Status.IsOffloadNodeStatus())
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	assert.NotEmpty(t, woc.wf.Status.Nodes)
	assert.Empty(t, woc.wf.Status.CompressedNodes)
}

// TestPersistWithoutLargeWfSupport verifies persistence error with no largeWFsuppport
func TestPersistErrorWithoutLargeWfSupport(t *testing.T) {
	defer makeMax()()
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	controller.offloadNodeStatusRepo = getMockDBCtx(errors.New("23324", "test"), false)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, wf.Status.IsOffloadNodeStatus())
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	assert.NotEmpty(t, woc.wf.Status.Nodes)
	assert.Empty(t, woc.wf.Status.CompressedNodes)
}

// TestPersistWithoutLargeWfSupport verifies persistence with largeWFsuppport
func TestPersistWithLargeWfSupport(t *testing.T) {
	defer makeMax()()
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	controller.offloadNodeStatusRepo = getMockDBCtx(nil, true)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	// check the saved version has been offloaded
	assert.True(t, wf.Status.IsOffloadNodeStatus())
	assert.Empty(t, wf.Status.Nodes)
	assert.Empty(t, wf.Status.CompressedNodes)
	// check the updated in-memory version is pre-offloaded state
	assert.True(t, woc.wf.Status.IsOffloadNodeStatus())
	assert.NotEmpty(t, woc.wf.Status.Nodes)
	assert.Empty(t, woc.wf.Status.CompressedNodes)
}

// TestPersistWithoutLargeWfSupport verifies persistence error with largeWFsuppport
func TestPersistErrorWithLargeWfSupport(t *testing.T) {
	defer makeMax()()
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(helloWorldWfPersist)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	controller.offloadNodeStatusRepo = getMockDBCtx(errors.New("23324", "test"), true)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, woc.wf.Status.Phase)
	// check the saved version has not been offloaded
	assert.False(t, wf.Status.IsOffloadNodeStatus())
	assert.NotEmpty(t, woc.wf.Status.Nodes)
	assert.Empty(t, woc.wf.Status.CompressedNodes)
	// check the updated in-memory version is pre-offloaded state
	assert.False(t, woc.wf.Status.IsOffloadNodeStatus())
	assert.NotEmpty(t, woc.wf.Status.Nodes)
	assert.Empty(t, woc.wf.Status.CompressedNodes)
}

func makeMax() func() {
	return packer.SetMaxWorkflowSize(50)
}
