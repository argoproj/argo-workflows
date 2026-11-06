package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// pooledTask is a ledger for task my-wf-main-1, enqueued and taken through
// steps: "claim" by pool-abc-1, "release" and "interrupt".
func pooledTask(steps ...string) *v1alpha1.PooledTaskStatus {
	task := &v1alpha1.PooledTaskStatus{TaskName: "my-wf-main-1"}
	now := metav1.Now()
	task.Enqueue(now)
	for _, step := range steps {
		switch step {
		case "claim":
			task.Claim("claim-uid", now, "pool-abc-1", "pod-uid", 0)
		case "release":
			task.Release(now)
		case "interrupt":
			task.Interrupt(now)
		}
	}
	return task
}

func TestPooledTaskPodName(t *testing.T) {
	claimedWithoutClaimant := pooledTask("claim")
	claimedWithoutClaimant.WorkerPodName = ""
	tests := []struct {
		name       string
		task       *v1alpha1.PooledTaskStatus
		ended      bool
		wantName   string
		wantMarker string
	}{
		{name: "enqueued", task: pooledTask(), wantName: "my-wf-main-1", wantMarker: PooledTaskQueued},
		{name: "released", task: pooledTask("claim", "release"), wantName: "my-wf-main-1", wantMarker: PooledTaskQueued},
		{name: "ended unclaimed", task: pooledTask(), ended: true, wantName: "my-wf-main-1", wantMarker: PooledTaskUnclaimed},
		{name: "ended after a release", task: pooledTask("claim", "release"), ended: true, wantName: "my-wf-main-1", wantMarker: PooledTaskReleased},
		{name: "claimed", task: pooledTask("claim"), wantName: "pool-abc-1"},
		{name: "re-claimed after a release", task: pooledTask("claim", "release", "claim"), wantName: "pool-abc-1"},
		{name: "interrupted", task: pooledTask("claim", "interrupt"), ended: true, wantName: "pool-abc-1"},
		{name: "claimed without a claimant", task: claimedWithoutClaimant, wantName: "my-wf-main-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, marker := PooledTaskPodName(tt.task, tt.ended)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantMarker, marker)
		})
	}
}
