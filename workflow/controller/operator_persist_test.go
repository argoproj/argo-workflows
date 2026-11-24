package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
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

// TestPersistUpdatesMarksReapplyFailedOnNonConflictError verifies that a non-conflict
// Update error (e.g. connection reset) marks the workflow reapply-failed in memory, so the
// controller's throttler defer keeps the parallelism slot instead of releasing it while the
// in-memory workflow may not match the API object.
func TestPersistUpdatesMarksReapplyFailedOnNonConflictError(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfPersist)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	controller.offloadNodeStatusRepo, controller.hydrator = getMockDBCtx(nil, false)

	// Make the workflow Update fail with a non-conflict error.
	controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierr.NewInternalError(fmt.Errorf("connection reset by peer"))
	})

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.updated = true // force persistUpdates to attempt the Update
	woc.persistUpdates(ctx)

	assert.True(t, woc.reapplyFailed, "non-conflict persist error should mark reapply-failed to keep the throttler slot")
}

// TestPhaseCounterNotDoubleCountedOnFailedUpdate verifies that total_count only counts a
// workflow phase transition once the update carrying it has been persisted: an Update that
// fails must not count the transition, and the later reconcile that persists the same
// transition must count it exactly once.
// https://github.com/argoproj/argo-workflows/issues/17571
func TestPhaseCounterNotDoubleCountedOnFailedUpdate(t *testing.T) {
	for _, tt := range []struct {
		name     string
		failWith func() error
	}{
		{
			name: "non-conflict error",
			failWith: func() error {
				return apierr.NewInternalError(fmt.Errorf("connection reset by peer"))
			},
		},
		{
			name: "conflict",
			failWith: func() error {
				return apierr.NewConflict(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, "hello-world", fmt.Errorf("the object has been modified"))
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cancel, controller := newController(logging.TestContext(t.Context()))
			defer cancel()

			ctx := logging.TestContext(t.Context())
			wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
			wf := wfv1.MustUnmarshalWorkflow(helloWorldWfPersist)
			wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
			require.NoError(t, err)

			controller.offloadNodeStatusRepo, controller.hydrator = getMockDBCtx(nil, false)

			// Fail workflow Updates until the retry below, then let them through.
			failing := true
			controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
				if failing {
					return true, nil, tt.failWith()
				}
				return false, nil, nil
			})

			runningCount := func() int64 {
				t.Helper()
				attribs := attribute.NewSet(attribute.String("phase", "Running"), attribute.String("namespace", wf.Namespace))
				val, metricErr := testExporter.GetInt64CounterValue(ctx, "total_count", &attribs)
				if metricErr != nil {
					return 0
				}
				return val
			}

			// First reconcile: the workflow enters Running but the update fails, so the
			// transition never reaches the API server and must not be counted.
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
			assert.Equal(t, int64(0), runningCount(), "a phase transition whose update failed must not be counted")

			// Second reconcile starts from the stored phase, makes the same transition
			// again and this time persists it: it must be counted exactly once.
			failing = false
			stored, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, wfv1.WorkflowUnknown, stored.Status.Phase)

			woc = newWorkflowOperationCtx(ctx, stored, controller)
			woc.operate(ctx)
			require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
			assert.Equal(t, int64(1), runningCount(), "the persisted phase transition must be counted exactly once")
		})
	}
}
