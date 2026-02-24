package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// applyExecutionControl will ensure a pod's execution control annotation is up-to-date
// kills any pending and running pods (except agent pod) when workflow has reached its deadline
func (woc *wfOperationCtx) applyExecutionControl(ctx context.Context, pod *apiv1.Pod, wfNodesLock *sync.RWMutex) {
	if pod == nil || woc.isAgentPod(pod) {
		return
	}

	nodeID := woc.nodeID(pod)
	wfNodesLock.RLock()
	node, err := woc.wf.Status.Nodes.Get(nodeID)
	wfNodesLock.RUnlock()
	if err != nil {
		woc.log.WithField("nodeID", nodeID).Error(ctx, "was unable to obtain node for nodeID")
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
					Info(ctx, "Terminating pod as part of workflow shutdown")
				woc.controller.PodController.TerminateContainers(ctx, pod.Namespace, pod.Name)
				msg := fmt.Sprintf("workflow shutdown with strategy:  %s", woc.GetShutdownStrategy())
				woc.handleExecutionControlError(ctx, nodeID, wfNodesLock, msg)
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
					WithField("workflowDeadline", woc.workflowDeadline).
					Info(ctx, "Terminating pod which has exceeded workflow deadline")
				woc.controller.PodController.TerminateContainers(ctx, pod.Namespace, pod.Name)
				woc.handleExecutionControlError(ctx, nodeID, wfNodesLock, "Step exceeded its deadline")
				return
			}
		}
	}
	if woc.GetShutdownStrategy().Enabled() {
		if _, onExitPod := pod.Labels[common.LabelKeyOnExit]; !woc.GetShutdownStrategy().ShouldExecute(onExitPod) {
			woc.log.WithField("podName", pod.Name).
				Info(ctx, "Terminating on-exit pod")
			woc.controller.PodController.TerminateContainers(ctx, pod.Namespace, pod.Name)
		}
	}
}

// handleExecutionControlError marks a node as failed with an error message
func (woc *wfOperationCtx) handleExecutionControlError(ctx context.Context, nodeID string, wfNodesLock *sync.RWMutex, errorMsg string) {
	wfNodesLock.Lock()
	defer wfNodesLock.Unlock()

	node, err := woc.wf.Status.Nodes.Get(nodeID)
	if err != nil {
		woc.log.WithField("nodeID", nodeID).Error(ctx, "was not able to obtain node for nodeID")
		return
	}
	woc.markNodePhase(ctx, node.Name, wfv1.NodeFailed, errorMsg)

	children, err := woc.wf.Status.Nodes.NestedChildrenStatus(nodeID)
	if err != nil {
		woc.log.WithError(err).Error(ctx, "was not able to obtain children")
		return
	}

	// if node is a pod created from ContainerSet template
	// then need to fail child nodes so they will not hang in Pending after pod deletion
	for _, child := range children {
		if !child.Fulfilled() {
			woc.markNodePhase(ctx, child.Name, wfv1.NodeFailed, errorMsg)
		}
	}
}

// killDaemonedChildren kill any daemoned pods of a steps or DAG template node.
func (woc *wfOperationCtx) killDaemonedChildren(ctx context.Context, nodeID string) {
	if nodeID != "" {
		woc.log.WithField("nodeID", nodeID).Debug(ctx, "Checking daemoned children")
	}
	for _, childNode := range woc.wf.Status.Nodes {
		if childNode.BoundaryID != nodeID {
			continue
		}
		if !childNode.IsDaemoned() {
			continue
		}
		podName := util.GeneratePodName(woc.wf.Name, childNode.Name, util.GetTemplateFromNode(childNode), childNode.ID, util.GetWorkflowPodNameVersion(woc.wf))
		woc.controller.PodController.TerminateContainers(ctx, woc.wf.Namespace, podName)
		childNode.Phase = wfv1.NodeSucceeded
		childNode.Daemoned = nil
		woc.wf.Status.Nodes.Set(ctx, childNode.ID, childNode)
		woc.updated = true
	}
}
