package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// applyExecutionControl will ensure a pod's execution control annotation is up-to-date
// kills any pending pods when workflow has reached it's deadline
func (woc *wfOperationCtx) applyExecutionControl(ctx context.Context, pod *apiv1.Pod, wfNodesLock *sync.RWMutex) {
	if pod == nil {
		return
	}

	nodeID := woc.nodeID(pod)

	switch pod.Status.Phase {
	case apiv1.PodSucceeded, apiv1.PodFailed:
		// Skip any pod which are already completed
		return
	case apiv1.PodPending:
		// Check if we are currently shutting down
		if woc.GetShutdownStrategy().Enabled() {
			// Only delete pods that are not part of an onExit handler if we are "Stopping" or all pods if we are "Terminating"
			_, onExitPod := pod.Labels[common.LabelKeyOnExit]

			if !woc.GetShutdownStrategy().ShouldExecute(onExitPod) {
				woc.log.Infof("Deleting Pending pod %s/%s as part of workflow shutdown with strategy: %s", pod.Namespace, pod.Name, woc.GetShutdownStrategy())
				err := woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err == nil {
					msg := fmt.Sprintf("workflow shutdown with strategy:  %s", woc.GetShutdownStrategy())
					woc.handleExecutionControlError(nodeID, wfNodesLock, msg)
					return
				}
				// If we fail to delete the pod, fall back to setting the annotation
				woc.log.Warnf("Failed to delete %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
		// Check if we are past the workflow deadline. If we are, and the pod is still pending
		// then we should simply delete it and mark the pod as Failed
		if woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline) {
			// pods that are part of an onExit handler aren't subject to the deadline
			_, onExitPod := pod.Labels[common.LabelKeyOnExit]
			if !onExitPod {
				woc.log.Infof("Deleting Pending pod %s/%s which has exceeded workflow deadline %s", pod.Namespace, pod.Name, woc.workflowDeadline)
				err := woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err == nil {
					woc.handleExecutionControlError(nodeID, wfNodesLock, "Step exceeded its deadline")
					return
				}
				// If we fail to delete the pod, fall back to setting the annotation
				woc.log.Warnf("Failed to delete %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
	}
	if woc.GetShutdownStrategy().Enabled() {
		if _, onExitPod := pod.Labels[common.LabelKeyOnExit]; !woc.GetShutdownStrategy().ShouldExecute(onExitPod) {
			woc.log.Infof("Shutting down pod %s", pod.Name)
			woc.controller.queuePodForCleanup(woc.wf.Namespace, pod.Name, shutdownPod)
		}
	}
}

// handleExecutionControlError marks a node as failed with an error message
func (woc *wfOperationCtx) handleExecutionControlError(nodeID string, wfNodesLock *sync.RWMutex, errorMsg string) {
	wfNodesLock.Lock()
	defer wfNodesLock.Unlock()

	node := woc.wf.Status.Nodes[nodeID]
	woc.markNodePhase(node.Name, wfv1.NodeFailed, errorMsg)

	// if node is a pod created from ContainerSet template
	// then need to fail child nodes so they will not hang in Pending after pod deletion
	for _, nodeID := range node.Children {
		child := woc.wf.Status.Nodes[nodeID]
		woc.markNodePhase(child.Name, wfv1.NodeFailed, errorMsg)
	}
}

// killDaemonedChildren kill any daemoned pods of a steps or DAG template node.
func (woc *wfOperationCtx) killDaemonedChildren(nodeID string) {
	woc.log.Infof("Checking daemoned children of %s", nodeID)
	for _, childNode := range woc.wf.Status.Nodes {
		if childNode.BoundaryID != nodeID {
			continue
		}
		if childNode.Daemoned == nil || !*childNode.Daemoned {
			continue
		}
		woc.controller.queuePodForCleanup(woc.wf.Namespace, childNode.ID, shutdownPod)
		childNode.Phase = wfv1.NodeSucceeded
		childNode.Daemoned = nil
		woc.wf.Status.Nodes[childNode.ID] = childNode
		woc.updated = true
	}
}
