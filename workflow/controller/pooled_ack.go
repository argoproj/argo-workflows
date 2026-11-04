package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// A pooled node's TaskSet is deleted by the controller once the node is
// finished (ackPooledTasks), and any the workflow still has are deleted before
// it completes, so nothing pooled outlives it. Neither touches results or
// ledgers, so neither can look like a release, which is a claim going while
// its node hasn't ended.

// goneTaskSets remembers, per workflow, the pooled TaskSets the controller
// has found gone: deleted by an ack or a completion, or already deleted or
// replaced. They aren't deleted again; a UID is never reused, so a
// TaskSet remembered gone can't come back. It is memory only, so after a
// restart a workflow's first completion deletes what its ledgers record once
// more. A workflow's record is dropped when it completes or is deleted.
type goneTaskSets struct {
	mu   sync.Mutex
	uids map[types.UID]map[types.UID]bool
}

func (g *goneTaskSets) has(workflow, taskSet types.UID) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.uids[workflow][taskSet]
}

func (g *goneTaskSets) record(workflow types.UID, refs ...dispatch.TaskSetRef) {
	if len(refs) == 0 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.uids == nil {
		g.uids = map[types.UID]map[types.UID]bool{}
	}
	if g.uids[workflow] == nil {
		g.uids[workflow] = map[types.UID]bool{}
	}
	for _, ref := range refs {
		g.uids[workflow][ref.UID] = true
	}
}

func (g *goneTaskSets) forget(workflow types.UID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.uids, workflow)
}

// pooledTaskSetRef is the ref of the TaskSet a pooled node's ledger records.
func pooledTaskSetRef(node *wfv1.NodeStatus) dispatch.TaskSetRef {
	return dispatch.TaskSetRef{NodeID: node.ID, UID: node.PooledTask.TaskSetUID}
}

// ackPooledTasks deletes the TaskSet of every pooled node that the workflow,
// as just persisted, has finished, with the ledger's UID as
// precondition. Like a pod's cleanup it runs only once the workflow write
// recording the node finished has succeeded, so a lost write can't leave a
// node unfinished with its TaskSet gone. A node finished by this operation is
// acked even if its TaskSet isn't cached, since it may have been created
// moments ago; one finished earlier is acked again only while the cache
// still has its TaskSet, as after a failed ack, so a restart doesn't delete
// every finished node's TaskSet again. A failed ack requeues.
func (woc *wfOperationCtx) ackPooledTasks(ctx context.Context) {
	// A completed workflow's TaskSets are gone, since its completion waited
	// for that, and its record may already have been dropped.
	if woc.wf.Status.Fulfilled() {
		return
	}
	gone := &woc.controller.pooledTaskSetsGone
	for _, node := range woc.wf.Status.Nodes {
		if !isPooled(&node) || !node.Fulfilled() || node.PooledTask.TaskSetUID == "" {
			continue
		}
		ref := pooledTaskSetRef(&node)
		if gone.has(woc.wf.UID, ref.UID) || !woc.finishedNow(&node) && !woc.pooledTaskSetCached(ref) {
			continue
		}
		log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "taskSetUID": ref.UID})
		err := woc.controller.dispatcher.Ack(ctx, woc.pooledKey(&node), ref.UID)
		if err != nil && !errors.Is(err, dispatch.ErrStale) {
			log.WithError(err).Warn(ctx, "Failed to delete a finished pooled task's WorkflowTaskSet")
			woc.requeue()
			continue
		}
		gone.record(woc.wf.UID, ref)
		log.Debug(ctx, "Deleted a finished pooled task's WorkflowTaskSet")
	}
}

// finishedNow reports whether this operation finished the pooled node, or
// gave it the ledger it has: it wasn't finished, with that ledger, before.
func (woc *wfOperationCtx) finishedNow(node *wfv1.NodeStatus) bool {
	prev, ok := woc.preExecutionNodeStatuses[node.ID]
	return !ok || !prev.Fulfilled() || prev.PooledTask == nil || prev.PooledTask.TaskSetUID != node.PooledTask.TaskSetUID
}

// pooledTaskSetCached reports whether the pooled TaskSet informer holds the
// TaskSet ref names.
func (woc *wfOperationCtx) pooledTaskSetCached(ref dispatch.TaskSetRef) bool {
	key := dispatch.Key{Namespace: woc.wf.Namespace, NodeID: ref.NodeID}
	_, ok := cachedWithUID[metav1.Object](woc.controller.pooledTaskSetInformer.GetStore(), key.Namespace+"/"+key.TaskSetName(), ref.UID)
	return ok
}

// pooledTaskSetsRemain deletes the workflow's pooled TaskSets before it
// completes, since a TaskSet left behind once its results are deleted could be
// claimed and run again. They are found from the cache, owned by this
// workflow, and from the ledgers, since a TaskSet created this operation may
// not be cached yet; those already found gone are left out. It
// reports whether any may remain, in which case completion is held and the
// workflow requeued. A TaskSet whose node hasn't ended is deleted too: once
// the workflow completes, nothing would take its result.
func (woc *wfOperationCtx) pooledTaskSetsRemain(ctx context.Context) bool {
	gone := &woc.controller.pooledTaskSetsGone
	var cached, ledgers []dispatch.TaskSetRef
	objs, _ := woc.controller.pooledTaskSetInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, indexes.WorkflowIndexValue(woc.wf.Namespace, woc.wf.Name))
	for _, obj := range objs {
		m, err := meta.Accessor(obj)
		if err != nil || !dispatch.OwnedByWorkflow(m, woc.wf.UID) {
			continue
		}
		if ref, ok := dispatch.TaskSetRefFromMeta(m); ok && !gone.has(woc.wf.UID, ref.UID) {
			cached = append(cached, ref)
		}
	}
	for _, node := range woc.wf.Status.Nodes {
		if isPooled(&node) && !gone.has(woc.wf.UID, node.PooledTask.TaskSetUID) {
			ledgers = append(ledgers, pooledTaskSetRef(&node))
		}
	}
	if len(cached) == 0 && len(ledgers) == 0 {
		return false
	}
	found, err := woc.controller.dispatcher.Completion(ctx, woc.wf.Namespace, cached, ledgers)
	gone.record(woc.wf.UID, found...)
	if err != nil {
		woc.log.WithError(err).Warn(ctx, "Failed to delete the workflow's pooled WorkflowTaskSets, so can't mark workflow completed")
		woc.holdCompletionForPooledTaskSets(err)
		woc.requeue()
		return true
	}
	return false
}

// holdCompletionForPooledTaskSets says on the workflow why it can't complete,
// and emits a Warning event when that changes. Completion has no time bound:
// a leftover TaskSet could be claimed and its task run again once the
// workflow's results are deleted, so it waits for whoever fixes the deletes
// (a permission, a webhook) or deletes the TaskSets. Marking the workflow
// completed replaces the message.
func (woc *wfOperationCtx) holdCompletionForPooledTaskSets(err error) {
	message := fmt.Sprintf("waiting to delete pooled WorkflowTaskSets: %v", err)
	if woc.wf.Status.Message == message {
		return
	}
	woc.wf.Status.Message = message
	woc.updated = true
	woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowPooledTaskSetsRemain", message)
}
