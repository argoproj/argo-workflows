package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

// applyExecutionControl will ensure a pod's execution control annotation is up-to-date
// kills any pending pods when workflow has reached it's deadline
func (woc *wfOperationCtx) applyExecutionControl(ctx context.Context, clusterName wfv1.ClusterName, un *unstructured.Unstructured, wfNodesLock *sync.RWMutex) error {
	pod, err := util.PodFromUnstructured(un)
	if err != nil {
		return fmt.Errorf("failed to convert unstructured to pod: %w", err)
	}
	return woc.applyPodExecutionControl(ctx, clusterName, pod, wfNodesLock)
}

func (woc *wfOperationCtx) applyPodExecutionControl(ctx context.Context, clusterName wfv1.ClusterName, pod *apiv1.Pod, wfNodesLock *sync.RWMutex) error {
	switch pod.Status.Phase {
	case apiv1.PodSucceeded, apiv1.PodFailed:
		// Skip any pod which are already completed
		return nil
	case apiv1.PodPending:
		// Check if we are currently shutting down
		k, err := woc.controller.dynamicInterfaceX(clusterName, pod.Namespace)
		if err != nil {
			return err
		}
		if woc.execWf.Spec.Shutdown != "" {
			// Only delete pods that are not part of an onExit handler if we are "Stopping" or all pods if we are "Terminating"
			_, onExitPod := pod.Labels[common.LabelKeyOnExit]

			if !woc.wf.Spec.Shutdown.ShouldExecute(onExitPod) {
				woc.log.Infof("Deleting Pending pod %s/%s as part of workflow shutdown with strategy: %s", pod.Namespace, pod.Name, woc.wf.Spec.Shutdown)
				err := k.Resource(common.PodGVR).Namespace(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err == nil {
					wfNodesLock.Lock()
					defer wfNodesLock.Unlock()
					node := woc.wf.Status.Nodes[pod.Name]
					woc.markNodePhase(node.Name, wfv1.NodeFailed, fmt.Sprintf("workflow shutdown with strategy:  %s", woc.execWf.Spec.Shutdown))
					return nil
				}
				// If we fail to delete the pod, fall back to setting the annotation
				woc.log.Warnf("Failed to delete %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
		// Check if we are past the workflow deadline. If we are, and the pod is still pending
		// then we should simply delete it and mark the pod as Failed
		if woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline) {
			//pods that are part of an onExit handler aren't subject to the deadline
			_, onExitPod := pod.Labels[common.LabelKeyOnExit]
			if !onExitPod {
				woc.log.Infof("Deleting Pending pod %s/%s which has exceeded workflow deadline %s", pod.Namespace, pod.Name, woc.workflowDeadline)
				err := k.Resource(common.PodGVR).Namespace(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err == nil {
					wfNodesLock.Lock()
					defer wfNodesLock.Unlock()
					node := woc.wf.Status.Nodes[pod.Name]
					woc.markNodePhase(node.Name, wfv1.NodeFailed, "Step exceeded its deadline")
					return nil
				}
				// If we fail to delete the pod, fall back to setting the annotation
				woc.log.Warnf("Failed to delete %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
	}

	var podExecCtl common.ExecutionControl
	if execCtlStr, ok := pod.Annotations[common.AnnotationKeyExecutionControl]; ok && execCtlStr != "" {
		err := json.Unmarshal([]byte(execCtlStr), &podExecCtl)
		if err != nil {
			woc.log.Warnf("Failed to unmarshal execution control from pod %s", pod.Name)
		}
	}
	containerName := common.WaitContainerName
	// A resource template does not have a wait container,
	// instead the only container is the main container (which is running argoexec)
	if len(pod.Spec.Containers) == 1 {
		containerName = common.MainContainerName
	}

	if woc.wf.Spec.Shutdown != "" {
		if _, onExitPod := pod.Labels[common.LabelKeyOnExit]; !woc.wf.Spec.Shutdown.ShouldExecute(onExitPod) {
			podExecCtl.Deadline = &time.Time{}
			woc.log.Infof("Applying shutdown deadline for pod %s", pod.Name)
			return woc.updateExecutionControl(ctx, clusterName, pod.Namespace, pod.Name, podExecCtl, containerName)
		}
	}

	if woc.workflowDeadline != nil {
		if podExecCtl.Deadline == nil || woc.workflowDeadline.Before(*podExecCtl.Deadline) {
			podExecCtl.Deadline = woc.workflowDeadline
			woc.log.Infof("Applying sooner Workflow Deadline for pod %s at: %v", pod.Name, woc.workflowDeadline)
			return woc.updateExecutionControl(ctx, clusterName, pod.Namespace, pod.Name, podExecCtl, containerName)
		}
	}

	return nil
}

// killDaemonedChildren kill any daemoned pods of a steps or DAG template node.
func (woc *wfOperationCtx) killDaemonedChildren(ctx context.Context, nodeID string) error {
	woc.log.Infof("Checking daemoned children of %s", nodeID)
	var firstErr error
	execCtl := common.ExecutionControl{
		Deadline: &time.Time{},
	}
	for _, childNode := range woc.wf.Status.Nodes {
		if childNode.BoundaryID != nodeID {
			continue
		}
		if childNode.Daemoned == nil || !*childNode.Daemoned {
			continue
		}
		tmpl := woc.execWf.GetTemplateByName(childNode.TemplateName)
		clusterName := tmpl.ClusterName
		namespace := wfv1.NamespaceOr(tmpl.Namespace, woc.wf.Namespace)
		err := woc.updateExecutionControl(ctx, clusterName, namespace, childNode.ID, execCtl, common.WaitContainerName)
		if err != nil {
			woc.log.Errorf("Failed to update execution control of node %s: %+v", childNode.ID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// updateExecutionControl updates the execution control parameters
func (woc *wfOperationCtx) updateExecutionControl(ctx context.Context, clusterName wfv1.ClusterName, namespace, podName string, execCtl common.ExecutionControl, containerName string) error {
	execCtlBytes, err := json.Marshal(execCtl)
	if err != nil {
		return errors.InternalWrapError(err)
	}

	woc.log.Infof("Updating execution control of %s: %s", podName, execCtlBytes)
	k, err := woc.controller.dynamicInterfaceX(clusterName, namespace)
	if err != nil {
		return err
	}
	_, err = k.Resource(common.PodGVR).Namespace(namespace).Patch(ctx, podName, types.MergePatchType, []byte(fmt.Sprintf(`{"metadata": {"annotations": {"%s": "%s"}}}`, common.AnnotationKeyExecutionControl, string(execCtlBytes))), metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// Ideally we would simply annotate the pod with the updates and be done with it, allowing
	// the executor to notice the updates naturally via the Downward API annotations volume
	// mounted file. However, updates to the Downward API volumes take a very long time to
	// propagate (minutes). The following code fast-tracks this by signaling the executor
	// using SIGUSR2 that something changed.
	woc.log.Infof("Signalling %s of updates", podName)
	restConfig, err := woc.controller.restConfigX(clusterName, namespace)
	if err != nil {
		return err
	}
	exec, err := common.ExecPodContainer(
		restConfig, woc.wf.ObjectMeta.Namespace, podName,
		containerName, true, true, "sh", "-c", "kill -s USR2 $(pidof argoexec)",
	)
	if err != nil {
		return err
	}
	go func() {
		// This call is necessary to actually send the exec. Since signalling is best effort,
		// it is launched as a goroutine and the error is discarded
		_, _, err = common.GetExecutorOutput(exec)
		if err != nil {
			woc.log.Warnf("Signal command failed: %v", err)
			return
		}
		woc.log.Infof("Signal of %s (%s) successfully issued", podName, common.WaitContainerName)
	}()

	return nil
}
