package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// newTestSink returns a sink over a fake task-result client and records
// what it writes to the termination log.
func newTestSink(t *testing.T, recordFailure bool) (context.Context, *taskResultSink, *[]string) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	tr, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	p := &Process{
		taskResultClient:          argofake.NewClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace),
		Tracing:                   tr,
		taskResultRetryTimeout:    time.Minute,
		interimResultRetryTimeout: time.Minute,
		taskResultBackoff:         defaultTaskResultBackoff,
	}
	s := p.NewTaskResultSink(TaskConfig{
		WorkflowName: fakeWorkflow,
		WorkflowUID:  types.UID(fakeWorkflowUID),
		NodeID:       fakeNodeID,
	}, recordFailure).(*taskResultSink)
	var terminated []string
	s.writeTerminate = func(msg string) { terminated = append(terminated, msg) }
	return ctx, s, &terminated
}

func getTaskResult(ctx context.Context, t *testing.T, s *taskResultSink) *wfv1.WorkflowTaskResult {
	t.Helper()
	r, err := s.client.Get(ctx, fakeNodeID, metav1.GetOptions{})
	require.NoError(t, err)
	return r
}

func TestTaskResultSink_StartCreatesPlaceholder(t *testing.T) {
	ctx, s, _ := newTestSink(t, false)
	require.NoError(t, s.Start(ctx))
	r := getTaskResult(ctx, t, s)
	assert.Equal(t, fakeWorkflow, r.Labels[common.LabelKeyWorkflow])
	assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])
	require.Len(t, r.OwnerReferences, 1)
	assert.Equal(t, fakeWorkflow, r.OwnerReferences[0].Name)
}

func TestTaskResultSink_ReportThenFinalize(t *testing.T) {
	ctx, s, _ := newTestSink(t, false)
	require.NoError(t, s.Start(ctx))
	require.NoError(t, s.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Result: new("r")}}))
	assert.Nil(t, getTaskResult(ctx, t, s).Outputs, "Report leaves the write to Finalize")
	require.NoError(t, s.Finalize(ctx))
	r := getTaskResult(ctx, t, s)
	require.NotNil(t, r.Outputs)
	assert.Equal(t, "r", *r.Outputs.Result)
	assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
}

func TestTaskResultSink_PodFailDoesNotTouchResult(t *testing.T) {
	ctx, s, terminated := newTestSink(t, false)
	require.NoError(t, s.Start(ctx))
	s.Fail(ctx, errors.New("boom"))
	assert.Equal(t, []string{"boom"}, *terminated)
	r := getTaskResult(ctx, t, s)
	assert.Empty(t, r.Phase)
	assert.Empty(t, r.Message)
}

func TestTaskResultSink_WorkerFirstFailureWins(t *testing.T) {
	ctx, s, terminated := newTestSink(t, true)
	s.Fail(ctx, errors.New("Step exceeded its deadline"))
	s.Fail(ctx, errors.New("signal: terminated"))
	require.NoError(t, s.Finalize(ctx))
	r := getTaskResult(ctx, t, s)
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
	assert.Equal(t, "Step exceeded its deadline", r.Message)
	assert.Empty(t, *terminated)
}

// The deadline and progress goroutines report alongside the main flow, so
// the sink must be safe for concurrent use, and still keep the first
// failure in worker mode.
func TestTaskResultSink_ConcurrentUse(t *testing.T) {
	ctx, s, _ := newTestSink(t, true)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			s.Fail(ctx, fmt.Errorf("failure %d", i))
			assert.NoError(t, s.Report(ctx, wfv1.NodeResult{Progress: "1/2"}))
		})
	}
	wg.Wait()
	require.NoError(t, s.Finalize(ctx))
	r := getTaskResult(ctx, t, s)
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
}

// Reports merge as successive merge patches of the task result would:
// objects merge, and a later value or list replaces an earlier one.
func TestTaskResultSink_ReportsMerge(t *testing.T) {
	ctx, s, _ := newTestSink(t, false)
	require.NoError(t, s.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{
		Result:     new("first"),
		Parameters: []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr("v")}},
	}}))
	require.NoError(t, s.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{
		Result:    new("second"),
		Artifacts: wfv1.Artifacts{{Name: "a"}},
	}}))
	require.NoError(t, s.Finalize(ctx))
	r := getTaskResult(ctx, t, s)
	require.NotNil(t, r.Outputs)
	assert.Equal(t, "second", *r.Outputs.Result)
	require.Len(t, r.Outputs.Parameters, 1)
	assert.Equal(t, "v", r.Outputs.Parameters[0].Value.String())
	require.Len(t, r.Outputs.Artifacts, 1)
	assert.Equal(t, "a", r.Outputs.Artifacts[0].Name)
}

// A worker's failure and outputs reach the API server in the write that
// completes the result, so the controller never sees it complete without
// the failure.
func TestTaskResultSink_WorkerFailureInCompletingWrite(t *testing.T) {
	ctx, s, _ := newTestSink(t, true)
	require.NoError(t, s.Start(ctx))
	require.NoError(t, s.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Result: new("r")}}))
	s.Fail(ctx, errors.New("cleanup failed"))
	r := getTaskResult(ctx, t, s)
	assert.Empty(t, r.Phase, "Fail leaves the write to Finalize")
	assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])

	require.NoError(t, s.Finalize(ctx))
	r = getTaskResult(ctx, t, s)
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
	assert.Equal(t, "cleanup failed", r.Message)
	require.NotNil(t, r.Outputs)
	assert.Equal(t, "r", *r.Outputs.Result)
	assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
}

// A failure reported after Finalize, which only an error from Finalize
// itself can cause, is written straight away.
func TestTaskResultSink_WorkerFailureAfterFinalize(t *testing.T) {
	ctx, s, _ := newTestSink(t, true)
	require.NoError(t, s.Start(ctx))
	require.NoError(t, s.Finalize(ctx))
	s.Fail(ctx, errors.New("late"))
	r := getTaskResult(ctx, t, s)
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
	assert.Equal(t, "late", r.Message)
}

// When the API server rejects a worker's result, Finalize still completes
// it and records the rejection as the task's failure.
func TestTaskResultSink_WorkerRejectedResult(t *testing.T) {
	ctx, s, _ := newTestSink(t, true)
	require.NoError(t, s.Start(ctx))
	clientset := argofake.NewClientset()
	_, err := clientset.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace).Create(ctx, getTaskResult(ctx, t, s), metav1.CreateOptions{})
	require.NoError(t, err)
	clientset.PrependReactor("patch", "workflowtaskresults", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		if strings.Contains(string(action.(k8stesting.PatchAction).GetPatch()), `"outputs"`) {
			return true, nil, apierr.NewRequestEntityTooLargeError("too big")
		}
		return false, nil, nil
	})
	s.client = clientset.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
	require.NoError(t, s.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Result: new("r")}}))

	err = s.Finalize(ctx)
	require.Error(t, err)
	r := getTaskResult(ctx, t, s)
	assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
	assert.Equal(t, wfv1.NodeFailed, r.Phase)
	assert.Contains(t, r.Message, "failed to report outputs: ")
	assert.Nil(t, r.Outputs)
}
