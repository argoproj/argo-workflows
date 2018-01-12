package controller

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// wfOperationCtx is the context for evaluation and operation of a single workflow
type wfOperationCtx struct {
	// wf is the workflow object
	wf *wfv1.Workflow
	// orig is the original workflow object for purposes of creating a patch
	orig *wfv1.Workflow
	// updated indicates whether or not the workflow object itself was updated
	// and needs to be persisted back to kubernetes
	updated bool
	// log is an logrus logging context to corrolate logs with a workflow
	log *log.Entry
	// controller reference to workflow controller
	controller *WorkflowController
	// globalParms holds any parameters that are available to be referenced
	// in the global scope (e.g. workflow.parameters.XXX).
	globalParams map[string]string
	// map of pods which need to be labeled with completed=true
	completedPods map[string]bool
	// deadline is the dealine time in which this operation should relinquish
	// its hold on the workflow so that an operation does not run for too long
	// and starve other workqueue items. It also enables workflow progress to
	// be periodically synced to the database.
	deadline time.Time
}

// maxOperationTime is the maximum time a workflow operation is allowed to run
// for before requeuing the workflow onto the workqueue.
const maxOperationTime time.Duration = 10 * time.Second

// wfScope contains the current scope of variables available when iterating steps in a workflow
type wfScope struct {
	tmpl  *wfv1.Template
	scope map[string]interface{}
}

// newWorkflowOperationCtx creates and initializes a new wfOperationCtx object.
func newWorkflowOperationCtx(wf *wfv1.Workflow, wfc *WorkflowController) *wfOperationCtx {
	woc := wfOperationCtx{
		wf:      wf.DeepCopyObject().(*wfv1.Workflow),
		orig:    wf,
		updated: false,
		log: log.WithFields(log.Fields{
			"workflow":  wf.ObjectMeta.Name,
			"namespace": wf.ObjectMeta.Namespace,
		}),
		controller:    wfc,
		globalParams:  make(map[string]string),
		completedPods: make(map[string]bool),
		deadline:      time.Now().UTC().Add(maxOperationTime),
	}

	if woc.wf.Status.Nodes == nil {
		woc.wf.Status.Nodes = make(map[string]wfv1.NodeStatus)
	}

	return &woc
}

// operateWorkflow is the main operator logic of a workflow.
// It evaluates the current state of the workflow, and its pods
// and decides how to proceed down the execution path.
// TODO: an error returned by this method should result in requeing the
// workflow to be retried at a later time
func (wfc *WorkflowController) operateWorkflow(wf *wfv1.Workflow) {
	if wf.ObjectMeta.Labels[common.LabelKeyCompleted] == "true" {
		// can get here if we already added the completed=true label,
		// but we are still draining the controller's workflow workqueue
		return
	}
	var err error
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	woc := newWorkflowOperationCtx(wf, wfc)
	defer woc.persistUpdates()
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(error); ok {
				woc.markWorkflowError(rerr, true)
			} else {
				woc.markWorkflowPhase(wfv1.NodeError, true, fmt.Sprintf("%v", r))
			}
			woc.log.Errorf("Recovered from panic: %+v\n%s", r, debug.Stack())
		}
	}()
	woc.log.Infof("Processing workflow")
	// Perform one-time workflow validation
	if woc.wf.Status.Phase == "" {
		woc.markWorkflowRunning()
		err := common.ValidateWorkflow(woc.wf)
		if err != nil {
			woc.markWorkflowFailed(fmt.Sprintf("invalid spec: %s", err.Error()))
			return
		}
	} else {
		err = woc.podReconciliation()
		if err != nil {
			woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
			// TODO: we need to re-add to the workqueue, but should happen in caller
			return
		}
	}
	woc.globalParams[common.GlobalVarWorkflowName] = wf.ObjectMeta.Name
	woc.globalParams[common.GlobalVarWorkflowUID] = string(wf.ObjectMeta.UID)
	for _, param := range wf.Spec.Arguments.Parameters {
		woc.globalParams["workflow.parameters."+param.Name] = *param.Value
	}

	err = woc.createPVCs()
	if err != nil {
		woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
		woc.markWorkflowError(err, true)
		return
	}
	err = woc.executeTemplate(wf.Spec.Entrypoint, wf.Spec.Arguments, wf.ObjectMeta.Name)
	if err != nil {
		if errors.IsCode(errors.CodeTimeout, err) {
			// A timeout indicates we took too long operating on the workflow.
			// Return so we can persist all the work we have done so far, and
			// requeue the workflow for another day. TODO: move this into the caller
			key, err := cache.MetaNamespaceKeyFunc(woc.wf)
			if err == nil {
				wfc.wfQueue.Add(key)
			}
			return
		}
		woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
	}
	node := woc.wf.Status.Nodes[woc.wf.NodeID(wf.ObjectMeta.Name)]
	if !node.Completed() {
		return
	}

	var onExitNode *wfv1.NodeStatus
	if wf.Spec.OnExit != "" {
		if node.Phase == wfv1.NodeSkipped {
			// treat skipped the same as Succeeded for workflow.status
			woc.globalParams[common.GlobalVarWorkflowStatus] = string(wfv1.NodeSucceeded)
		} else {
			woc.globalParams[common.GlobalVarWorkflowStatus] = string(node.Phase)
		}
		woc.log.Infof("Running OnExit handler: %s", wf.Spec.OnExit)
		onExitNodeName := wf.ObjectMeta.Name + ".onExit"
		err = woc.executeTemplate(wf.Spec.OnExit, wf.Spec.Arguments, onExitNodeName)
		if err != nil {
			if errors.IsCode(errors.CodeTimeout, err) {
				key, err := cache.MetaNamespaceKeyFunc(woc.wf)
				if err == nil {
					wfc.wfQueue.Add(key)
				}
				return
			}
			woc.log.Errorf("%s error: %+v", onExitNodeName, err)
		}
		xitNode := woc.wf.Status.Nodes[woc.wf.NodeID(onExitNodeName)]
		onExitNode = &xitNode
		if !onExitNode.Completed() {
			return
		}
	}

	err = woc.deletePVCs()
	if err != nil {
		woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
		// Mark the workflow with an error message and return, but intentionally do not
		// markCompletion so that we can retry PVC deletion (TODO: use workqueue.ReAdd())
		// This error phase may be cleared if a subsequent delete attempt is successful.
		woc.markWorkflowError(err, false)
		return
	}

	// If we get here, the workflow completed, all PVCs were deleted successfully, and
	// exit handlers were executed. We now need to infer the workflow phase from the
	// node phase.
	switch node.Phase {
	case wfv1.NodeSucceeded, wfv1.NodeSkipped:
		if onExitNode != nil && !onExitNode.Successful() {
			// if main workflow succeeded, but the exit node was unsuccessful
			// the workflow is now considered unsuccessful.
			woc.markWorkflowPhase(onExitNode.Phase, true, onExitNode.Message)
		} else {
			woc.markWorkflowSuccess()
		}
	case wfv1.NodeFailed:
		woc.markWorkflowFailed(node.Message)
	case wfv1.NodeError:
		woc.markWorkflowPhase(wfv1.NodeError, true, node.Message)
	default:
		// NOTE: we should never make it here because if the the node was 'Running'
		// we should have returned earlier.
		err = errors.InternalErrorf("Unexpected node phase %s: %+v", wf.ObjectMeta.Name, err)
		woc.markWorkflowError(err, true)
	}
}

// persistUpdates will PATCH a workflow with any updates made during workflow operation.
// It also labels any pods as completed if we have extracted everything we need from it.
func (woc *wfOperationCtx) persistUpdates() {
	if !woc.updated {
		return
	}
	oldData, err := json.Marshal(woc.orig)
	if err != nil {
		woc.log.Errorf("Error marshalling orig wf for patch: %+v", err)
		return
	}
	newData, err := json.Marshal(woc.wf)
	if err != nil {
		woc.log.Errorf("Error marshalling wf for patch: %+v", err)
		return
	}
	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		woc.log.Errorf("Error creating patch: %+v", err)
		return
	}
	if string(patchBytes) != "{}" {
		wfClient := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows(woc.wf.ObjectMeta.Namespace)
		_, err = wfClient.Patch(woc.wf.ObjectMeta.Name, types.MergePatchType, patchBytes)
		if err != nil {
			woc.log.Errorf("Error applying patch %s: %v", string(patchBytes), err)
			return
		}
		woc.log.Infof("Patch successful")
	}
	if len(woc.completedPods) > 0 {
		woc.log.Infof("Labeling %d completed pods", len(woc.completedPods))
		for podName := range woc.completedPods {
			err = common.AddPodLabel(woc.controller.kubeclientset, podName, woc.wf.ObjectMeta.Namespace, common.LabelKeyCompleted, "true")
			if err != nil {
				woc.log.Errorf("Failed adding completed label to pod %s: %+v", podName, err)
			}
		}
	}
}

func (woc *wfOperationCtx) processNodeRetries(node *wfv1.NodeStatus) error {
	if node.Completed() {
		return nil
	}
	lastChildNode, err := woc.getLastChildNode(node)
	if err != nil {
		return fmt.Errorf("Failed to find last child of node " + node.Name)
	}

	if lastChildNode == nil {
		return nil
	}

	if !lastChildNode.Completed() {
		// last child node is still running.
		return nil
	}

	if lastChildNode.Successful() {
		node.Outputs = lastChildNode.Outputs.DeepCopy()
		woc.markNodePhase(node.Name, wfv1.NodeSucceeded)
		return nil
	}

	if !lastChildNode.CanRetry() {
		woc.log.Infof("Node cannot be retried. Marking it failed")
		woc.markNodePhase(node.Name, wfv1.NodeFailed, lastChildNode.Message)
		return nil
	}

	if node.RetryStrategy.Limit != nil && int32(len(node.Children)) > *node.RetryStrategy.Limit {
		woc.log.Infoln("No more retries left. Failing...")
		woc.markNodePhase(node.Name, wfv1.NodeFailed, "No more retries left")
		return nil
	}

	woc.log.Infof("%d child nodes of %s failed. Trying again...", len(node.Children), node.Name)
	return nil
}

// podReconciliation is the process by which a workflow will examine all its related
// pods and update the node state before continuing the evaluation of the workflow.
// Records all pods which were observed completed, which will be labeled completed=true
// after successful persist of the workflow.
func (woc *wfOperationCtx) podReconciliation() error {
	podList, err := woc.getWorkflowPods(false)
	if err != nil {
		return err
	}
	seenPods := make(map[string]bool)
	for _, pod := range podList.Items {
		nodeNameForPod := pod.Annotations[common.AnnotationKeyNodeName]
		nodeID := woc.wf.NodeID(nodeNameForPod)
		seenPods[nodeID] = true
		if node, ok := woc.wf.Status.Nodes[nodeID]; ok {
			if newState := assessNodeStatus(&pod, &node); newState != nil {
				woc.wf.Status.Nodes[nodeID] = *newState
				woc.updated = true
			}
			if woc.wf.Status.Nodes[pod.ObjectMeta.Name].Completed() {
				woc.completedPods[pod.ObjectMeta.Name] = true
			}
		}
	}

	if len(podList.Items) > 0 {
		return nil
	}
	// If we get here, our initial query for pods related to this workflow returned nothing.
	// (note that our initial query excludes Pending pods for performance reasons since
	// there's generally no action needed to be taken on pods in a Pending state).
	// There are only a few scenarios where the pod list would have been empty:
	//  1. this workflow's pods are still pending (ideal case)
	//  2. this workflow's pods were deleted unbeknownst to the controller
	//  3. combination of 1 and 2
	// In order to detect scenario 2, we repeat the pod reconciliation process, this time
	// including Pending pods in the query. If one of our nodes does not show up in this list,
	// it implies that the pod was deleted without the controller seeing the event.
	woc.log.Info("Checking for deleted pods")
	podList, err = woc.getWorkflowPods(true)
	if err != nil {
		return err
	}
	for _, pod := range podList.Items {
		nodeNameForPod := pod.Annotations[common.AnnotationKeyNodeName]
		nodeID := woc.wf.NodeID(nodeNameForPod)
		seenPods[nodeID] = true
		if node, ok := woc.wf.Status.Nodes[nodeID]; ok {
			if newState := assessNodeStatus(&pod, &node); newState != nil {
				woc.wf.Status.Nodes[nodeID] = *newState
				woc.updated = true
			}
			if woc.wf.Status.Nodes[pod.ObjectMeta.Name].Completed() {
				woc.completedPods[pod.ObjectMeta.Name] = true
			}
		}
	}

	// Now iterate the workflow pod nodes which we still believe to be incomplete.
	// If the pod was not seen in the pod list, it means the pod was deleted and it
	// is now impossible to infer status. The only thing we can do at this point is
	// to mark the node with Error.
	for nodeID, node := range woc.wf.Status.Nodes {
		if len(node.Children) > 0 || node.Completed() {
			// node is not a pod, or it is already complete
			continue
		}

		if _, ok := seenPods[nodeID]; !ok {
			node.Message = "pod deleted"
			node.Phase = wfv1.NodeError
			woc.wf.Status.Nodes[nodeID] = node
			woc.log.Warnf("pod %s deleted", nodeID)
			woc.updated = true
		}
	}
	return nil
}

// getWorkflowPods returns all pods related to the current workflow
func (woc *wfOperationCtx) getWorkflowPods(includePending bool) (*apiv1.PodList, error) {
	options := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=false",
			common.LabelKeyWorkflow,
			woc.wf.ObjectMeta.Name,
			common.LabelKeyCompleted),
	}
	if !includePending {
		options.FieldSelector = "status.phase!=Pending"
	}
	podList, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).List(options)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return podList, nil
}

// assessNodeStatus compares the current state of a pod with its corresponding node
// and returns the new node status if something changed
func assessNodeStatus(pod *apiv1.Pod, node *wfv1.NodeStatus) *wfv1.NodeStatus {
	var newPhase wfv1.NodePhase
	var newDaemonStatus *bool
	var message string
	updated := false
	f := false
	switch pod.Status.Phase {
	case apiv1.PodPending:
		return nil
	case apiv1.PodSucceeded:
		newPhase = wfv1.NodeSucceeded
		newDaemonStatus = &f
	case apiv1.PodFailed:
		newPhase, message = inferFailedReason(pod)
		newDaemonStatus = &f
	case apiv1.PodRunning:
		tmplStr, ok := pod.Annotations[common.AnnotationKeyTemplate]
		if !ok {
			log.Warnf("%s missing template annotation", pod.ObjectMeta.Name)
			return nil
		}
		var tmpl wfv1.Template
		err := json.Unmarshal([]byte(tmplStr), &tmpl)
		if err != nil {
			log.Warnf("%s template annotation unreadable: %v", pod.ObjectMeta.Name, err)
			return nil
		}
		if tmpl.Daemon == nil || !*tmpl.Daemon {
			// incidental state change of a running pod. No need to inspect further
			return nil
		}
		// pod is running and template is marked daemon. check if everything is ready
		for _, ctrStatus := range pod.Status.ContainerStatuses {
			if !ctrStatus.Ready {
				return nil
			}
		}
		// proceed to mark node status as succeeded (and daemoned)
		newPhase = wfv1.NodeSucceeded
		t := true
		newDaemonStatus = &t
		log.Infof("Processing ready daemon pod: %v", pod.ObjectMeta.SelfLink)
	default:
		newPhase = wfv1.NodeError
		message = fmt.Sprintf("Unexpected pod phase for %s: %s", pod.ObjectMeta.Name, pod.Status.Phase)
		log.Error(message)
	}

	if newDaemonStatus != nil {
		if *newDaemonStatus == false {
			// if the daemon status switched to false, we prefer to just unset daemoned status field
			// (as opposed to setting it to false)
			newDaemonStatus = nil
		}
		if (newDaemonStatus != nil && node.Daemoned == nil) || (newDaemonStatus == nil && node.Daemoned != nil) {
			log.Infof("Setting node %v daemoned: %v -> %v", node, node.Daemoned, newDaemonStatus)
			node.Daemoned = newDaemonStatus
			updated = true
			if pod.Status.PodIP != "" && pod.Status.PodIP != node.PodIP {
				// only update Pod IP for daemoned nodes to reduce number of updates
				log.Infof("Updating daemon node %s IP %s -> %s", node, node.PodIP, pod.Status.PodIP)
				node.PodIP = pod.Status.PodIP
			}
		}
	}
	outputStr, ok := pod.Annotations[common.AnnotationKeyOutputs]
	if ok && node.Outputs == nil {
		updated = true
		log.Infof("Setting node %v outputs", node)
		var outputs wfv1.Outputs
		err := json.Unmarshal([]byte(outputStr), &outputs)
		if err != nil {
			log.Errorf("Failed to unmarshal %s outputs from pod annotation: %v", pod.Name, err)
			node.Phase = wfv1.NodeError
		} else {
			node.Outputs = &outputs
		}
	}
	if message != "" && node.Message != message {
		log.Infof("Updating node %s message: %s", node, message)
		node.Message = message
	}
	if node.Phase != newPhase {
		log.Infof("Updating node %s status %s -> %s", node, node.Phase, newPhase)
		updated = true
		node.Phase = newPhase
	}
	if node.Completed() && node.FinishedAt.IsZero() {
		updated = true
		if !node.IsDaemoned() {
			node.FinishedAt = getLatestFinishedAt(pod)
		}
		if node.FinishedAt.IsZero() {
			// If we get here, the container is daemoned so the
			// finishedAt might not have been set.
			node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
		}
	}
	if updated {
		return node
	}
	return nil
}

// getLatestFinishedAt returns the latest finishAt timestamp from all the
// containers of this pod.
func getLatestFinishedAt(pod *apiv1.Pod) metav1.Time {
	var latest metav1.Time
	for _, ctr := range pod.Status.InitContainerStatuses {
		if ctr.State.Terminated != nil && ctr.State.Terminated.FinishedAt.After(latest.Time) {
			latest = ctr.State.Terminated.FinishedAt
		}
	}
	for _, ctr := range pod.Status.ContainerStatuses {
		if ctr.State.Terminated != nil && ctr.State.Terminated.FinishedAt.After(latest.Time) {
			latest = ctr.State.Terminated.FinishedAt
		}
	}
	return latest
}

// inferFailedReason returns metadata about a Failed pod to be used in its NodeStatus
// Returns a tuple of the new phase and message
func inferFailedReason(pod *apiv1.Pod) (wfv1.NodePhase, string) {
	if pod.Status.Message != "" {
		// Pod has a nice error message. Use that.
		return wfv1.NodeFailed, pod.Status.Message
	}
	annotatedMsg := pod.Annotations[common.AnnotationKeyNodeMessage]
	// We only get one message to set for the overall node status.
	// If mutiple containers failed, in order of preference:
	// init, main (annotated), main (exit code), wait, sidecars
	for _, ctr := range pod.Status.InitContainerStatuses {
		if ctr.State.Terminated == nil {
			// We should never get here
			log.Warnf("Pod %s phase was Failed but %s did not have terminated state", pod.ObjectMeta.Name, ctr.Name)
			continue
		}
		if ctr.State.Terminated.ExitCode == 0 {
			continue
		}
		errMsg := fmt.Sprintf("failed to load artifacts")
		for _, msg := range []string{annotatedMsg, ctr.State.Terminated.Message} {
			if msg != "" {
				errMsg += ": " + msg
				break
			}
		}
		// NOTE: we consider artifact load issues as Error instead of Failed
		return wfv1.NodeError, errMsg
	}
	failMessages := make(map[string]string)
	for _, ctr := range pod.Status.ContainerStatuses {
		if ctr.State.Terminated == nil {
			// We should never get here
			log.Warnf("Pod %s phase was Failed but %s did not have terminated state", pod.ObjectMeta.Name, ctr.Name)
			continue
		}
		if ctr.State.Terminated.ExitCode == 0 {
			continue
		}
		if ctr.Name == common.WaitContainerName {
			errDetails := ""
			for _, msg := range []string{annotatedMsg, ctr.State.Terminated.Message} {
				if msg != "" {
					errDetails = msg
					break
				}
			}
			if errDetails == "" {
				// executor is expected to annotate a message to the pod upon any errors.
				// If we failed to see the annotated message, it is likely the pod ran with
				// insufficient privileges. Give a hint to that effect.
				errDetails = fmt.Sprintf("verify serviceaccount %s:%s has necessary privileges", pod.ObjectMeta.Namespace, pod.Spec.ServiceAccountName)
			}
			errMsg := fmt.Sprintf("failed to save outputs: %s", errDetails)
			failMessages[ctr.Name] = errMsg
		} else {
			if ctr.State.Terminated.Message != "" {
				failMessages[ctr.Name] = ctr.State.Terminated.Message
			} else {
				failMessages[ctr.Name] = fmt.Sprintf("failed with exit code %d", ctr.State.Terminated.ExitCode)
			}
		}
	}
	if failMsg, ok := failMessages[common.MainContainerName]; ok {
		_, ok = failMessages[common.WaitContainerName]
		isResourceTemplate := !ok
		if isResourceTemplate && annotatedMsg != "" {
			// For resource templates, we prefer the annotated message
			// over the vanilla exit code 1 error
			return wfv1.NodeFailed, annotatedMsg
		}
		return wfv1.NodeFailed, failMsg
	}
	if failMsg, ok := failMessages[common.WaitContainerName]; ok {
		return wfv1.NodeError, failMsg
	}

	// If we get here, both the main and wait container succeeded.
	// Identify the sidecar which failed and give proper message
	// NOTE: we may need to distinguish between the main container
	// succeeding and ignoring the sidecar statuses. This is because
	// executor may have had to forcefully terminate a sidecar
	// (kill -9), resulting in an non-zero exit code of a sidecar,
	// and overall pod status as failed. Or the sidecar is actually
	// *expected* to fail non-zero and should be ignored. Users may
	// want the option to consider a step failed only if the main
	// container failed. For now return the first failure.
	for _, failMsg := range failMessages {
		return wfv1.NodeFailed, failMsg
	}
	return wfv1.NodeFailed, fmt.Sprintf("pod failed for unknown reason")
}

func (woc *wfOperationCtx) createPVCs() error {
	if woc.wf.Status.Phase != wfv1.NodeRunning {
		// Only attempt to create PVCs if workflow transitioned to Running state
		// (e.g. passed validation, or didn't already complete)
		return nil
	}
	if len(woc.wf.Spec.VolumeClaimTemplates) == len(woc.wf.Status.PersistentVolumeClaims) {
		// If we have already created the PVCs, then there is nothing to do.
		// This will also handle the case where workflow has no volumeClaimTemplates.
		return nil
	}
	if len(woc.wf.Status.PersistentVolumeClaims) == 0 {
		woc.wf.Status.PersistentVolumeClaims = make([]apiv1.Volume, len(woc.wf.Spec.VolumeClaimTemplates))
	}
	pvcClient := woc.controller.kubeclientset.CoreV1().PersistentVolumeClaims(woc.wf.ObjectMeta.Namespace)
	for i, pvcTmpl := range woc.wf.Spec.VolumeClaimTemplates {
		if pvcTmpl.ObjectMeta.Name == "" {
			return errors.Errorf(errors.CodeBadRequest, "volumeClaimTemplates[%d].metadata.name is required", i)
		}
		pvcTmpl = *pvcTmpl.DeepCopy()
		// PVC name will be <workflowname>-<volumeclaimtemplatename>
		refName := pvcTmpl.ObjectMeta.Name
		pvcName := fmt.Sprintf("%s-%s", woc.wf.ObjectMeta.Name, pvcTmpl.ObjectMeta.Name)
		woc.log.Infof("Creating pvc %s", pvcName)
		pvcTmpl.ObjectMeta.Name = pvcName
		pvcTmpl.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(woc.wf, wfv1.SchemaGroupVersionKind),
		}
		pvc, err := pvcClient.Create(&pvcTmpl)
		if err != nil {
			woc.markNodeError(woc.wf.ObjectMeta.Name, err)
			return err
		}
		vol := apiv1.Volume{
			Name: refName,
			VolumeSource: apiv1.VolumeSource{
				PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.ObjectMeta.Name,
				},
			},
		}
		woc.wf.Status.PersistentVolumeClaims[i] = vol
		woc.updated = true
	}
	return nil
}

func (woc *wfOperationCtx) deletePVCs() error {
	totalPVCs := len(woc.wf.Status.PersistentVolumeClaims)
	if totalPVCs == 0 {
		// PVC list already empty. nothing to do
		return nil
	}
	pvcClient := woc.controller.kubeclientset.CoreV1().PersistentVolumeClaims(woc.wf.ObjectMeta.Namespace)
	newPVClist := make([]apiv1.Volume, 0)
	// Attempt to delete all PVCs. Record first error encountered
	var firstErr error
	for _, pvc := range woc.wf.Status.PersistentVolumeClaims {
		woc.log.Infof("Deleting PVC %s", pvc.PersistentVolumeClaim.ClaimName)
		err := pvcClient.Delete(pvc.PersistentVolumeClaim.ClaimName, nil)
		if err != nil {
			if !apierr.IsNotFound(err) {
				woc.log.Errorf("Failed to delete pvc %s: %v", pvc.PersistentVolumeClaim.ClaimName, err)
				newPVClist = append(newPVClist, pvc)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}
	if len(newPVClist) != totalPVCs {
		// we were successful in deleting one ore more PVCs
		woc.log.Infof("Deleted %d/%d PVCs", totalPVCs-len(newPVClist), totalPVCs)
		woc.wf.Status.PersistentVolumeClaims = newPVClist
		woc.updated = true
	}
	return firstErr
}

func (woc *wfOperationCtx) getLastChildNode(node *wfv1.NodeStatus) (*wfv1.NodeStatus, error) {
	if len(node.Children) <= 0 {
		return nil, nil
	}

	lastChildNodeName := node.Children[len(node.Children)-1]
	lastChildNode, ok := woc.wf.Status.Nodes[lastChildNodeName]
	if !ok {
		return nil, fmt.Errorf("Failed to find node " + lastChildNodeName)
	}

	return &lastChildNode, nil
}

func (woc *wfOperationCtx) getNode(nodeName string) wfv1.NodeStatus {
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if !ok {
		panic("Failed to find node " + nodeName)
	}

	return node
}

func (woc *wfOperationCtx) executeTemplate(templateName string, args wfv1.Arguments, nodeName string) error {
	woc.log.Debugf("Evaluating node %s: template: %s", nodeName, templateName)
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		woc.log.Debugf("Node %s already completed", nodeName)
		return nil
	}
	tmpl := woc.wf.GetTemplate(templateName)
	if tmpl == nil {
		err := errors.Errorf(errors.CodeBadRequest, "Node %v error: template '%s' undefined", node, templateName)
		woc.markNodeError(nodeName, err)
		return err
	}

	tmpl, err := common.ProcessArgs(tmpl, args, woc.globalParams, false)
	if err != nil {
		woc.markNodeError(nodeName, err)
		return err
	}

	switch tmpl.GetType() {
	case wfv1.TemplateTypeContainer:
		if ok {
			if node.RetryStrategy != nil {
				if err = woc.processNodeRetries(&node); err != nil {
					return err
				}

				// The updated node status could've changed. Get the latest copy of the node.
				node = woc.getNode(node.Name)
				log.Infof("Node %s: Status: %s", node.Name, node.Phase)
				if node.Completed() {
					return nil
				}
				lastChildNode, err := woc.getLastChildNode(&node)
				if err != nil {
					return err
				}
				if !lastChildNode.Completed() {
					// last child node is still running.
					return nil
				}
			} else {
				// There are no retries configured and there's already a node entry for the container.
				// This means the container was already scheduled (or had a create pod error). Nothing
				// to more to do with this node.
				return nil
			}
		}

		// If the user has specified retries, a special "retries" non-leaf node
		// is created. This node acts as the parent of all retries that will be
		// done for the container. The status of this node should be "Success"
		// if any of the retries succeed. Otherwise, it is "Failed".

		// TODO(shri): Mark the current node as a "retry" node
		// Create a new child node as the first attempt node and
		// run the template in that node.
		nodeToExecute := nodeName
		if tmpl.RetryStrategy != nil {
			node := woc.markNodePhase(nodeName, wfv1.NodeRunning)
			retries := wfv1.RetryStrategy{}
			node.RetryStrategy = &retries
			node.RetryStrategy.Limit = tmpl.RetryStrategy.Limit
			woc.wf.Status.Nodes[nodeID] = *node

			// Create new node as child of 'node'
			newContainerName := fmt.Sprintf("%s(%d)", nodeName, len(node.Children))
			woc.markNodePhase(newContainerName, wfv1.NodeRunning)
			woc.addChildNode(nodeName, newContainerName)
			nodeToExecute = newContainerName
		}

		// We have not yet created the pod
		err = woc.executeContainer(nodeToExecute, tmpl)
	case wfv1.TemplateTypeSteps:
		if !ok {
			node = *woc.markNodePhase(nodeName, wfv1.NodeRunning)
			woc.log.Infof("Initialized workflow node %v", node)
		}
		err = woc.executeSteps(nodeName, tmpl)
	case wfv1.TemplateTypeScript:
		err = woc.executeScript(nodeName, tmpl)
	case wfv1.TemplateTypeResource:
		err = woc.executeResource(nodeName, tmpl)
	default:
		err = errors.Errorf("Template '%s' missing specification", tmpl.Name)
		woc.markNodeError(nodeName, err)
	}
	if err != nil {
		return err
	}
	if time.Now().UTC().After(woc.deadline) {
		woc.log.Warnf("Deadline exceeded")
		return errors.New(errors.CodeTimeout, "Deadline exceeded")
	}
	return nil
}

// markWorkflowPhase is a convenience method to set the phase of the workflow with optional message
// optionally marks the workflow completed, which sets the finishedAt timestamp and completed label
func (woc *wfOperationCtx) markWorkflowPhase(phase wfv1.NodePhase, markCompleted bool, message ...string) {
	if woc.wf.Status.Phase != phase {
		woc.log.Infof("Updated phase %s -> %s", woc.wf.Status.Phase, phase)
		woc.updated = true
		woc.wf.Status.Phase = phase
		if woc.wf.ObjectMeta.Labels == nil {
			woc.wf.ObjectMeta.Labels = make(map[string]string)
		}
		woc.wf.ObjectMeta.Labels[common.LabelKeyPhase] = string(phase)
	}
	if woc.wf.Status.StartedAt.IsZero() {
		woc.updated = true
		woc.wf.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}
	if len(message) > 0 && woc.wf.Status.Message != message[0] {
		woc.log.Infof("Updated message %s -> %s", woc.wf.Status.Message, message[0])
		woc.updated = true
		woc.wf.Status.Message = message[0]
	}

	switch phase {
	case wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeError:
		if markCompleted {
			woc.log.Infof("Marking workflow completed")
			woc.wf.Status.FinishedAt = metav1.Time{Time: time.Now().UTC()}
			if woc.wf.ObjectMeta.Labels == nil {
				woc.wf.ObjectMeta.Labels = make(map[string]string)
			}
			woc.wf.ObjectMeta.Labels[common.LabelKeyCompleted] = "true"
			woc.updated = true
		}
	}
}

func (woc *wfOperationCtx) markWorkflowRunning() {
	woc.markWorkflowPhase(wfv1.NodeRunning, false)
}

func (woc *wfOperationCtx) markWorkflowSuccess() {
	woc.markWorkflowPhase(wfv1.NodeSucceeded, true)
}

func (woc *wfOperationCtx) markWorkflowFailed(message string) {
	woc.markWorkflowPhase(wfv1.NodeFailed, true, message)
}

func (woc *wfOperationCtx) markWorkflowError(err error, markCompleted bool) {
	woc.markWorkflowPhase(wfv1.NodeError, markCompleted, err.Error())
}

// markNodePhase marks a node with the given phase, creating the node if necessary and handles timestamps
func (woc *wfOperationCtx) markNodePhase(nodeName string, phase wfv1.NodePhase, message ...string) *wfv1.NodeStatus {
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if !ok {
		node = wfv1.NodeStatus{
			ID:        nodeID,
			Name:      nodeName,
			Phase:     phase,
			StartedAt: metav1.Time{Time: time.Now().UTC()},
		}
	} else {
		node.Phase = phase
	}
	if len(message) > 0 {
		node.Message = message[0]
	}
	if node.Completed() && node.FinishedAt.IsZero() {
		node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
	}
	woc.wf.Status.Nodes[nodeID] = node
	woc.updated = true
	woc.log.Debugf("Marked node %s %s", nodeName, phase)
	return &node
}

// markNodeError is a convenience method to mark a node with an error and set the message from the error
func (woc *wfOperationCtx) markNodeError(nodeName string, err error) *wfv1.NodeStatus {
	return woc.markNodePhase(nodeName, wfv1.NodeError, err.Error())
}

func (woc *wfOperationCtx) executeContainer(nodeName string, tmpl *wfv1.Template) error {
	woc.log.Infof("Executing node %s with container template: %v\n", nodeName, tmpl)
	pod, err := woc.createWorkflowPod(nodeName, *tmpl.Container, tmpl)
	if err != nil {
		woc.markNodeError(nodeName, err)
		return err
	}
	node := woc.markNodePhase(nodeName, wfv1.NodeRunning)
	node.StartedAt = pod.CreationTimestamp
	woc.wf.Status.Nodes[node.ID] = *node
	woc.log.Infof("Initialized container node %v", node)
	return nil
}

// getTemplateOutputsFromScope resolves a template's outputs from the scope of the template
func getTemplateOutputsFromScope(tmpl *wfv1.Template, scope *wfScope) (*wfv1.Outputs, error) {
	if !tmpl.Outputs.HasOutputs() {
		return nil, nil
	}
	var outputs wfv1.Outputs
	if len(tmpl.Outputs.Parameters) > 0 {
		outputs.Parameters = make([]wfv1.Parameter, 0)
		for _, param := range tmpl.Outputs.Parameters {
			val, err := scope.resolveParameter(param.ValueFrom.Parameter)
			if err != nil {
				return nil, err
			}
			param.Value = &val
			param.ValueFrom = nil
			outputs.Parameters = append(outputs.Parameters, param)
		}
	}
	if len(tmpl.Outputs.Artifacts) > 0 {
		outputs.Artifacts = make([]wfv1.Artifact, 0)
		for _, art := range tmpl.Outputs.Artifacts {
			resolvedArt, err := scope.resolveArtifact(art.From)
			if err != nil {
				return nil, err
			}
			resolvedArt.Name = art.Name
			outputs.Artifacts = append(outputs.Artifacts, *resolvedArt)
		}
	}
	return &outputs, nil
}

func (woc *wfOperationCtx) executeScript(nodeName string, tmpl *wfv1.Template) error {
	mainCtr := apiv1.Container{
		Image:   tmpl.Script.Image,
		Command: tmpl.Script.Command,
		Args:    []string{common.ExecutorScriptSourcePath},
	}
	pod, err := woc.createWorkflowPod(nodeName, mainCtr, tmpl)
	if err != nil {
		woc.markNodeError(nodeName, err)
		return err
	}
	node := woc.markNodePhase(nodeName, wfv1.NodeRunning)
	node.StartedAt = pod.CreationTimestamp
	woc.wf.Status.Nodes[node.ID] = *node
	woc.log.Infof("Initialized script node %v", node)
	return nil
}

func (wfs *wfScope) addParamToScope(key, val string) {
	wfs.scope[key] = val
}

func (wfs *wfScope) addArtifactToScope(key string, artifact wfv1.Artifact) {
	wfs.scope[key] = artifact
}

func (wfs *wfScope) resolveVar(v string) (interface{}, error) {
	v = strings.TrimPrefix(v, "{{")
	v = strings.TrimSuffix(v, "}}")
	if strings.HasPrefix(v, "steps.") {
		val, ok := wfs.scope[v]
		if !ok {
			return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: {{%s}}", v)
		}
		return val, nil
	}
	parts := strings.Split(v, ".")
	// HACK (assuming it is an input artifact)
	art := wfs.tmpl.Inputs.GetArtifactByName(parts[2])
	if art != nil {
		return *art, nil
	}
	return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve input artifact: {{%s}}", v)
}

func (wfs *wfScope) resolveParameter(v string) (string, error) {
	val, err := wfs.resolveVar(v)
	if err != nil {
		return "", err
	}
	valStr, ok := val.(string)
	if !ok {
		return "", errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not a string", v)
	}
	return valStr, nil
}

func (wfs *wfScope) resolveArtifact(v string) (*wfv1.Artifact, error) {
	val, err := wfs.resolveVar(v)
	if err != nil {
		return nil, err
	}
	valArt, ok := val.(wfv1.Artifact)
	if !ok {
		return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not an artifact", v)
	}
	return &valArt, nil
}

// addChildNode adds a nodeID as a child to a parent
func (woc *wfOperationCtx) addChildNode(parent string, child string) {
	parentID := woc.wf.NodeID(parent)
	childID := woc.wf.NodeID(child)
	node, ok := woc.wf.Status.Nodes[parentID]
	if !ok {
		panic(fmt.Sprintf("parent node %s not initialized", parent))
	}
	if node.Children == nil {
		node.Children = make([]string, 0)
	}
	for _, nodeID := range node.Children {
		if childID == nodeID {
			// already exists
			return
		}
	}
	node.Children = append(node.Children, childID)
	woc.wf.Status.Nodes[parentID] = node
	woc.updated = true
}

// executeResource is runs a kubectl command against a manifest
func (woc *wfOperationCtx) executeResource(nodeName string, tmpl *wfv1.Template) error {
	mainCtr := apiv1.Container{
		Image:   woc.controller.Config.ExecutorImage,
		Command: []string{"argoexec"},
		Args:    []string{"resource", tmpl.Resource.Action},
		VolumeMounts: []apiv1.VolumeMount{
			volumeMountPodMetadata,
		},
		Env: execEnvVars,
	}
	pod, err := woc.createWorkflowPod(nodeName, mainCtr, tmpl)
	if err != nil {
		woc.markNodeError(nodeName, err)
		return err
	}
	node := woc.markNodePhase(nodeName, wfv1.NodeRunning)
	node.StartedAt = pod.CreationTimestamp
	woc.wf.Status.Nodes[node.ID] = *node
	woc.log.Infof("Initialized resource node %v", node)
	return nil
}
