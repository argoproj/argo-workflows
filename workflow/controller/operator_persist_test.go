package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
)

func getMockDBCtx(expectedError error, largeWfSupport bool) (*mocks.OffloadNodeStatusRepo, hydrator.Interface) {
	mockDBRepo := &mocks.OffloadNodeStatusRepo{}
	mockDBRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return("my-offloaded-version", expectedError)
	mockDBRepo.On("Get", mock.Anything, mock.Anything).Return(wfv1.Nodes{"my-node": wfv1.NodeStatus{}}, nil)
	mockDBRepo.On("IsEnabled").Return(largeWfSupport)
	return mockDBRepo, hydrator.New(mockDBRepo)
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
	cleanup := makeMax()
	defer cleanup()
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfPersist)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	controller.offloadNodeStatusRepo, controller.hydrator = getMockDBCtx(fmt.Errorf("not found"), false)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, wf.Status.IsOffloadNodeStatus())
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
}

// TestPersistErrorWithoutLargeWfSupport verifies persistence error with no largeWFsuppport
func TestPersistErrorWithoutLargeWfSupport(t *testing.T) {
	cleanup := makeMax()
	defer cleanup()
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfPersist)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	controller.offloadNodeStatusRepo, controller.hydrator = getMockDBCtx(errors.New("23324", "test"), false)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowError, wf.Status.Phase)
}

// TestPersistWithLargeWfSupport verifies persistence with largeWFsuppport
func TestPersistWithLargeWfSupport(t *testing.T) {
	cleanup := makeMax()
	defer cleanup()
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfPersist)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	controller.offloadNodeStatusRepo, controller.hydrator = getMockDBCtx(nil, true)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	// check the saved version has been offloaded
	assert.True(t, wf.Status.IsOffloadNodeStatus())
	assert.Empty(t, wf.Status.Nodes)
	assert.Empty(t, wf.Status.CompressedNodes)
	// check the updated in-memory version is pre-offloaded state
	assert.False(t, woc.wf.Status.IsOffloadNodeStatus())
	assert.NotEmpty(t, woc.wf.Status.Nodes)
	assert.Empty(t, woc.wf.Status.CompressedNodes)
}

// TestPersistErrorWithLargeWfSupport verifies persistence error with largeWFsuppport
func TestPersistErrorWithLargeWfSupport(t *testing.T) {
	cleanup := makeMax()
	defer cleanup()
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfPersist)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	controller.offloadNodeStatusRepo, controller.hydrator = getMockDBCtx(errors.New("23324", "test"), true)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
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
