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

// workflowReconcileTiming tracks a workflow's reconciliation cost and last finish
// time so completion events can leave enough idle time between reconciliations.
type workflowReconcileTiming struct {
	duration time.Duration // Smoothed operate() duration, excluding workflow lock wait.
	lastEnd  time.Time     // End of the most recent reconciliation.
}

// lookupCompletionDutyCyclePercent reads the target percentage of one worker's
// time per workflow. It defaults to 20 and rejects values outside 1–100.
func lookupCompletionDutyCyclePercent() (int, error) {
	value := env.LookupEnvStringOr("COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT", "20")
	percent, err := strconv.Atoi(value)
	if err != nil || percent < 1 || percent > 100 {
		return 0, fmt.Errorf("COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT must be an integer between 1 and 100; got %q", value)
	}
	return percent, nil
}

// recordWorkflowReconcile records utilization and updates the workflow's smoothed
// reconciliation duration when completion-triggered scheduling is enabled. The
// caller must measure start after acquiring the workflow lock. Sharing the
// lifecycle lock prevents finished or deleted workflows from regaining timing state.
func (wfc *WorkflowController) recordWorkflowReconcile(ctx context.Context, wf *wfv1.Workflow, start, end time.Time) {
	if wfc.taskSetCompletionDelay == nil {
		return
	}
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

// completionRequeueDelay returns the remaining wait needed to meet the workflow's
// duty-cycle target, bounded by the completion floor and normal requeue ceiling.
// Missing timing uses the floor; the ceiling wins if it is lower than the floor.
// The caller must have enabled the floor. This is not a hard quota: an earlier
// queued event can still reconcile sooner. Timing includes only finished reconciles;
// a completion arriving during operate() counts the in-flight time since lastEnd
// toward the idle gap and can therefore request too short a wait.
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
