package controller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// pooledKey is the dispatch key of the pooled node's task.
func (woc *wfOperationCtx) pooledKey(node *wfv1.NodeStatus) dispatch.Key {
	return woc.pooledKeyFor(node.ID)
}

// pooledKeyFor is the dispatch key of the task of the pooled node nodeID.
func (woc *wfOperationCtx) pooledKeyFor(nodeID string) dispatch.Key {
	return dispatch.Key{Namespace: woc.wf.Namespace, NodeID: nodeID}
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
