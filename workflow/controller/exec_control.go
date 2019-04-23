package controller

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

// applyExecutionControl will ensure a pod's execution control annotation is up-to-date
// kills any pending pods when workflow has reached it's deadline
func (woc *wfOperationCtx) applyExecutionControl(pod *apiv1.Pod, wfNodesLock *sync.RWMutex) error {
	if pod == nil {
		return nil
	}
	switch pod.Status.Phase {
	case apiv1.PodSucceeded, apiv1.PodFailed:
		// Skip any pod which are already completed
		return nil
	case apiv1.PodPending:
		// Check if we are past the workflow deadline. If we are, and the pod is still pending
		// then we should simply delete it and mark the pod as Failed
		if woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline) {
			woc.log.Infof("Deleting Pending pod %s/%s which has exceeded workflow deadline %s", pod.Namespace, pod.Name, woc.workflowDeadline)
			err := woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err == nil {
				wfNodesLock.Lock()
				defer wfNodesLock.Unlock()
				node := woc.wf.Status.Nodes[pod.Name]
				var message string
				if woc.workflowDeadline.IsZero() {
					message = "terminated"
				} else {
					message = fmt.Sprintf("step exceeded workflow deadline %s", *woc.workflowDeadline)
				}
				woc.markNodePhase(node.Name, wfv1.NodeFailed, message)
				return nil
			}
			// If we fail to delete the pod, fall back to setting the annotation
			woc.log.Warnf("Failed to delete %s/%s: %v", pod.Namespace, pod.Name, err)
		}
	}

	// Now ensure the pod's current annotation matches our desired deadline
	desiredExecCtl := common.ExecutionControl{
		Deadline: woc.workflowDeadline,
	}
	var podExecCtl common.ExecutionControl
	if execCtlStr, ok := pod.Annotations[common.AnnotationKeyExecutionControl]; ok && execCtlStr != "" {
		err := json.Unmarshal([]byte(execCtlStr), &podExecCtl)
		if err != nil {
			woc.log.Warnf("Failed to unmarshal execution control from pod %s", pod.Name)
		}
	}
	if podExecCtl.Deadline == nil && desiredExecCtl.Deadline == nil {
		return nil
	} else if podExecCtl.Deadline != nil && desiredExecCtl.Deadline != nil {
		if podExecCtl.Deadline.Equal(*desiredExecCtl.Deadline) {
			return nil
		}
	}
	if podExecCtl.Deadline != nil && podExecCtl.Deadline.IsZero() {
		// If the pod has already been explicitly signaled to terminate, then do nothing.
		// This can happen when daemon steps are terminated.
		woc.log.Infof("Skipping sync of execution control of pod %s. pod has been signaled to terminate", pod.Name)
		return nil
	}
	woc.log.Infof("Execution control for pod %s out-of-sync desired: %v, actual: %v", pod.Name, desiredExecCtl.Deadline, podExecCtl.Deadline)
	return woc.updateExecutionControl(pod.Name, desiredExecCtl)
}

// killDaemonedChildren kill any daemoned pods of a steps or DAG template node.
func (woc *wfOperationCtx) killDaemonedChildren(nodeID string) error {
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
		err := woc.updateExecutionControl(childNode.ID, execCtl)
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
func (woc *wfOperationCtx) updateExecutionControl(podName string, execCtl common.ExecutionControl) error {
	execCtlBytes, err := json.Marshal(execCtl)
	if err != nil {
		return errors.InternalWrapError(err)
	}

	woc.log.Infof("Updating execution control of %s: %s", podName, execCtlBytes)
	err = common.AddPodAnnotation(
		woc.controller.kubeclientset,
		podName,
		woc.wf.ObjectMeta.Namespace,
		common.AnnotationKeyExecutionControl,
		string(execCtlBytes),
	)
	if err != nil {
		return err
	}

	// Ideally we would simply annotate the pod with the updates and be done with it, allowing
	// the executor to notice the updates naturally via the Downward API annotations volume
	// mounted file. However, updates to the Downward API volumes take a very long time to
	// propagate (minutes). The following code fast-tracks this by signaling the executor
	// using SIGUSR2 that something changed.
	woc.log.Infof("Signalling %s of updates", podName)
	exec, err := common.ExecPodContainer(
		woc.controller.restConfig, woc.wf.ObjectMeta.Namespace, podName,
		common.WaitContainerName, true, true, "sh", "-c", "kill -s USR2 $(pidof argoexec)",
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
