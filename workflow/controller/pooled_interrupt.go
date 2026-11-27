package controller

import (
	"context"
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// When the controller ends a claimed pooled node, its task is interrupted, as
// a pod's containers are terminated: the TaskSet is deleted and the
// claim labelled for its worker to stop the task (Dispatcher.Interrupt), the
// ledger records the interrupt, and the node is marked failed. Like a pod's
// node until its wait container reports, the node isn't finished, so its
// locks stay held and its dependants and exit handlers wait, until its task
// is known to have stopped: its worker completes the claim, whose phase
// and outputs are then ignored, or deletes it; its claimant can no longer run
// it; or the lease slack has passed since the interrupt, for a worker that
// never confirms. An interrupt that finds the claim already deleted or
// replaced finishes the node at once. A finished node is never made
// unfinished again.

// pooledKey is the dispatch key of the pooled node's task.
func (woc *wfOperationCtx) pooledKey(node *wfv1.NodeStatus) dispatch.Key {
	return woc.pooledKeyFor(node.ID)
}

// pooledKeyFor is the dispatch key of the task of the pooled node nodeID.
func (woc *wfOperationCtx) pooledKeyFor(nodeID string) dispatch.Key {
	return dispatch.Key{Namespace: woc.wf.Namespace, NodeID: nodeID}
}

// leaseSlack is how long the controller waits for a claimed task's worker
// past the task's deadline, or after interrupting it, before treating the
// task as stopped.
func (woc *wfOperationCtx) leaseSlack() time.Duration {
	return time.Duration(woc.controller.Config.HotWorkers.GetLeaseSlackSeconds()) * time.Second
}

// interruptClaimedNode ends the unended, claimed pooled node with phase and
// message, and interrupts its task. If the interrupt fails, the node is left
// as it is and the workflow requeued, to try again.
func (woc *wfOperationCtx) interruptClaimedNode(ctx context.Context, node *wfv1.NodeStatus, phase wfv1.NodePhase, message string) {
	ledger := node.PooledTask
	err := woc.controller.dispatcher.Interrupt(ctx, woc.pooledKey(node), ledger.TaskSetUID, ledger.ClaimUID)
	lost := errors.Is(err, dispatch.ErrClaimLost)
	if err != nil && !lost {
		woc.log.WithFields(logging.Fields{"nodeID": node.ID, "claimUID": ledger.ClaimUID}).WithError(err).
			Warn(ctx, "Failed to interrupt a pooled task")
		woc.requeue()
		return
	}
	woc.endInterruptedNode(ctx, node, lost, phase, message)
}

// endInterruptedNode ends the unended pooled node, whose task its ledger has
// claimed and whose TaskSet an interrupt has just deleted, with phase and
// message, and records the interrupt on its ledger. Its task result stays in
// progress until the task has stopped, unless lost says the claim was already
// gone, in which case nothing runs the task and the node finishes now.
func (woc *wfOperationCtx) endInterruptedNode(ctx context.Context, node *wfv1.NodeStatus, lost bool, phase wfv1.NodePhase, message string) {
	woc.controller.pooledTaskSetsGone.record(woc.wf.UID, pooledTaskSetRef(node))
	updated := node.DeepCopy()
	updated.PooledTask.Interrupt(metav1.Now())
	woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
	if lost {
		woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
	} else {
		woc.wf.Status.MarkTaskResultIncomplete(ctx, node.ID)
	}
	woc.markNodePhase(ctx, node.Name, phase, message)
	woc.updated = true
	woc.log.WithFields(logging.Fields{
		"nodeID":    node.ID,
		"claimUID":  updated.PooledTask.ClaimUID,
		"workerPod": updated.PooledTask.WorkerPodName,
		"claimLost": lost,
		"message":   message,
	}).Info(ctx, "Interrupted pooled task")
}

// endPooledTasksOnShutdownOrDeadline ends the workflow's unended pooled nodes
// that its shutdown strategy or its deadline stops, as pod reconciliation
// terminates their pods (applyExecutionControl), with pods' messages. Stop
// and the deadline spare exit-handler nodes, so a running exit handler goes
// on; terminate spares nothing. Each node's task is shut down
// (Dispatcher.Shutdown): a queued one's TaskSet is dropped, which fails it at
// once, unless the drop finds a claim, which is interrupted, as is a claimed
// one, whose node then waits for its task to stop. A task that can't be shut
// down is left as it is, and tried again at the next reconcile.
func (woc *wfOperationCtx) endPooledTasksOnShutdownOrDeadline(ctx context.Context) {
	shutdown := woc.GetShutdownStrategy()
	deadlineExceeded := woc.workflowDeadlineExceeded()
	if !shutdown.Enabled() && !deadlineExceeded {
		return
	}
	var tasks []dispatch.LedgerEntry
	messages := map[string]string{}
	for id, node := range woc.wf.Status.Nodes {
		if !isPooled(&node) || node.Phase.Completed() {
			continue
		}
		onExit := woc.pooledOnExit(ctx, &node)
		switch {
		case shutdown.Enabled() && !shutdown.ShouldExecute(onExit):
			messages[id] = shutdownMessage(shutdown)
		case deadlineExceeded && !onExit:
			messages[id] = common.DeadlineExceededMessage
		default:
			continue
		}
		tasks = append(tasks, dispatch.LedgerEntry{Key: woc.pooledKey(&node), Ledger: *node.PooledTask.DeepCopy()})
	}
	if len(tasks) == 0 {
		return
	}
	for _, outcome := range woc.controller.dispatcher.Shutdown(ctx, tasks) {
		node, err := woc.wf.Status.Nodes.Get(outcome.NodeID)
		if err != nil {
			continue
		}
		woc.applyPooledEnding(ctx, node, outcome, wfv1.NodeFailed, messages[outcome.NodeID])
	}
}

// pooledOnExit reports whether the pooled node runs an exit handler, which
// stop and the workflow deadline spare. As a pod's onExit label says it for
// the pod, the node's TaskSet carries the label dispatch set from the same
// flag. The node tree, which misses the attempts of a retried workflow exit
// handler (their boundary is the workflow's), is the fallback for a TaskSet
// not cached.
func (woc *wfOperationCtx) pooledOnExit(ctx context.Context, node *wfv1.NodeStatus) bool {
	if taskSet, ok := woc.cachedPooledTaskSet(pooledTaskSetRef(node)); ok {
		return taskSet.GetLabels()[common.LabelKeyOnExit] == "true"
	}
	return node.IsPartOfExitHandler(ctx, woc.wf.Status.Nodes)
}

// applyPooledEnding ends the unended pooled node with phase and message, as
// outcome says ending its task went: the drop of its queued task or the
// interrupt of its claimed one, by a shutdown (Dispatcher.Shutdown) or on
// its own (dropPooledTask). A task dropped before any worker claimed it
// finishes the node at once, since nothing runs it. A claim the drop found
// that has completed is the node's ending, as it ended first; one still
// running is recorded and was interrupted, as was a claimed task, so the node
// waits for the task to stop, unless the claim was already gone. A task that
// couldn't be ended leaves the node as it is, to be tried again at the next
// reconcile.
func (woc *wfOperationCtx) applyPooledEnding(ctx context.Context, node *wfv1.NodeStatus, outcome dispatch.ShutdownOutcome, phase wfv1.NodePhase, message string) {
	log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "message": message})
	switch {
	case outcome.Err != nil:
		log.WithError(outcome.Err).Warn(ctx, "Failed to end a pooled task")
		woc.requeue()
	case outcome.Found != nil && outcome.Found.Completed:
		woc.applyFoundCompletedClaim(ctx, node, *outcome.Found)
	case outcome.Found != nil:
		node = woc.recordFoundClaim(ctx, node, outcome.Found.Claim)
		woc.endInterruptedNode(ctx, node, outcome.Finished, phase, message)
	case node.PooledTask.State() == wfv1.PooledTaskEnqueued:
		// Dropped before any worker claimed it, so nothing runs it.
		woc.controller.pooledTaskSetsGone.record(woc.wf.UID, pooledTaskSetRef(node))
		woc.endPooledNode(ctx, node, phase, message)
		log.Info(ctx, "Dropped pooled task")
	default:
		woc.endInterruptedNode(ctx, node, outcome.Finished, phase, message)
	}
}

// recordFoundClaim records on the unended node, and returns it with, claim, a
// claim the drop of its queued task found, which the ledger doesn't hold yet:
// as ingest records a claim, the node is Running on its claimant, which the
// pool pod cache may not hold yet (recordClaim).
func (woc *wfOperationCtx) recordFoundClaim(ctx context.Context, node *wfv1.NodeStatus, claim dispatch.Claim) *wfv1.NodeStatus {
	updated := node.DeepCopy()
	woc.claimPooledTask(ctx, updated, claim, woc.claimantPod(claim.WorkerPodName, claim.WorkerPodUID))
	woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
	return updated
}

// applyFoundCompletedClaim applies found, a completed claim the drop of the
// unended node's queued task found, as the node's ending, since it ended
// before the drop: the drop's live read of its result is ingested, as the
// cache may not have it.
func (woc *wfOperationCtx) applyFoundCompletedClaim(ctx context.Context, node *wfv1.NodeStatus, found dispatch.FoundClaim) {
	woc.controller.pooledTaskSetsGone.record(woc.wf.UID, pooledTaskSetRef(node))
	woc.applyPooledResult(ctx, node, found.Result, found.Claim)
}

// checkEndedPooledTasks checks every pooled node that has ended. An
// incomplete claim on its TaskSet that the controller hasn't interrupted, and
// that isn't a dead claimant's, is interrupted: a claim racing the
// node's ending, such as one of liveness's endings, which delete nothing, or
// the claim of a node something else ended without interrupting it. An
// unfinished node is finished once its task has stopped.
func (woc *wfOperationCtx) checkEndedPooledTasks(ctx context.Context) {
	woc.eachPooledNode(func(node *wfv1.NodeStatus) bool { return node.Phase.Completed() }, func(node *wfv1.NodeStatus) {
		claim, ok := taskSetClaim(woc.controller.taskResultInformer.GetStore(), woc.wf.Namespace, pooledTaskSetRef(node))
		if ok && !claim.Completed && !claimInterrupted(node.PooledTask, claim) && !deadClaim(node, claim) {
			woc.interruptEndedClaim(ctx, node, claim)
			return
		}
		if !node.Fulfilled() {
			woc.finishStoppedPooledTask(ctx, node)
		}
	})
}

// claimInterrupted says whether the controller has interrupted claim: its
// result carries the interrupt label, or the ledger records an interrupt of
// it, whose label the result cache may not show yet.
func claimInterrupted(ledger *wfv1.PooledTaskStatus, claim dispatch.Claim) bool {
	return claim.Interrupted || ledger.State() == wfv1.PooledTaskInterrupted && ledger.ClaimUID == claim.UID
}

// deadClaim says whether claim is the one the finished node's ledger holds,
// which nothing runs: a node finishes with its claim held only when its
// claimant is gone (endDeadClaim), and its worker, if it lives on, fails the
// claim on restart (FailOwn).
func deadClaim(node *wfv1.NodeStatus, claim dispatch.Claim) bool {
	return node.Fulfilled() && node.PooledTask.State() == wfv1.PooledTaskClaimed && node.PooledTask.ClaimUID == claim.UID
}

// interruptEndedClaim interrupts claim, an incomplete claim on the ended
// node's TaskSet that the controller hasn't interrupted. The ledger records
// the claim, if it doesn't hold it yet (recordClaim), and the interrupt,
// unless it has recorded an interrupt already. The node's task result is left
// as it is: a finished node stays finished, and an unfinished one waits
// for its task to stop. A claim already gone needs nothing.
func (woc *wfOperationCtx) interruptEndedClaim(ctx context.Context, node *wfv1.NodeStatus, claim dispatch.Claim) {
	log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "claimUID": claim.UID, "workerPod": claim.WorkerPodName})
	err := woc.controller.dispatcher.Interrupt(ctx, woc.pooledKey(node), node.PooledTask.TaskSetUID, claim.UID)
	if errors.Is(err, dispatch.ErrClaimLost) {
		log.Debug(ctx, "Not interrupting a claim on an ended pooled task that is already gone")
		return
	}
	if err != nil {
		log.WithError(err).Warn(ctx, "Failed to interrupt a claim on an ended pooled task")
		woc.requeue()
		return
	}
	woc.controller.pooledTaskSetsGone.record(woc.wf.UID, pooledTaskSetRef(node))
	updated := node.DeepCopy()
	ledger := updated.PooledTask
	now := metav1.Now()
	if ledger.ClaimUID != claim.UID {
		recordClaim(ledger, claim, woc.claimantPod(claim.WorkerPodName, claim.WorkerPodUID), now)
	}
	ledger.Interrupt(now)
	woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
	woc.updated = true
	log.WithField("finished", node.Fulfilled()).Info(ctx, "Interrupted a claim on an ended pooled task")
}

// finishStoppedPooledTask finishes the ended, unfinished pooled node once the
// task its ledger claims has stopped: its claim was deleted or replaced, as
// its worker does once it has stopped the task; its claimant can no longer
// run it; or, once interrupted, the lease slack has passed since the
// interrupt, for a worker that never confirms. A claim its worker
// completed finishes the node too, which ingest applies.
func (woc *wfOperationCtx) finishStoppedPooledTask(ctx context.Context, node *wfv1.NodeStatus) {
	ledger := node.PooledTask
	switch ledger.State() {
	case wfv1.PooledTaskClaimed, wfv1.PooledTaskInterrupted:
	default:
		return
	}
	if why := woc.pooledTaskStopped(ctx, node); why != "" {
		woc.finishPooledNode(ctx, node.ID, why)
		return
	}
	interruptedAt := ledger.InterruptedAt()
	if interruptedAt == nil {
		return
	}
	if woc.requeueUntil(interruptedAt.Add(woc.leaseSlack())) {
		return
	}
	woc.finishPooledNode(ctx, node.ID, "the lease slack passed since its task was interrupted")
}

// pooledTaskStopped says why the task the ended node's ledger claims has
// stopped, or is empty if it may still be running. A claim missing from the
// result cache, or a claimant missing from the pool pod cache, is read live:
// a claim recorded from a live read, as an interrupt records one, may not be
// cached yet, nor may its claimant.
func (woc *wfOperationCtx) pooledTaskStopped(ctx context.Context, node *wfv1.NodeStatus) string {
	ledger := node.PooledTask
	log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "claimUID": ledger.ClaimUID, "workerPod": ledger.WorkerPodName})
	if !woc.pooledClaimCached(node) {
		result, err := woc.liveClaim(ctx, node, ledger.ClaimUID)
		switch {
		case err != nil:
			log.WithError(err).Warn(ctx, "Failed to read an interrupted pooled task's claim")
			woc.requeue()
			return ""
		case result == nil:
			return "its claim was deleted"
		}
		if claim, ok := dispatch.ClaimFromResult(result); ok && claim.Completed {
			return "its claim was completed"
		}
		// Still to be cached, which requeues the workflow.
		return ""
	}
	if pod := woc.claimantPod(ledger.WorkerPodName, ledger.WorkerPodUID); pod != nil {
		return claimantGone(pod, ledger.WorkerPodName, ledger.WorkerRestartCount)
	}
	why, err := woc.claimantDeleted(ctx, ledger.WorkerPodName, ledger.WorkerPodUID)
	if err != nil {
		log.WithError(err).Warn(ctx, "Failed to get the claimant pod of an interrupted pooled task")
		woc.requeue()
	}
	return why
}

// finishPooledNode finishes the ended pooled node, whose task has stopped for
// the reason why: its task result is synced, so its locks are released and
// its dependants and exit handlers start.
func (woc *wfOperationCtx) finishPooledNode(ctx context.Context, id, why string) {
	woc.wf.Status.MarkTaskResultComplete(ctx, id)
	node, err := woc.wf.Status.Nodes.Get(id)
	if err != nil {
		return
	}
	if node.FinishedAt.IsZero() {
		node.FinishedAt = metav1.Now()
		woc.wf.Status.Nodes.Set(ctx, id, *node)
	}
	woc.updated = true
	woc.log.WithFields(logging.Fields{"nodeID": id, "reason": why}).Info(ctx, "Pooled task stopped, so its node is finished")
}
