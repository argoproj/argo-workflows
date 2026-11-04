package controller

import (
	"context"
	"fmt"
	"reflect"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// reconcilePooledResult ingests a result that is a pooled task's claim or
// belongs to a pooled node, and says whether it did; any other result is a
// pod's. A result is applied to its node only if it carries the TaskSet UID
// in the node's ledger, so it is fenced before anything, including its
// completed label, is read from it:
//
//   - a claim on a node with no ledger is left alone: the node may not exist
//     yet, or the status write that recorded its ledger was lost and the next
//     dispatch adopts the TaskSet the claim is on;
//   - a result of a pooled node with another TaskSet UID, or none, is
//     discarded, since a worker can claim the node's TaskSet only once no
//     result holds the name.
//
// Neither changes the node, nor its TaskResultSynced.
func (woc *wfOperationCtx) reconcilePooledResult(ctx context.Context, result *wfv1.WorkflowTaskResult) bool {
	claim, isClaim := dispatch.ClaimFromResult(result)
	node, err := woc.wf.Status.Nodes.Get(result.Name)
	pooled := err == nil && isPooled(node)
	switch {
	case !isClaim && !pooled:
		return false
	case !pooled:
		woc.log.WithFields(logging.Fields{"result": result.Name, "taskSetUID": claim.TaskSetUID}).
			Debug(ctx, "Leaving a pooled task's result for a node with no ledger")
	case !isClaim || claim.TaskSetUID != node.PooledTask.TaskSetUID:
		woc.discardPooledResult(ctx, node, result, claim.TaskSetUID)
	default:
		woc.applyPooledResult(ctx, node, result, claim)
	}
	return true
}

// discardPooledResult deletes a result of the pooled node that doesn't carry
// the TaskSet UID in its ledger. A failed delete is retried by a requeue,
// since nothing else would bring the workflow back to it.
func (woc *wfOperationCtx) discardPooledResult(ctx context.Context, node *wfv1.NodeStatus, result *wfv1.WorkflowTaskResult, taskSetUID types.UID) {
	log := woc.log.WithFields(logging.Fields{
		"nodeID":           node.ID,
		"resultUID":        result.UID,
		"resultTaskSetUID": taskSetUID,
		"ledgerTaskSetUID": node.PooledTask.TaskSetUID,
	})
	if err := woc.controller.dispatcher.Discard(ctx, woc.pooledKey(node), result.UID); err != nil {
		log.WithError(err).Warn(ctx, "Failed to discard a stale pooled task result")
		woc.requeue()
		return
	}
	log.Info(ctx, "Discarded a pooled task result that isn't for the node's WorkflowTaskSet")
}

// applyPooledResult applies a pooled node's claim, which carries the ledger's
// TaskSet UID, to the node. A claim the ledger doesn't hold yet is recorded
// first (claimPooledTask). While the claim is incomplete the node's task
// result is in progress, and only its progress is applied. Once complete, the
// worker has written phase, message and outputs, exit code included, in one
// write, so they are applied together, as a pod's status and task result
// would be, and a succeeded node's outputs are memoized. The node's spans
// follow its phase, as a pod's node's do. A node that has
// already ended, by its own result or by the controller, takes nothing from
// it but the arrival of a completed result, which finishes it.
//
// An incomplete claim is left alone until its claimant pod is cached, so the
// ledger never records a claim without the worker container restart count it
// was first seen with. A completed claim is applied at once: its claimant's
// restarts no longer matter.
func (woc *wfOperationCtx) applyPooledResult(ctx context.Context, node *wfv1.NodeStatus, result *wfv1.WorkflowTaskResult, claim dispatch.Claim) {
	if node.Phase.Completed() {
		if claim.Completed && woc.wf.Status.IsTaskResultIncomplete(node.ID) {
			woc.finishPooledNode(ctx, node.ID, "its claim was completed")
		}
		return
	}
	newClaim := claim.UID != node.PooledTask.ClaimUID
	var pod *apiv1.Pod
	if newClaim {
		pod = woc.claimantPod(claim.WorkerPodName, claim.WorkerPodUID)
		if pod == nil && !claim.Completed {
			woc.log.WithFields(logging.Fields{"nodeID": node.ID, "workerPod": claim.WorkerPodName}).
				Debug(ctx, "Leaving a pooled task's claim until its claimant pod is cached")
			return
		}
	}
	if claim.Completed {
		woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
	} else {
		woc.wf.Status.MarkTaskResultIncomplete(ctx, node.ID)
	}
	marked, err := woc.wf.Status.Nodes.Get(node.ID)
	if err != nil {
		return
	}
	updated := marked.DeepCopy()
	if newClaim {
		woc.claimPooledTask(ctx, updated, claim, pod)
	}
	if result.Progress.IsValid() {
		updated.Progress = result.Progress
	}
	if claim.Completed {
		woc.completePooledNode(ctx, updated, result)
		woc.log.WithFields(logging.Fields{"nodeID": node.ID, "phase": updated.Phase, "workerPod": claim.WorkerPodName}).Info(ctx, "Pooled task completed")
	}
	if !reflect.DeepEqual(node, updated) {
		woc.log.WithFields(logging.Fields{"nodeID": node.ID, "phase": updated.Phase}).Debug(ctx, "pooled task result changed")
		woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
		woc.controller.tracing.ChangeNodePhase(ctx, woc.wf.Namespace+"/"+woc.wf.Name, updated.ID, updated.Phase, updated.Message)
		woc.updated = true
	}
}

// completePooledNode records the completed result of a pooled node's task on
// it. A result completed without a final phase can't be retried by its
// worker, so the node errors rather than waiting for ever.
func (woc *wfOperationCtx) completePooledNode(ctx context.Context, node *wfv1.NodeStatus, result *wfv1.WorkflowTaskResult) {
	if result.Phase.Completed() {
		node.Phase, node.Message = result.Phase, result.Message
	} else {
		node.Phase, node.Message = wfv1.NodeError, fmt.Sprintf("pooled task completed with phase %q", result.Phase)
	}
	if result.Outputs != nil {
		node.Outputs = result.Outputs.DeepCopy()
	}
	if node.FinishedAt.IsZero() {
		node.FinishedAt = metav1.Now()
	}
	if err := woc.saveMemoizedOutputs(ctx, node); err != nil {
		node.Phase = wfv1.NodeError
		node.Message = err.Error()
	}
}
