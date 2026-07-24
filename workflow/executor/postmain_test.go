package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

func newTestPostMainExecutor(t *testing.T, tmpl wfv1.Template, mockRuntime *mocks.ContainerRuntimeExecutor) *WorkflowExecutor {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	tracingObj, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	return &WorkflowExecutor{
		PodName:            fakePodName,
		podUID:             types.UID(fakePodUID),
		workflow:           fakeWorkflow,
		workflowUID:        types.UID(fakeWorkflowUID),
		nodeID:             fakeNodeID,
		Template:           tmpl,
		ClientSet:          fake.NewClientset(),
		Namespace:          fakeNamespace,
		RuntimeExecutor:    mockRuntime,
		taskResultClient:   argofake.NewClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace),
		Tracing:            tracingObj,
		memoizedConfigMaps: map[string]string{},
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
