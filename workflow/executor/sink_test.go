package executor

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// newTestSink returns a sink over a fake task-result client and records
// what it writes to the termination log.
func newTestSink(t *testing.T) (context.Context, *taskResultSink, *[]string) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	tr, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	p := &Process{
		taskResultClient: argofake.NewClientset().ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace),
		Tracing:          tr,
	}
	s := p.NewTaskResultSink(TaskConfig{
		WorkflowName: fakeWorkflow,
		WorkflowUID:  types.UID(fakeWorkflowUID),
		NodeID:       fakeNodeID,
	}).(*taskResultSink)
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
	ctx, s, _ := newTestSink(t)
	require.NoError(t, s.Start(ctx))
	r := getTaskResult(ctx, t, s)
	assert.Equal(t, fakeWorkflow, r.Labels[common.LabelKeyWorkflow])
	assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])
	require.Len(t, r.OwnerReferences, 1)
	assert.Equal(t, fakeWorkflow, r.OwnerReferences[0].Name)
}

func TestTaskResultSink_Create(t *testing.T) {
	for _, tt := range []struct {
		name       string
		instanceID string
		labels     map[string]string
	}{
		{"WithoutInstanceID", "", map[string]string{
			common.LabelKeyWorkflow:               fakeWorkflow,
			common.LabelKeyReportOutputsCompleted: "false",
		}},
		{"WithInstanceID", "my-instance", map[string]string{
			common.LabelKeyWorkflow:               fakeWorkflow,
			common.LabelKeyReportOutputsCompleted: "false",
			common.LabelKeyControllerInstanceID:   "my-instance",
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, s, _ := newTestSink(t)
			s.instanceID = tt.instanceID
			require.NoError(t, s.create(ctx, wfv1.NodeResult{Progress: "1/2"}))
			assert.True(t, s.created)

			result := getTaskResult(ctx, t, s)
			assert.Equal(t, &wfv1.WorkflowTaskResult{
				TypeMeta: metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskResultKind},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fakeNodeID,
					Namespace: fakeNamespace,
					Labels:    tt.labels,
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: workflow.APIVersion, Kind: workflow.WorkflowKind, Name: fakeWorkflow, UID: fakeWorkflowUID,
					}},
				},
				NodeResult: wfv1.NodeResult{Progress: "1/2"},
			}, result)
		})
	}
}

func TestTaskResultSink_ReportThenFinalize(t *testing.T) {
	ctx, s, _ := newTestSink(t)
	require.NoError(t, s.Start(ctx))
	require.NoError(t, s.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Result: new("r")}}))
	require.NoError(t, s.Finalize(ctx))
	r := getTaskResult(ctx, t, s)
	require.NotNil(t, r.Outputs)
	assert.Equal(t, "r", *r.Outputs.Result)
	assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
}

func TestTaskResultSink_PodFailDoesNotTouchResult(t *testing.T) {
	ctx, s, terminated := newTestSink(t)
	require.NoError(t, s.Start(ctx))
	s.Fail(ctx, errors.New("boom"))
	assert.Equal(t, []string{"boom"}, *terminated)
	r := getTaskResult(ctx, t, s)
	assert.Empty(t, r.Phase)
	assert.Empty(t, r.Message)
}

// The deadline and progress goroutines report alongside the main flow, so
// the sink must be safe for concurrent use.
func TestTaskResultSink_ConcurrentUse(t *testing.T) {
	ctx, s, _ := newTestSink(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			assert.NoError(t, s.Report(ctx, wfv1.NodeResult{Progress: "1/2"}))
		})
	}
	wg.Wait()
	r := getTaskResult(ctx, t, s)
	assert.Equal(t, wfv1.Progress("1/2"), r.Progress)
}
