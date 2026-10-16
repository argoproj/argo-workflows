package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/env"
)

type workflowReconcileTiming struct {
	duration time.Duration
	lastEnd  time.Time
}

func lookupCompletionDutyCyclePercent() (int, error) {
	value := env.LookupEnvStringOr("COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT", "20")
	percent, err := strconv.Atoi(value)
	if err != nil || percent < 1 || percent > 100 {
		return 0, fmt.Errorf("COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT must be an integer between 1 and 100; got %q", value)
	}
	return percent, nil
}

func (wfc *WorkflowController) recordWorkflowReconcile(ctx context.Context, wf *wfv1.Workflow, start, end time.Time) {
	wfc.lastWrittenVersions.mutex.Lock()
	defer wfc.lastWrittenVersions.mutex.Unlock()
	wfc.expireCompletedVersions()

	last := wfc.reconcileTimings[wf.UID]
	duration := end.Sub(start)
	if !last.lastEnd.IsZero() && end.After(last.lastEnd) {
		// The first reconcile has no preceding interval to measure utilisation over.
		wfc.metrics.RecordWorkflowReconcileUtilization(ctx, float64(duration)/float64(end.Sub(last.lastEnd)))
	}
	if !reconciliationNeeded(wf) || !wfc.lastWrittenVersions.versions[wf.UID].completedAt.IsZero() {
		delete(wfc.reconcileTimings, wf.UID)
		return
	}
	if !last.lastEnd.IsZero() {
		// EWMA with alpha=0.2 dampens single-reconcile cost spikes.
		duration = last.duration + (duration-last.duration)/5
	}
	if wfc.reconcileTimings == nil {
		wfc.reconcileTimings = make(map[types.UID]workflowReconcileTiming)
	}
	wfc.reconcileTimings[wf.UID] = workflowReconcileTiming{duration: duration, lastEnd: end}
}

// completionRequeueDelay only changes completion-triggered scheduling; an earlier
// queued event can still reconcile sooner. The caller must have enabled the floor.
func (wfc *WorkflowController) completionRequeueDelay(uid types.UID, now time.Time) time.Duration {
	wfc.lastWrittenVersions.mutex.RLock()
	last := wfc.reconcileTimings[uid]
	wfc.lastWrittenVersions.mutex.RUnlock()

	floor := min(*wfc.taskSetCompletionDelay, wfc.requeueTime)
	if last.lastEnd.IsZero() {
		return floor
	}
	// Work in floating-point nanoseconds until clamping to avoid overflowing a
	// duration when a costly reconcile is combined with a low duty cycle.
	gap := float64(last.duration) * float64(100-wfc.completionDutyCyclePercent) / float64(wfc.completionDutyCyclePercent)
	delay := gap - float64(now.Sub(last.lastEnd))
	if delay >= float64(wfc.requeueTime) {
		return wfc.requeueTime
	}
	if delay <= float64(floor) {
		return floor
	}
	return time.Duration(delay)
}
