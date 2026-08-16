package estimation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
					"my-baseline":             {StartedAt: a, FinishedAt: b},
					"my-baseline-873244444":   {StartedAt: a, FinishedAt: b},
					"my-baseline-873244444.x": {StartedAt: a, FinishedAt: b},
				},
			},
		},
	}
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateWorkflowDuration())
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateNodeDuration(ctx, "my-wf"))
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateNodeDuration(ctx, "1"))
}
