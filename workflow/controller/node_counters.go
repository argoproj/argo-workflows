package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type counter struct {
	key    counterType
	ifNode func(wfv1.NodeStatus) bool
}

type counterType string

const (
	counterTypeActivePods              = "activePods"
	counterTypeActiveChildren          = "activeChildren"
	counterTypeFailedOrErroredChildren = "failedOrErroredChildren"
)

func getActivePodsCounter(boundaryID string) counter {
	return counter{
		key: counterTypeActivePods,
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
		key: counterTypeActiveChildren,
		ifNode: func(node wfv1.NodeStatus) bool {
			return node.BoundaryID == boundaryID &&
				// Only count Pods, Steps, or DAGs
				(node.Type == wfv1.NodeTypePod || node.Type == wfv1.NodeTypeSteps || node.Type == wfv1.NodeTypeDAG) &&
				// Only count Running or Pending nodes
				(node.Phase == wfv1.NodePending || node.Phase == wfv1.NodeRunning)
		},
	}
}

func getFailedOrErroredChildrenCounter(boundaryID string) counter {
	return counter{
		key: counterTypeFailedOrErroredChildren,
		ifNode: func(node wfv1.NodeStatus) bool {
			return node.BoundaryID == boundaryID &&
				// Only count Pods, Steps, or DAGs
				(node.Type == wfv1.NodeTypePod || node.Type == wfv1.NodeTypeSteps || node.Type == wfv1.NodeTypeDAG) &&
				// Only count Failed or Errored nodes
				(node.Phase == wfv1.NodeFailed || node.Phase == wfv1.NodeError)
		},
	}
}

type count map[counterType]int

func (c count) addKeyIfNotPresent(key counterType) {
	if _, ok := c[key]; !ok {
		c[key] = 0
	}
}

func (c count) count(key counterType) {
	c[key]++
}

func (c count) getCount() int {
	if len(c) != 1 {
		panic("getCount applied to a count with multiple types")
	}
	for _, val := range c {
		return val
	}
	panic("unreachable: we know count has exactly one element and it wasn't returned")
}

func (c count) getCountType(key counterType) int {
	return c[key]
}

func (woc *wfOperationCtx) countNodes(counters ...counter) count {
	count := make(count)
	for _, node := range woc.wf.Status.Nodes {
		for _, counter := range counters {
			count.addKeyIfNotPresent(counter.key)
			if counter.ifNode(node) {
				count.count(counter.key)
			}
		}
	}
	return count
}
