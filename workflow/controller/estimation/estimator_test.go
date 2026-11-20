package estimation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func Test_estimator(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	a := metav1.Time{}
	b := metav1.Time{Time: time.Time{}.Add(time.Second)}
	p := &estimator{
		&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			Status: wfv1.WorkflowStatus{
				Nodes: map[string]wfv1.NodeStatus{
					"my-wf":             {StartedAt: a, FinishedAt: a},
					"my-wy-873244444":   {StartedAt: a, FinishedAt: a},
					"my-wy-873244444.x": {StartedAt: a, FinishedAt: a},
				},
			},
		},
		&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-baseline"},
			Status: wfv1.WorkflowStatus{
				StartedAt:  a,
				FinishedAt: b,
				Nodes: map[string]wfv1.NodeStatus{
					"my-baseline":             {Name: "my-baseline", StartedAt: a, FinishedAt: b},
					"my-baseline-873244444":   {Name: "1", StartedAt: a, FinishedAt: b},
					"my-baseline-873244444.x": {StartedAt: a, FinishedAt: b},
				},
			},
		},
	}
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateWorkflowDuration())
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateNodeDuration(ctx, "my-wf"))
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateNodeDuration(ctx, "1"))
}

func Test_estimatorMissingBaselineNode(t *testing.T) {
	hook := logging.NewTestHook()
	ctx := logging.WithLogger(t.Context(), logging.NewTestLogger(logging.Debug, logging.Text, hook))
	p := &estimator{
		&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "my-wf"}},
		&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "my-baseline"}},
	}

	assert.Equal(t, wfv1.EstimatedDuration(0), p.EstimateNodeDuration(ctx, "my-wf.missing"))

	entries := hook.AllEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "was unable to obtain node for nodeName", entries[0].Msg)
	assert.Equal(t, "my-baseline.missing", entries[0].Fields["nodeName"])
	assert.Equal(t, logging.Debug, entries[0].Level, "a node missing from the baseline is expected and must not be logged as an error")
}
