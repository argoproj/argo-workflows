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
			// Only count pods that are created.
			woc.nodePodExist(node)
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

func (woc *wfOperationCtx) countNodes(counter counter) int64 {
	count := 0
	for _, node := range woc.wf.Status.Nodes {
		if counter(node) {
			count++
		}
	}
	return int64(count)
}
