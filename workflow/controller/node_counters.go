package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type counter func(wfv1.NodeStatus) bool

func (woc *wfOperationCtx) getActivePodsCounter(boundaryID string) counter {
	return func(node wfv1.NodeStatus) bool {
		return node.Type == wfv1.NodeTypePod &&
			// Only count pods that match the provided boundaryID, or all if no boundaryID was provided
			(boundaryID == "" || node.BoundaryID == boundaryID) &&
			// Only count Running or Pending pods
			(node.Phase == wfv1.NodePending || node.Phase == wfv1.NodeRunning) &&
			// Only count pods that are NOT waiting for a lock
			(node.SynchronizationStatus == nil || node.SynchronizationStatus.Waiting == "") &&
			// Only count nodes with a backing execution: a created pod or an enqueued pooled task.
			woc.nodeHasExecution(node)
	}
}

func getActiveChildrenCounter(boundaryID string) counter {
	return func(node wfv1.NodeStatus) bool {
		return node.BoundaryID == boundaryID &&
			// Only count Pods, Steps, or DAGs
			(node.Type == wfv1.NodeTypePod || node.Type == wfv1.NodeTypeSteps || node.Type == wfv1.NodeTypeDAG) &&
			// Only count Running or Pending nodes
			(node.Phase == wfv1.NodePending || node.Phase == wfv1.NodeRunning)
	}
}

func getUnsuccessfulChildrenCounter(boundaryID string) counter {
	return func(node wfv1.NodeStatus) bool {
		return node.BoundaryID == boundaryID &&
			// Only count Pods, Steps, or DAGs
			(node.Type == wfv1.NodeTypePod || node.Type == wfv1.NodeTypeSteps || node.Type == wfv1.NodeTypeDAG) &&
			// Only count Failed or Errored nodes
			(node.Phase == wfv1.NodeFailed || node.Phase == wfv1.NodeError)
	}
}

func (woc *wfOperationCtx) getActivePods(boundaryID string) int64 {
	return woc.countNodes(woc.getActivePodsCounter(boundaryID))
}

func (woc *wfOperationCtx) getActiveChildren(boundaryID string) int64 {
	return woc.countNodes(getActiveChildrenCounter(boundaryID))
}

func (woc *wfOperationCtx) getUnsuccessfulChildren(boundaryID string) int64 {
	return woc.countNodes(getUnsuccessfulChildrenCounter(boundaryID))
}

func (woc *wfOperationCtx) nodePodExist(node wfv1.NodeStatus) bool {
	_, podExist, _ := woc.podExists(node.ID)
	return podExist
}

// isPooled reports whether node's task has been enqueued for a hot worker
// pool. A pooled node never has a pod of its own.
func isPooled(node *wfv1.NodeStatus) bool {
	return node.PooledTask != nil
}

// nodeHasExecution reports whether node's task has been handed to something
// that runs it: its pod has been created, or it has been enqueued for a pool.
func (woc *wfOperationCtx) nodeHasExecution(node wfv1.NodeStatus) bool {
	return isPooled(&node) || woc.nodePodExist(node)
}

func (woc *wfOperationCtx) countNodes(counter counter) int64 {
	count := 0
	for _, node := range woc.wf.Status.Nodes {
		if counter(node) {
			count++
		}
	}
	return int64(count)
}
