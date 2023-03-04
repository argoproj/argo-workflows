package controller

import (
	"fmt"
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

// applyExecutionControl will ensure a pod's execution control annotation is up-to-date
// kills any pending and running pods when workflow has reached it's deadline
func (woc *wfOperationCtx) applyExecutionControl(pod *apiv1.Pod, wfNodesLock *sync.RWMutex) {
	if pod == nil {
		return
	}

	nodeID := woc.nodeID(pod)
	wfNodesLock.RLock()
	node, ok := woc.wf.Status.Nodes[nodeID]
	wfNodesLock.RUnlock()
	if !ok {
		return
	}
	// node is already completed
	if node.Fulfilled() {
		return
	}
	switch pod.Status.Phase {
	case apiv1.PodSucceeded, apiv1.PodFailed:
		// Skip any pod which are already completed
		return
	case apiv1.PodPending, apiv1.PodRunning:
		// Check if we are currently shutting down
		if woc.GetShutdownStrategy().Enabled() {
			// Only delete pods that are not part of an onExit handler if we are "Stopping" or all pods if we are "Terminating"
			_, onExitPod := pod.Labels[common.LabelKeyOnExit]

			if !woc.GetShutdownStrategy().ShouldExecute(onExitPod) {
				woc.log.WithField("podName", pod.Name).
					WithField("shutdownStrategy", woc.GetShutdownStrategy()).
					Info("Terminating pod as part of workflow shutdown")
				woc.controller.queuePodForCleanup(pod.Namespace, pod.Name, terminateContainers)
				msg := fmt.Sprintf("workflow shutdown with strategy:  %s", woc.GetShutdownStrategy())
				woc.handleExecutionControlError(nodeID, wfNodesLock, msg)
				return
			}
		}
		// Check if we are past the workflow deadline. If we are, and the pod is still pending
		// then we should simply delete it and mark the pod as Failed
		if woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline) {
			// pods that are part of an onExit handler aren't subject to the deadline
			_, onExitPod := pod.Labels[common.LabelKeyOnExit]
			if !onExitPod {
				woc.log.WithField("podName", pod.Name).
					WithField(" workflowDeadline", woc.workflowDeadline).
					Info("Terminating pod which has exceeded workflow deadline")
				woc.controller.queuePodForCleanup(pod.Namespace, pod.Name, terminateContainers)
				woc.handleExecutionControlError(nodeID, wfNodesLock, "Step exceeded its deadline")
				return
			}
		}
	}
	if woc.GetShutdownStrategy().Enabled() {
		if _, onExitPod := pod.Labels[common.LabelKeyOnExit]; !woc.GetShutdownStrategy().ShouldExecute(onExitPod) {
			woc.log.WithField("podName", pod.Name).
				Info("Terminating on-exit pod")
			woc.controller.queuePodForCleanup(woc.wf.Namespace, pod.Name, terminateContainers)
		}
	}
}

// handleExecutionControlError marks a node as failed with an error message
func (woc *wfOperationCtx) handleExecutionControlError(nodeID string, wfNodesLock *sync.RWMutex, errorMsg string) {
	wfNodesLock.Lock()
	defer wfNodesLock.Unlock()

	node := woc.wf.Status.Nodes[nodeID]
	woc.markNodePhase(node.Name, wfv1.NodeFailed, errorMsg)

	children := []wfv1.NodeStatus{}
	q := []wfv1.NodeStatus{node}
	for {
		if len(q) <= 0 {
			break
		}
		childNode := q[0]
		q = q[1:]
		for _, nodeID := range childNode.Children {
			q = append(q, woc.wf.Status.Nodes[nodeID])
		}
		children = append(children, childNode)
	}

	// if node is a pod created from ContainerSet template
	// then need to fail child nodes so they will not hang in Pending after pod deletion
	for _, child := range children {
		if !child.IsExitNode() && !child.Fulfilled() {
			woc.markNodePhase(child.Name, wfv1.NodeFailed, errorMsg)
		}
	}
}

// killDaemonedChildren kill any daemoned pods of a steps or DAG template node.
func (woc *wfOperationCtx) killDaemonedChildren(nodeID string) {
	woc.log.Infof("Checking daemoned children of %s", nodeID)
	for _, childNode := range woc.wf.Status.Nodes {
		if childNode.BoundaryID != nodeID {
			continue
		}
		if !childNode.IsDaemoned() {
			continue
		}
		podName := util.GeneratePodName(woc.wf.Name, childNode.Name, childNode.TemplateName, childNode.ID, util.GetWorkflowPodNameVersion(woc.wf))
		woc.controller.queuePodForCleanup(woc.wf.Namespace, podName, terminateContainers)
		childNode.Phase = wfv1.NodeSucceeded
		childNode.Daemoned = nil
		woc.wf.Status.Nodes[childNode.ID] = childNode
		woc.updated = true
	}
}
