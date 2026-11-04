package controller

import (
	"context"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// A queued pooled task past one of its own deadlines is dropped
// (Dispatcher.Drop), as a pod past its deadline is deleted or killed. A task
// no worker has claimed yet is bounded as a pending pod is: its template's
// timeout and pendingTimeout, from its node's start, end it when its template
// is reconciled (checkPooledTimeouts), as checkConstraints ends a pending
// pod's node, which for a retried template sees only the Retry node; and so
// does its absolute deadline. A task a worker has released has run, so it is
// bounded as a running pod is, by its absolute deadline alone, whether or not
// its template is reconciled (checkReleasedDeadline); its pendingTimeout no
// longer applies. A claimed task is its worker's to stop at its deadline,
// backed by its lease (checkLease).

// dropPooledTask ends the unended pooled node, whose ledger has its task
// queued, with phase and message, as a shutdown ends it
// (Dispatcher.Shutdown): its TaskSet is deleted, and then its claim read
// live, so a claim racing the drop is found and interrupted. A completed one
// is the node's ending, and with none the node finishes at once
// (applyPooledEnding).
func (woc *wfOperationCtx) dropPooledTask(ctx context.Context, node *wfv1.NodeStatus, phase wfv1.NodePhase, message string) {
	task := dispatch.LedgerEntry{Key: woc.pooledKey(node), Ledger: *node.PooledTask.DeepCopy()}
	for _, outcome := range woc.controller.dispatcher.Shutdown(ctx, []dispatch.LedgerEntry{task}) {
		woc.applyPooledEnding(ctx, node, outcome, phase, message)
	}
}

// timedOutPhase is the phase a node past its template's timeout ends in:
// Failed, or Error for a hook, which like a hook that could not run ends its
// task's boundary Error.
func timedOutPhase(node *wfv1.NodeStatus) wfv1.NodePhase {
	if node.NodeFlag != nil && node.NodeFlag.Hooked {
		return wfv1.NodeError
	}
	return wfv1.NodeFailed
}

// checkPooledTimeouts drops the task of the unended pooled node, whose
// processed template is tmpl, if no worker has claimed it yet and it is past
// a deadline: its template's timeout or pendingTimeout, with
// ErrTimeout's message, as checkConstraints ends a pending pod's node, or its
// absolute deadline, with the deadline message its worker would have given.
// Until then the workflow is requeued for the earliest. It returns the node
// as it now is.
func (woc *wfOperationCtx) checkPooledTimeouts(ctx context.Context, node *wfv1.NodeStatus, tmpl *wfv1.Template) *wfv1.NodeStatus {
	ledger := node.PooledTask
	if node.Phase.Completed() || ledger.State() != wfv1.PooledTaskEnqueued || ledger.Released() {
		return node
	}
	now := time.Now().UTC()
	timeout, pendingTimeout, err := templateTimeoutDeadlines(tmpl, node, now)
	log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "template": tmpl.Name})
	switch {
	case err != nil:
		log.WithError(err).Info(ctx, "Queued pooled task exceeded its timeout")
		woc.dropPooledTask(ctx, node, timedOutPhase(node), err.Error())
	case ledger.Deadline != nil && now.After(ledger.Deadline.Time):
		log.WithField("deadline", ledger.Deadline).Info(ctx, "Queued pooled task exceeded its deadline")
		woc.dropPooledTask(ctx, node, wfv1.NodeFailed, common.DeadlineExceededMessage)
	default:
		var deadline *time.Time
		if ledger.Deadline != nil {
			deadline = &ledger.Deadline.Time
		}
		if first := earliest(deadline, timeout, pendingTimeout); first != nil {
			woc.requeueAfter(first.Sub(now))
		}
		return node
	}
	if updated, err := woc.wf.Status.Nodes.Get(node.ID); err == nil {
		return updated
	}
	return node
}

// checkReleasedDeadline drops the task of the unended, released pooled node
// once it is past its absolute deadline, with the deadline message its
// worker would have given, and reports whether it tried. Until then the
// workflow is requeued for it.
func (woc *wfOperationCtx) checkReleasedDeadline(ctx context.Context, node *wfv1.NodeStatus) bool {
	deadline := node.PooledTask.Deadline
	if deadline == nil {
		return false
	}
	if remaining := time.Until(deadline.Time); remaining > 0 {
		woc.requeueAfter(remaining)
		return false
	}
	woc.log.WithFields(logging.Fields{"nodeID": node.ID, "deadline": deadline, "releases": node.PooledTask.Releases}).
		Info(ctx, "Released pooled task exceeded its deadline")
	woc.dropPooledTask(ctx, node, wfv1.NodeFailed, common.DeadlineExceededMessage)
	return true
}
