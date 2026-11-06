package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/pool"
)

// Every reconcile, once the cached results have been applied and releases
// recorded, each unended pooled node is checked against the caches, so that a
// task whose worker or TaskSet is gone ends rather than being lost, and the
// workflow can complete. Informer lag is never read as either: the caches
// sync before any reconcile and never drop an object before its deletion, so
// only an object created moments ago can be missing from them while it
// exists, and that is waited for, for the grace recentlyStartedPodDuration
// gives, as podReconciliation waits for a pod that started recently. The
// result is read live before a claimant is recorded dead, since the result
// cache may lag behind what the claim became. A node still waiting for a
// worker also shows its pool's health. A claimed task past its lease is
// interrupted; an interrupted node is checkEndedPooledTasks's.

// checkPooledTasks checks every unended pooled node's task: a claimed one for
// its claimant, an enqueued one, including one released, for an unrecorded
// claim, its TaskSet and its pool's health.
func (woc *wfOperationCtx) checkPooledTasks(ctx context.Context) {
	woc.eachPooledNode(func(node *wfv1.NodeStatus) bool { return !node.Phase.Completed() }, func(node *wfv1.NodeStatus) {
		switch node.PooledTask.State() {
		case wfv1.PooledTaskClaimed:
			woc.checkClaimedTask(ctx, node)
		case wfv1.PooledTaskEnqueued:
			woc.checkEnqueuedTask(ctx, node)
		}
	})
}

// eachPooledNode calls f with each of the workflow's pooled nodes for which
// match is true, as the node is when f is called. Every node is matched
// before f is first called, so f may update nodes.
func (woc *wfOperationCtx) eachPooledNode(match func(*wfv1.NodeStatus) bool, f func(*wfv1.NodeStatus)) {
	var ids []string
	for id, node := range woc.wf.Status.Nodes {
		if isPooled(&node) && match(&node) {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		if node, err := woc.wf.Status.Nodes.Get(id); err == nil {
			f(node)
		}
	}
}

// checkClaimedTask ends the claimed node if its claimant can no longer run
// its task. The claimant was cached when the claim was recorded, so a
// claimant missing from the cache was deleted. A claim missing from the
// cache was deleted too; releasePooledTasks didn't take it for a release
// because the TaskSet isn't cached, so the node may have lost its TaskSet.
func (woc *wfOperationCtx) checkClaimedTask(ctx context.Context, node *wfv1.NodeStatus) {
	ledger := node.PooledTask
	if !woc.pooledClaimCached(node) {
		woc.checkTaskSetLost(ctx, node)
		return
	}
	pod := woc.claimantPod(ledger.WorkerPodName, ledger.WorkerPodUID)
	if why := claimantGone(pod, ledger.WorkerPodName, ledger.WorkerRestartCount); why != "" {
		woc.endDeadClaim(ctx, node, ledger.ClaimUID, why)
		return
	}
	woc.checkLease(ctx, node)
}

// pooledLease is when the lease of the task the ledger claims ends: slack
// after its worker stops it (pool.StopAt), from the deadlines the ledger
// records as dispatched and the claim's time. A task that isn't claimed, or
// has no deadline, has no lease, as a pod with no deadline runs for ever.
func pooledLease(ledger *wfv1.PooledTaskStatus, slack time.Duration) (time.Time, bool) {
	claimedAt := ledger.ClaimedAt()
	if claimedAt == nil {
		return time.Time{}, false
	}
	stop, _ := pool.StopAt(ledger.Deadline, ledger.ActiveDeadlineSeconds, claimedAt.Time)
	if stop.IsZero() {
		return time.Time{}, false
	}
	return stop.Add(slack), true
}

// checkLease interrupts the claimed node's task once its lease has ended:
// its worker should have stopped it at its deadline, so the worker is
// wedged. The node fails as one whose pod passed its deadline does. Until
// then the workflow is requeued for the lease's end. The claim's time is the
// API server's, so a skewed controller clock moves the end by the skew,
// which the slack dwarfs.
func (woc *wfOperationCtx) checkLease(ctx context.Context, node *wfv1.NodeStatus) {
	lease, ok := pooledLease(node.PooledTask, woc.leaseSlack())
	if !ok {
		return
	}
	if woc.requeueUntil(lease) {
		return
	}
	woc.log.WithFields(logging.Fields{"nodeID": node.ID, "lease": lease, "workerPod": node.PooledTask.WorkerPodName}).
		Info(ctx, "Pooled task's lease has ended")
	woc.interruptClaimedNode(ctx, node, wfv1.NodeFailed, common.DeadlineExceededMessage)
}

// claimantGone says why the claimant pod podName, as cached, can no longer
// run the task it claimed when its worker container had restarted restarts
// times, or is empty if it can: it was deleted or replaced (pod is nil), it
// has ended, as an evicted pod has Failed, or its worker container has
// restarted since and isn't running. A pod being deleted can still run the
// task: its worker completes or releases it.
//
// The worker never recovers a task from an earlier run: when it runs again
// it fails every claim its pod holds before it claims anything.
// So a restart since the claim is taken for the claim's end only once the
// worker container isn't running, as when it crash loops before it can fail
// the claim, and a running worker fails the claim, or made it. That lets
// restarts be lower than the claimant's count: the pool pod cache can still
// hold the pod's status from before the restart whose run made the claim,
// and a claimant not cached when its claim was recorded counts 0. Restarts
// are compared by count, so clock skew between nodes doesn't matter.
func claimantGone(pod *apiv1.Pod, podName string, restarts int32) string {
	switch {
	case pod == nil:
		return fmt.Sprintf("worker pod %s deleted", podName)
	case pod.Status.Phase == apiv1.PodFailed:
		why := []string{"worker pod " + podName + " failed"}
		for _, s := range []string{pod.Status.Reason, pod.Status.Message} {
			if s != "" {
				why = append(why, s)
			}
		}
		return strings.Join(why, ": ")
	case pod.Status.Phase == apiv1.PodSucceeded:
		return fmt.Sprintf("worker pod %s exited", podName)
	case workerRestartCount(pod) > restarts && !workerRunning(pod):
		return common.WorkerRestartedMessage(podName)
	}
	return ""
}

// checkEnqueuedTask checks an enqueued node: a released one's deadline
// (checkReleasedDeadline), a claim on its TaskSet that ingest hasn't
// recorded, because its claimant pod isn't cached yet, and otherwise its
// TaskSet and its pool's health.
func (woc *wfOperationCtx) checkEnqueuedTask(ctx context.Context, node *wfv1.NodeStatus) {
	if node.PooledTask.Released() && woc.checkReleasedDeadline(ctx, node) {
		return
	}
	ref := pooledTaskSetRef(node)
	if claim, ok := taskSetClaim(woc.controller.taskResultInformer.GetStore(), woc.wf.Namespace, ref); ok {
		woc.checkUnrecordedClaim(ctx, node, claim)
		return
	}
	if !woc.pooledTaskSetCached(ref) {
		woc.checkTaskSetLost(ctx, node)
		return
	}
	woc.applyPoolHealth(ctx, node)
}

// applyPoolHealth shows the health of its pool on an enqueued node waiting
// for a worker: while the pool is degraded the node stays Pending and its
// message carries the reason, and a pool that can't be built, which will
// never serve the task, ends the node as Error with the reason, as a pod that
// can't be created ends its node. The message is written only when it
// changes. Dispatch and every claim clear an enqueued node's message, so the
// pool's health is the only message it has, and it is cleared once the pool
// is OK or no longer known. Without pool health, nothing is shown. The pool controller
// requeues the workflows waiting on a pool when its health changes.
func (woc *wfOperationCtx) applyPoolHealth(ctx context.Context, node *wfv1.NodeStatus) {
	if woc.controller.poolHealth == nil {
		return
	}
	ledger := node.PooledTask
	health, known := woc.controller.poolHealth.Health(woc.wf.Namespace, ledger.PoolHash)
	log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "pool": ledger.PoolName, "poolHash": ledger.PoolHash, "health": health.State})
	message := ""
	switch {
	case known && health.State == dispatch.HealthUnbuildable:
		log.WithField("reason", health.Reason).Info(ctx, "Pooled task's pool can't be built")
		woc.endPooledNode(ctx, node, wfv1.NodeError, fmt.Sprintf("pool %q can't be built: %s", ledger.PoolName, health.Reason))
		return
	case known && health.State == dispatch.HealthDegraded:
		message = fmt.Sprintf("pool %q is degraded: %s", ledger.PoolName, health.Reason)
	}
	if node.Message == message {
		return
	}
	log.WithField("message", message).Info(ctx, "Pooled task's pool health changed")
	updated := node.DeepCopy()
	updated.Message = message
	woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
	woc.updated = true
}

// checkUnrecordedClaim checks claim, an incomplete claim on the enqueued
// node's TaskSet whose claimant pod isn't cached (a completed one would have
// ended the node). Until the grace has passed since the claim that is
// informer lag; then the pod is read live, and if it is gone the claimant
// died before the cache saw it. The claim's time is the API server's, so a
// skewed clock only moves the read; what the read finds decides.
func (woc *wfOperationCtx) checkUnrecordedClaim(ctx context.Context, node *wfv1.NodeStatus, claim dispatch.Claim) {
	if woc.requeueUntil(claim.CreatedAt.Add(recentlyStartedPodDuration(ctx))) {
		return
	}
	why, err := woc.claimantDeleted(ctx, claim.WorkerPodName, claim.WorkerPodUID)
	switch {
	case err != nil:
		woc.log.WithFields(logging.Fields{"nodeID": node.ID, "workerPod": claim.WorkerPodName}).WithError(err).
			Warn(ctx, "Failed to get the claimant pod of a pooled task")
		woc.requeue()
	case why != "":
		woc.endDeadClaim(ctx, node, claim.UID, why)
	}
}

// claimantDeleted reads the claimant pod name, which the pool pod cache
// doesn't hold, live, and says why it can no longer run its task if it was
// deleted or replaced, as claimantGone does. It is empty if the pod exists:
// it is still to be cached, which requeues the workflow.
func (woc *wfOperationCtx) claimantDeleted(ctx context.Context, name string, uid types.UID) (string, error) {
	pod, err := woc.getPod(ctx, name)
	switch {
	case apierr.IsNotFound(err) || err == nil && pod.UID != uid:
		return claimantGone(nil, name, 0), nil
	case err != nil:
		return "", err
	}
	return "", nil
}

// liveClaim reads the pooled node's result live, as the result cache may lag
// behind it, and returns it if it is the claim claimUID, or nil if that claim
// was deleted or replaced.
func (woc *wfOperationCtx) liveClaim(ctx context.Context, node *wfv1.NodeStatus, claimUID types.UID) (*wfv1.WorkflowTaskResult, error) {
	result, err := woc.controller.dispatcher.Result(ctx, woc.pooledKey(node))
	switch {
	case errors.Is(err, dispatch.ErrNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	case result.UID != claimUID:
		return nil, nil
	}
	return result, nil
}

// endDeadClaim ends the node, whose claim claimUID is held by a claimant that
// can no longer run its task, as Error with why, as a pod's node ends when
// its pod is deleted, with its task result synced, since nothing will
// complete the claim. It reads the claim live first: one its claimant
// completed is applied instead, as the node's first ending, and one deleted
// or replaced was released, which the result cache will show.
func (woc *wfOperationCtx) endDeadClaim(ctx context.Context, node *wfv1.NodeStatus, claimUID types.UID, why string) {
	log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "claimUID": claimUID, "reason": why})
	result, err := woc.liveClaim(ctx, node, claimUID)
	if err != nil {
		log.WithError(err).Warn(ctx, "Failed to read a pooled task's claim")
		woc.requeue()
		return
	}
	var claim dispatch.Claim
	ok := false
	if result != nil {
		claim, ok = dispatch.ClaimFromResult(result)
	}
	switch {
	case !ok:
		log.Info(ctx, "Not ending a pooled task whose claimant is gone: its claim was released")
		woc.requeue()
	case claim.Completed:
		log.Info(ctx, "Applying the completed claim of a pooled task whose claimant is gone")
		woc.applyPooledResult(ctx, node, result, claim)
	default:
		updated := node.DeepCopy()
		if updated.PooledTask.ClaimUID != claim.UID {
			// Its claimant was never cached.
			recordClaim(updated.PooledTask, claim, nil, metav1.Now())
			woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
		}
		log.Info(ctx, "Pooled task's claimant is gone")
		woc.endPooledNode(ctx, updated, wfv1.NodeError, why)
	}
}

// checkTaskSetLost ends the node, whose TaskSet or claim isn't cached, once
// the grace since its enqueue has passed: until then its TaskSet may just not
// be cached yet, but after it the TaskSet was deleted, and with no claim
// recorded or cached no worker runs its task, nor can one claim it. An ended node's TaskSet is deleted by
// its ack, and an ended node never gets here, so an ack is never taken for a
// lost TaskSet.
func (woc *wfOperationCtx) checkTaskSetLost(ctx context.Context, node *wfv1.NodeStatus) {
	if woc.requeueUntil(node.PooledTask.EnqueuedAt.Add(recentlyStartedPodDuration(ctx))) {
		return
	}
	woc.log.WithFields(logging.Fields{"nodeID": node.ID, "taskSetUID": node.PooledTask.TaskSetUID}).Info(ctx, "Pooled task's WorkflowTaskSet is gone")
	woc.endPooledNode(ctx, node, wfv1.NodeError, fmt.Sprintf("WorkflowTaskSet %s deleted", woc.pooledKey(node).TaskSetName()))
}

// endPooledNode ends the unended pooled node, whose task nothing runs, with
// phase and message, with its task result synced, since nothing will report
// it, so the workflow can complete.
func (woc *wfOperationCtx) endPooledNode(ctx context.Context, node *wfv1.NodeStatus, phase wfv1.NodePhase, message string) {
	woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
	woc.markNodePhase(ctx, node.Name, phase, message)
}
