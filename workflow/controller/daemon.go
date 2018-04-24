package controller

import (
	"encoding/json"
	"time"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
)

// killDeamonedChildren kill any daemoned pods of a steps or DAG template node.
func (woc *wfOperationCtx) killDeamonedChildren(nodeID string) error {
	woc.log.Infof("Checking deamoned children of %s", nodeID)
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
			woc.log.Errorf("Failed to update execution control of %s: %+v", childNode, err)
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
		common.WaitContainerName, true, true, "sh", "-c", "kill -s USR2 1",
	)
	if err != nil {
		return err
	}
	go func() {
		// This call is necessary to actually send the exec. Since signalling is best effort,
		// it is launched as a goroutine and the error is discarded
		_, _, err = common.GetExecutorOutput(exec)
		if err != nil {
			log.Warnf("Signal command failed: %v", err)
			return
		}
		log.Infof("Signal of %s (%s) successfully issued", podName, common.WaitContainerName)
	}()

	return nil
}
