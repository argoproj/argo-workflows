package estimation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestDurationEstimator(t *testing.T) {
	startedAt := metav1.Time{}
	finishedAt := metav1.Time{Time: time.Time{}.Add(time.Second)}
	p := DurationEstimator{
		&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
		},
		&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-baseline"},
			Status: wfv1.WorkflowStatus{
				StartedAt:  startedAt,
				FinishedAt: finishedAt,
				Nodes: map[string]wfv1.NodeStatus{
					"my-baseline":           {StartedAt: startedAt, FinishedAt: finishedAt},
					"my-baseline-873244444": {StartedAt: startedAt, FinishedAt: finishedAt},
				},
			},
		},
	}
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateWorkflowDuration())
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateNodeDuration("my-wf"))
	assert.Equal(t, wfv1.EstimatedDuration(1), p.EstimateNodeDuration("1"))
}
