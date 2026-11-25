package controller

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// workflowReconcileTiming tracks a workflow's reconciliation cost and last finish
// time so completion events can leave enough idle time between reconciliations.
type workflowReconcileTiming struct {
	duration time.Duration // Smoothed reconciliation duration, excluding workflow lock wait.
	lastEnd  time.Time     // End of the most recent reconciliation.
}

// recordWorkflowReconcile records utilization and updates the workflow's smoothed
// reconciliation duration when completion-triggered scheduling is enabled. The
// caller must measure start after acquiring the workflow lock and reading the
// cached workflow, before conversion, copying, hydration and operate(). Sharing the
// lifecycle lock prevents finished or deleted workflows from regaining timing state.
func (wfc *WorkflowController) recordWorkflowReconcile(ctx context.Context, wf *wfv1.Workflow, start, end time.Time) {
	if wfc.minCompletionRequeueDelay == nil {
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
// a completion arriving during reconciliation counts the in-flight time since lastEnd
// toward the idle gap and can therefore request too short a wait.
func (wfc *WorkflowController) completionRequeueDelay(uid types.UID, now time.Time) time.Duration {
	wfc.lastWrittenVersions.mutex.RLock()
	last := wfc.reconcileTimings[uid]
	wfc.lastWrittenVersions.mutex.RUnlock()

	floor := wfc.minCompletionDelay()
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

// minCompletionDelay is the shortest wait after a node completion:
// COMPLETED_REQUEUE_TIME_MS, capped at DEFAULT_REQUEUE_TIME. If
// COMPLETED_REQUEUE_TIME_MS is not set, it is DEFAULT_REQUEUE_TIME.
func (wfc *WorkflowController) minCompletionDelay() time.Duration {
	if wfc.minCompletionRequeueDelay == nil {
		return wfc.requeueTime
	}
	return min(*wfc.minCompletionRequeueDelay, wfc.requeueTime)
}

// workflowOwnerUID returns the UID of the owning Workflow. Argo always sets this
// owner reference; objects without one (tests, or created outside Argo) return "",
// which has no reconcile timing, so completionRequeueDelay uses the floor.
func workflowOwnerUID(refs []metav1.OwnerReference) types.UID {
	for _, owner := range refs {
		if owner.Kind == workflow.WorkflowKind {
			return owner.UID
		}
	}
	return ""
}

// enqueueNodeCompletion enqueues a workflow whose node has just completed. With
// COMPLETED_REQUEUE_TIME_MS unset it behaves like any other event (AddRateLimited);
// otherwise it waits the duty-cycled completion delay for the owning workflow.
// The workqueue coalesces pending keys and serializes work per key, and
// processNextItem still rejects stale workflow versions with a rate-limited retry.
func (wfc *WorkflowController) enqueueNodeCompletion(ctx context.Context, key string, owners []metav1.OwnerReference) {
	if wfc.minCompletionRequeueDelay == nil {
		wfc.wfQueue.AddRateLimited(key)
		return
	}
	delay := wfc.completionRequeueDelay(workflowOwnerUID(owners), time.Now())
	wfc.metrics.RecordWorkflowCompletionRequeueDelay(ctx, delay.Seconds())
	wfc.outdatedCompletionRetries.markEnqueuedByCompletion(key)
	wfc.wfQueue.AddAfter(key, delay)
}
