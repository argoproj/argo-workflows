package controller

import (
	"fmt"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type counter struct {
	// key of the counter
	key string
	// increment count ifNode return true
	ifNode func(wfv1.NodeStatus) bool
}

type counterType string

const (
	counterTypeActivePods           counterType = "activePods"
	counterTypeActiveChildren       counterType = "activeChildren"
	counterTypeUnsuccessfulChildren counterType = "unsuccessfulChildren"
)

func getKey(counterType counterType, boundaryID string) string {
	return fmt.Sprintf("%s/%s", counterType, boundaryID)
}

func getActivePodsCounter(boundaryID string) counter {
	return counter{
		key: getKey(counterTypeActivePods, boundaryID),
		ifNode: func(node wfv1.NodeStatus) bool {
			return node.Type == wfv1.NodeTypePod &&
				// Only count pods that match the provided boundaryID, or all if no boundaryID was provided
				(boundaryID == "" || node.BoundaryID == boundaryID) &&
				// Only count Running or Pending pods
				(node.Phase == wfv1.NodePending || node.Phase == wfv1.NodeRunning) &&
				// Only count pods that are NOT waiting for a lock
				(node.SynchronizationStatus == nil || node.SynchronizationStatus.Waiting == "")
		},
	}
}

func getActiveChildrenCounter(boundaryID string) counter {
	return counter{
		key: getKey(counterTypeActiveChildren, boundaryID),
		ifNode: func(node wfv1.NodeStatus) bool {
			return node.BoundaryID == boundaryID &&
				// Only count Pods, Steps, or DAGs
				(node.Type == wfv1.NodeTypePod || node.Type == wfv1.NodeTypeSteps || node.Type == wfv1.NodeTypeDAG) &&
				// Only count Running or Pending nodes
				(node.Phase == wfv1.NodePending || node.Phase == wfv1.NodeRunning)
		},
	}
}

func getUnsuccessfulChildrenCounter(boundaryID string) counter {
	return counter{
		key: getKey(counterTypeUnsuccessfulChildren, boundaryID),
		ifNode: func(node wfv1.NodeStatus) bool {
			return node.BoundaryID == boundaryID &&
				// Only count Pods, Steps, or DAGs
				(node.Type == wfv1.NodeTypePod || node.Type == wfv1.NodeTypeSteps || node.Type == wfv1.NodeTypeDAG) &&
				// Only count Failed or Errored nodes
				(node.Phase == wfv1.NodeFailed || node.Phase == wfv1.NodeError)
		},
	}
}

type count map[string]int64

func (woc *wfOperationCtx) getActivePods(boundaryID string) int64 {
	key := getKey(counterTypeActivePods, boundaryID)
	if _, ok := woc.countedNodes[key]; !ok {
		woc.countNodes(getActivePodsCounter(boundaryID))
	}
	return woc.countedNodes[key]
}

func (woc *wfOperationCtx) getActiveChildren(boundaryID string) int64 {
	key := getKey(counterTypeActiveChildren, boundaryID)
	if _, ok := woc.countedNodes[key]; !ok {
		woc.countNodes(getActiveChildrenCounter(boundaryID))
	}
	return woc.countedNodes[key]
}

func (woc *wfOperationCtx) getUnsuccessfulChildren(boundaryID string) int64 {
	key := getKey(counterTypeUnsuccessfulChildren, boundaryID)
	if _, ok := woc.countedNodes[key]; !ok {
		woc.countNodes(getUnsuccessfulChildrenCounter(boundaryID))
	}
	return woc.countedNodes[key]
}


func (woc *wfOperationCtx) countNodes(counter counter) {
	if woc.countedNodes == nil {
		woc.countedNodes = make(count)
	}

	woc.countedNodes[counter.key] = 0
	for _, node := range woc.wf.Status.Nodes {
		if counter.ifNode(node) {
			woc.countedNodes[counter.key]++
		}
	}
}
