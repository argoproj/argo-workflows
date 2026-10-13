package executor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	typedv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	utilretry "github.com/argoproj/argo-workflows/v4/util/retry"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

func newTestPostMainExecutor(t *testing.T, tmpl wfv1.Template, mockRuntime *mocks.ContainerRuntimeExecutor) *WorkflowExecutor {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	tracingObj, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	return &WorkflowExecutor{
		Process: &Process{
			PodName:            fakePodName,
			podUID:             types.UID(fakePodUID),
			ClientSet:          fake.NewClientset(),
			Namespace:          fakeNamespace,
			RuntimeExecutor:    mockRuntime,
			taskResultClient:   argofake.NewClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace),
			Tracing:            tracingObj,
			memoizedConfigMaps: map[string]string{},
			retryBackoff:       utilretry.ExecutorRetry(ctx),
			// The defaults NewProcess would set.
			taskResultRetryTimeout:    utilretry.DefaultExecutorTaskResultRetryTimeout,
			interimResultRetryTimeout: defaultInterimResultRetryTimeout,
			taskResultBackoff:         defaultTaskResultBackoff,
		},
		workflow:    fakeWorkflow,
		workflowUID: types.UID(fakeWorkflowUID),
		nodeID:      fakeNodeID,
		Template:    tmpl,
	}
}

// TestPostMain_ResourceShortCircuit verifies that Resource templates take the
// short-circuit branch in PostMain: after Wait, ReportOutputsLogs runs and
// the function returns without attempting CaptureScriptResult, SaveParameters,
// or SaveArtifacts. The Resource branch is shared by both wait and supervisor
// callers — a regression here breaks both pod layouts.
func TestPostMain_ResourceShortCircuit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockRuntime := &mocks.ContainerRuntimeExecutor{}
	tmpl := wfv1.Template{
		Name: "resource-tmpl",
		Resource: &wfv1.ResourceTemplate{
			Action:   "get",
			Manifest: "apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n",
		},
	}
	we := newTestPostMainExecutor(t, tmpl, mockRuntime)

	mockRuntime.On("Wait", mock.Anything, []string{"main"}).Return(nil)

	err := we.PostMain(ctx, ctx, false)
	require.NoError(t, err)
	// CaptureScriptResult / SaveParameters / SaveArtifacts should NOT be
	// reached for Resource templates. We assert by checking that the
	// mock's GetOutputStream was never invoked (CaptureScriptResult
	// would call it). Resource templates also have no outputs to save.
	mockRuntime.AssertNotCalled(t, "GetOutputStream")
}

// TestPostMain_HappyPath verifies a simple Container template flows through
// the full post-main pipeline (no Resource short-circuit) without error.
// The template has no outputs so SaveArtifacts/SaveParameters/CaptureScriptResult
// are reached but exit cleanly.
func TestPostMain_HappyPath(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockRuntime := &mocks.ContainerRuntimeExecutor{}
	tmpl := wfv1.Template{
		Name: "container-tmpl",
	}
	we := newTestPostMainExecutor(t, tmpl, mockRuntime)

	mockRuntime.On("Wait", mock.Anything, []string{"main"}).Return(nil)

	err := we.PostMain(ctx, ctx, false)
	require.NoError(t, err)
	mockRuntime.AssertCalled(t, "Wait", mock.Anything, []string{"main"})
}

// TestPostMain_PreMainFailedSkipsOutputs verifies that when the supervisor's
// pre-main phase failed (preMainFailed=true), PostMain does NOT attempt to
// capture the script result, save output parameters, or save output artifacts.
// main never ran, so those calls would fail with confusing "file not found"
// errors on required artifacts. The template below has a required (non-optional)
// output artifact whose staging would call CopyFile on the runtime; asserting
// CopyFile/GetOutputStream are never invoked confirms the branch is skipped,
// and NoError confirms the missing required artifact does not surface.
func TestPostMain_PreMainFailedSkipsOutputs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockRuntime := &mocks.ContainerRuntimeExecutor{}
	tmpl := wfv1.Template{
		Name: "container-tmpl",
		Outputs: wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				// Required output artifact main never produced (base-image path).
				{Name: "out1", Path: "/tmp/out1"},
			},
		},
	}
	we := newTestPostMainExecutor(t, tmpl, mockRuntime)

	mockRuntime.On("Wait", mock.Anything, []string{"main"}).Return(nil)

	err := we.PostMain(ctx, ctx, true)
	require.NoError(t, err, "pre-main failure must not surface as a missing-output-artifact error")
	// SaveArtifacts (CopyFile) and CaptureScriptResult (GetOutputStream) must be
	// skipped entirely when pre-main failed.
	mockRuntime.AssertNotCalled(t, "CopyFile")
	mockRuntime.AssertNotCalled(t, "GetOutputStream")
}

// TestPostMain_WaitErrorIsAggregated verifies that a Wait error propagates
// into the executor's error list and surfaces via HasError, but does NOT
// short-circuit the rest of the pipeline — outputs from main are still
// captured opportunistically.
func TestPostMain_WaitErrorIsAggregated(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockRuntime := &mocks.ContainerRuntimeExecutor{}
	tmpl := wfv1.Template{Name: "container-tmpl"}
	we := newTestPostMainExecutor(t, tmpl, mockRuntime)

	mockRuntime.On("Wait", mock.Anything, []string{"main"}).Return(assert.AnError)

	err := we.PostMain(ctx, ctx, false)
	require.Error(t, err, "Wait error must surface via HasError")
}

// serialTaskResults wraps a task result client and records how many writes
// were in flight at once. Progress writes stall until their ctx ends, as
// they would against an API server that has stopped answering.
type serialTaskResults struct {
	typedv1alpha1.WorkflowTaskResultInterface
	progressStarted chan struct{}
	startOnce       sync.Once
	progressWrites  atomic.Int32

	mu          sync.Mutex
	inFlight    int
	maxInFlight int
}

func (c *serialTaskResults) begin() (end func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight++
	c.maxInFlight = max(c.maxInFlight, c.inFlight)
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.inFlight--
	}
}

func (c *serialTaskResults) stallProgress(ctx context.Context) error {
	c.progressWrites.Add(1)
	c.startOnce.Do(func() { close(c.progressStarted) })
	<-ctx.Done()
	return ctx.Err()
}

func (c *serialTaskResults) Create(ctx context.Context, tr *wfv1.WorkflowTaskResult, opts metav1.CreateOptions) (*wfv1.WorkflowTaskResult, error) {
	end := c.begin()
	defer end()
	if tr.Progress != "" {
		return nil, c.stallProgress(ctx)
	}
	return c.WorkflowTaskResultInterface.Create(ctx, tr, opts)
}

func (c *serialTaskResults) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*wfv1.WorkflowTaskResult, error) {
	end := c.begin()
	defer end()
	if bytes.Contains(data, []byte(`"progress"`)) {
		return nil, c.stallProgress(ctx)
	}
	return c.WorkflowTaskResultInterface.Patch(ctx, name, pt, data, opts, subresources...)
}

// TestPostMain_ProgressReportsStopBeforeFinalReports verifies that main
// exiting while a progress report is stuck retrying stops the progress
// monitor, and cuts off the stuck report, before the final reports write
// the same task result. Both Collect variants are covered; the pod layouts
// differ only in Prepare.
func TestPostMain_ProgressReportsStopBeforeFinalReports(t *testing.T) {
	for name, tmpl := range map[string]wfv1.Template{
		"Container": {Name: "container-tmpl"},
		"Resource":  {Name: "resource-tmpl", Resource: &wfv1.ResourceTemplate{Action: "get"}},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			mockRuntime := &mocks.ContainerRuntimeExecutor{}
			we := newTestPostMainExecutor(t, tmpl, mockRuntime)
			taskResults := &serialTaskResults{WorkflowTaskResultInterface: we.taskResultClient, progressStarted: make(chan struct{})}
			we.taskResultClient = taskResults
			we.annotationPatchTickDuration = time.Millisecond
			we.readProgressFileTickDuration = time.Millisecond
			we.progressFile = filepath.Join(t.TempDir(), "progress")
			require.NoError(t, os.WriteFile(we.progressFile, []byte("1/2\n"), 0o600))
			// main exits while a progress report is stuck.
			mockRuntime.On("Wait", mock.Anything, []string{"main"}).Run(func(mock.Arguments) {
				select {
				case <-taskResults.progressStarted:
				case <-time.After(10 * time.Second):
				}
			}).Return(nil)

			require.NoError(t, we.PostMain(ctx, ctx, false))
			we.FinalizeOutput(ctx)
			progressWrites := taskResults.progressWrites.Load()
			require.Positive(t, progressWrites, "no progress report was made")
			time.Sleep(20 * time.Millisecond) // many progress ticks
			assert.Equal(t, progressWrites, taskResults.progressWrites.Load(), "progress reports continued after Wait returned")
			assert.Equal(t, 1, taskResults.maxInFlight, "task result writes overlapped")
			assert.Empty(t, we.errors)
		})
	}
}
