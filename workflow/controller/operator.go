package controller

import (
	"encoding/json"
	"fmt"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/util/retry"
	"github.com/argoproj/argo/workflow/common"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// log is an logrus logging context to corralate logs with a workflow
	log *log.Entry
	// controller reference to workflow controller
	controller *WorkflowController
	// globalParams holds any parameters that are available to be referenced
	// in the global scope (e.g. workflow.parameters.XXX).
	globalParams map[string]string
	// map of pods which need to be labeled with completed=true
	completedPods map[string]bool
	// deadline is the dealine time in which this operation should relinquish
	// its hold on the workflow so that an operation does not run for too long
	// and starve other workqueue items. It also enables workflow progress to
	// be periodically synced to the database.
	deadline time.Time
	// activePods tracks the number of active (Running/Pending) pods for controlling
	// parallelism
	activePods int64
}

var (
	// ErrDeadlineExceeded indicates the operation exceeded its deadline for execution
	ErrDeadlineExceeded = errors.New(errors.CodeTimeout, "Deadline exceeded")
	// ErrParallelismReached indicates this workflow reached its parallelism limit
	ErrParallelismReached = errors.New(errors.CodeForbidden, "Max parallelism reached")
)

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
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
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

// operate is the main operator logic of a workflow. It evaluates the current state of the workflow,
// and its pods and decides how to proceed down the execution path.
// TODO: an error returned by this method should result in requeuing the workflow to be retried at a
// later time
func (woc *wfOperationCtx) operate() {
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
		err := woc.podReconciliation()
		if err != nil {
			woc.log.Errorf("%s error: %+v", woc.wf.ObjectMeta.Name, err)
			// TODO: we need to re-add to the workqueue, but should happen in caller
			return
		}
	}
	if woc.wf.Spec.Suspend != nil && *woc.wf.Spec.Suspend {
		woc.log.Infof("workflow suspended")
		return
	}
	if woc.wf.Spec.Parallelism != nil {
		woc.activePods = woc.countActivePods()
	}

	woc.setGlobalParameters()

	err := woc.createPVCs()
	if err != nil {
		woc.log.Errorf("%s pvc create error: %+v", woc.wf.ObjectMeta.Name, err)
		woc.markWorkflowError(err, true)
		return
	}
	var workflowStatus wfv1.NodePhase
	var workflowMessage string
	node, _ := woc.executeTemplate(woc.wf.Spec.Entrypoint, woc.wf.Spec.Arguments, woc.wf.ObjectMeta.Name, "")
	if node == nil || !node.Completed() {
		// node can be nil if a workflow created immediately in a parallelism == 0 state
		return
	}
	workflowStatus = node.Phase
	workflowMessage = node.Message

	var onExitNode *wfv1.NodeStatus
	if woc.wf.Spec.OnExit != "" {
		if workflowStatus == wfv1.NodeSkipped {
			// treat skipped the same as Succeeded for workflow.status
			woc.globalParams[common.GlobalVarWorkflowStatus] = string(wfv1.NodeSucceeded)
		} else {
			woc.globalParams[common.GlobalVarWorkflowStatus] = string(workflowStatus)
		}
		woc.log.Infof("Running OnExit handler: %s", woc.wf.Spec.OnExit)
		onExitNodeName := woc.wf.ObjectMeta.Name + ".onExit"
		onExitNode, _ = woc.executeTemplate(woc.wf.Spec.OnExit, woc.wf.Spec.Arguments, onExitNodeName, "")
		if onExitNode == nil || !onExitNode.Completed() {
			return
		}
	}

	err = woc.deletePVCs()
	if err != nil {
		woc.log.Errorf("%s error: %+v", woc.wf.ObjectMeta.Name, err)
		// Mark the workflow with an error message and return, but intentionally do not
		// markCompletion so that we can retry PVC deletion (TODO: use workqueue.ReAdd())
		// This error phase may be cleared if a subsequent delete attempt is successful.
		woc.markWorkflowError(err, false)
		return
	}

	// If we get here, the workflow completed, all PVCs were deleted successfully, and
	// exit handlers were executed. We now need to infer the workflow phase from the
	// node phase.
	switch workflowStatus {
	case wfv1.NodeSucceeded, wfv1.NodeSkipped:
		if onExitNode != nil && !onExitNode.Successful() {
			// if main workflow succeeded, but the exit node was unsuccessful
			// the workflow is now considered unsuccessful.
			woc.markWorkflowPhase(onExitNode.Phase, true, onExitNode.Message)
		} else {
			woc.markWorkflowSuccess()
		}
	case wfv1.NodeFailed:
		woc.markWorkflowFailed(workflowMessage)
	case wfv1.NodeError:
		woc.markWorkflowPhase(wfv1.NodeError, true, workflowMessage)
	default:
		// NOTE: we should never make it here because if the the node was 'Running'
		// we should have returned earlier.
		err = errors.InternalErrorf("Unexpected node phase %s: %+v", woc.wf.ObjectMeta.Name, err)
		woc.markWorkflowError(err, true)
	}
}

// setGlobalParameters sets the globalParam map with global parameters
func (woc *wfOperationCtx) setGlobalParameters() {
	woc.globalParams[common.GlobalVarWorkflowName] = woc.wf.ObjectMeta.Name
	woc.globalParams[common.GlobalVarWorkflowNamespace] = woc.wf.ObjectMeta.Namespace
	woc.globalParams[common.GlobalVarWorkflowUID] = string(woc.wf.ObjectMeta.UID)
	for _, param := range woc.wf.Spec.Arguments.Parameters {
		woc.globalParams["workflow.parameters."+param.Name] = *param.Value
	}
	if woc.wf.Status.Outputs != nil {
		for _, param := range woc.wf.Status.Outputs.Parameters {
			woc.globalParams["workflow.outputs.parameters."+param.Name] = *param.Value
		}
	}
}

func (woc *wfOperationCtx) getNodeByName(nodeName string) *wfv1.NodeStatus {
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// persistUpdates will update a workflow with any updates made during workflow operation.
// It also labels any pods as completed if we have extracted everything we need from it.
// NOTE: a previous implementation used Patch instead of Update, but Patch does not work with
// the fake CRD clientset which makes unit testing extremely difficult.
func (woc *wfOperationCtx) persistUpdates() {
	if !woc.updated {
		return
	}
	wfClient := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows(woc.wf.ObjectMeta.Namespace)
	_, err := wfClient.Update(woc.wf)
	if err != nil {
		woc.log.Warnf("Error updating workflow: %v", err)
		if !apierr.IsConflict(err) {
			return
		}
		woc.log.Info("Re-appying updates on latest version and retrying update")
		err = woc.reapplyUpdate(wfClient)
		if err != nil {
			woc.log.Infof("Failed to re-apply update: %+v", err)
			return
		}
	}
	woc.log.Info("Workflow update successful")

	// HACK(jessesuen) after we successfully persist an update to the workflow, the informer's
	// cache is now invalid. It's very common that we will need to immediately re-operate on a
	// workflow due to queuing by the pod workers. The following sleep gives a *chance* for the
	// informer's cache to catch up to the version of the workflow we just persisted. Without
	// this sleep, the next worker to work on this workflow will very likely operate on a stale
	// object and redo work.
	time.Sleep(1 * time.Second)

	// It is important that we *never* label pods as completed until we successfully updated the workflow
	// Failing to do so means we can have inconsistent state.
	for podName := range woc.completedPods {
		woc.controller.completedPods <- fmt.Sprintf("%s/%s", woc.wf.ObjectMeta.Namespace, podName)
	}
}

// reapplyUpdate GETs the latest version of the workflow, re-applies the updates and
// retries the UPDATE multiple times. For reasoning behind this technique, see:
// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
func (woc *wfOperationCtx) reapplyUpdate(wfClient v1alpha1.WorkflowInterface) error {
	// First generate the patch
	oldData, err := json.Marshal(woc.orig)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	newData, err := json.Marshal(woc.wf)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	// Next get latest version of the workflow, apply the patch and retyr the Update
	attempt := 1
	for {
		currWf, err := wfClient.Get(woc.wf.ObjectMeta.Name, metav1.GetOptions{})
		if !retry.IsRetryableKubeAPIError(err) {
			return errors.InternalWrapError(err)
		}
		currWfBytes, err := json.Marshal(currWf)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		newWfBytes, err := jsonpatch.MergePatch(currWfBytes, patchBytes)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		var newWf wfv1.Workflow
		err = json.Unmarshal(newWfBytes, &newWf)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		_, err = wfClient.Update(&newWf)
		if err == nil {
			woc.log.Infof("Update retry attempt %d successful", attempt)
			return nil
		}
		attempt++
		woc.log.Warnf("Update retry attempt %d failed: %v", attempt, err)
		if attempt > 5 {
			return err
		}
	}
}

// requeue this workflow onto the workqueue for later processing
func (woc *wfOperationCtx) requeue() {
	key, err := cache.MetaNamespaceKeyFunc(woc.wf)
	if err != nil {
		woc.log.Errorf("Failed to requeue workflow %s: %v", woc.wf.ObjectMeta.Name, err)
		return
	}
	woc.controller.wfQueue.Add(key)
}

func (woc *wfOperationCtx) processNodeRetries(node *wfv1.NodeStatus, retryStrategy wfv1.RetryStrategy) error {
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
		woc.wf.Status.Nodes[node.ID] = *node
		woc.markNodePhase(node.Name, wfv1.NodeSucceeded)
		return nil
	}

	if !lastChildNode.CanRetry() {
		woc.log.Infof("Node cannot be retried. Marking it failed")
		woc.markNodePhase(node.Name, wfv1.NodeFailed, lastChildNode.Message)
		return nil
	}

	if retryStrategy.Limit != nil && int32(len(node.Children)) > *retryStrategy.Limit {
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
	podList, err := woc.getRunningWorkflowPods()
	if err != nil {
		return err
	}
	seenPods := make(map[string]bool)

	performAssessment := func(pod *apiv1.Pod) {
		nodeNameForPod := pod.Annotations[common.AnnotationKeyNodeName]
		nodeID := woc.wf.NodeID(nodeNameForPod)
		seenPods[nodeID] = true
		if node, ok := woc.wf.Status.Nodes[nodeID]; ok {
			if newState := assessNodeStatus(pod, &node); newState != nil {
				woc.wf.Status.Nodes[nodeID] = *newState
				if node.Outputs != nil {
					for _, param := range node.Outputs.Parameters {
						woc.addParamToGlobalScope(param)
					}
					for _, art := range node.Outputs.Artifacts {
						woc.addArtifactToGlobalScope(art)
					}
				}
				woc.updated = true
			}
			if woc.wf.Status.Nodes[pod.ObjectMeta.Name].Completed() {
				woc.completedPods[pod.ObjectMeta.Name] = true
			}
		}
	}

	for _, pod := range podList.Items {
		performAssessment(&pod)
	}

	if len(podList.Items) > 0 {
		// if we saw related pods, no need to check for deleted pods yet.
		// we will get to them eventually.
		return nil
	}
	// If we get here, our initial query for pods related to this workflow returned nothing.
	// Note that our initial query excludes Pending/completed=true pods for performance reasons
	// since there's generally no action needed to be taken on pending pods or ones we have
	// already processed (completed=true).
	// There are a few scenarios where the pod list would have been empty:
	//  1. workflow's pods are still pending (best case scenario)
	//  2. workflow's pods were deleted unbeknownst to the controller
	//  3. workflow's pods were marked completed=true, but we are operating on a stale workflow object
	//  4. combination of any the above scenarios
	// In order to detect deleted pods, we repeat the pod reconciliation process, this time
	// including ALL workflow pods in the query. If any one of our nodes does not show up in this
	// returned list, it implies that the pod was deleted without the controller seeing the event.
	woc.log.Info("Checking for deleted pods")
	podList, err = woc.getAllWorkflowPods()
	if err != nil {
		return err
	}
	// Repeat the node assessment
	for _, pod := range podList.Items {
		performAssessment(&pod)
	}

	// Now iterate the workflow pod nodes which we still believe to be incomplete.
	// If the pod was not seen in the pod list, it means the pod was deleted and it
	// is now impossible to infer status. The only thing we can do at this point is
	// to mark the node with Error.
	for nodeID, node := range woc.wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod || node.Completed() {
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

// countActivePods counts the number of active (Pending/Running) pods.
// Optionally restricts it to a template invocation (boundaryID)
func (woc *wfOperationCtx) countActivePods(boundaryIDs ...string) int64 {
	var boundaryID = ""
	if len(boundaryIDs) > 0 {
		boundaryID = boundaryIDs[0]
	}
	var activePods int64
	// if we care about parallelism, count the active pods at the template level
	for _, node := range woc.wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod || node.Phase != wfv1.NodeRunning {
			continue
		}
		if boundaryID != "" && node.BoundaryID != boundaryID {
			continue
		}
		activePods++
	}
	return activePods
}

// getRunningWorkflowPods returns running pods of the current workflow.
func (woc *wfOperationCtx) getRunningWorkflowPods() (*apiv1.PodList, error) {
	options := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=false",
			common.LabelKeyWorkflow,
			woc.wf.ObjectMeta.Name,
			common.LabelKeyCompleted),
		FieldSelector: "status.phase!=Pending",
	}
	podList, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).List(options)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return podList, nil
}

// getAllWorkflowPods returns all pods related to the current workflow
func (woc *wfOperationCtx) getAllWorkflowPods() (*apiv1.PodList, error) {
	options := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s",
			common.LabelKeyWorkflow,
			woc.wf.ObjectMeta.Name),
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
	// If multiple containers failed, in order of preference:
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
				errMsg := fmt.Sprintf("failed with exit code %d", ctr.State.Terminated.ExitCode)
				if ctr.Name != common.MainContainerName {
					errMsg = fmt.Sprintf("sidecar '%s' %s", ctr.Name, errMsg)
				}
				failMessages[ctr.Name] = errMsg
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

// executeTemplate executes the template with the given arguments and returns the created NodeStatus
// for the created node (if created). Nodes may not be created if parallelism or deadline exceeded.
// nodeName is the name to be used as the name of the node, and boundaryID indicates which template
// boundary this node belongs to.
func (woc *wfOperationCtx) executeTemplate(templateName string, args wfv1.Arguments, nodeName string, boundaryID string) (*wfv1.NodeStatus, error) {
	woc.log.Debugf("Evaluating node %s: template: %s", nodeName, templateName)
	node := woc.getNodeByName(nodeName)
	if node != nil && node.Completed() {
		woc.log.Debugf("Node %s already completed", nodeName)
		return node, nil
	}

	// Check if we took too long operating on this workflow and immediately return if we did
	if time.Now().UTC().After(woc.deadline) {
		woc.log.Warnf("Deadline exceeded")
		woc.requeue()
		return node, ErrDeadlineExceeded
	}

	// Check if we exceeded template or workflow parallelism and immediately return if we did
	tmpl := woc.wf.GetTemplate(templateName)
	if tmpl == nil {
		err := errors.Errorf(errors.CodeBadRequest, "Node %v error: template '%s' undefined", node, templateName)
		return woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, "", boundaryID, wfv1.NodeError, err.Error()), err
	}
	if err := woc.checkParallelism(tmpl, node, boundaryID); err != nil {
		return node, err
	}

	// Perform parameter substitution of the template
	localParams := make(map[string]string)
	if tmpl.IsPodType() {
		localParams["pod.name"] = woc.wf.NodeID(nodeName)
	}
	tmpl, err := common.ProcessArgs(tmpl, args, woc.globalParams, localParams, false)
	if err != nil {
		return woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, templateName, boundaryID, wfv1.NodeError, err.Error()), err
	}

	// If the user has specified retries, node becomes a special retry node.
	// This node acts as a parent of all retries that will be done for
	// the container. The status of this node should be "Success" if any
	// of the retries succeed. Otherwise, it is "Failed".
	workNodeName := nodeName
	retryNodeName := ""
	if tmpl.IsLeaf() && tmpl.RetryStrategy != nil {
		retryNodeName = nodeName
		if node == nil {
			node = woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", boundaryID, wfv1.NodeRunning)
		}
		if err := woc.processNodeRetries(node, *tmpl.RetryStrategy); err != nil {
			woc.markNodeError(nodeName, err)
			return node, err
		}
		node = woc.getNodeByName(retryNodeName)
		woc.log.Infof("Node %s: Status: %s", retryNodeName, node.Phase)
		// The retry node might have completed by now.
		if node.Completed() {
			return node, nil
		}
		lastChildNode, err := woc.getLastChildNode(node)
		if err != nil {
			woc.markNodeError(retryNodeName, err)
			return node, err
		}
		if lastChildNode != nil && !lastChildNode.Completed() {
			// Last child node is still running.
			return node, nil
		}
		childNodeName := fmt.Sprintf("%s(%d)", retryNodeName, len(node.Children))
		// All work is done in a child
		workNodeName = childNodeName
	}

	switch tmpl.GetType() {
	case wfv1.TemplateTypeContainer:
		node = woc.executeContainer(workNodeName, tmpl, boundaryID)
	case wfv1.TemplateTypeSteps:
		node = woc.executeSteps(workNodeName, tmpl, boundaryID)
	case wfv1.TemplateTypeScript:
		node = woc.executeScript(workNodeName, tmpl, boundaryID)
	case wfv1.TemplateTypeResource:
		node = woc.executeResource(workNodeName, tmpl, boundaryID)
	case wfv1.TemplateTypeDAG:
		node = woc.executeDAG(workNodeName, tmpl, boundaryID)
	case wfv1.TemplateTypeSuspend:
		node = woc.executeSuspend(workNodeName, tmpl, boundaryID)
	default:
		err = errors.Errorf(errors.CodeBadRequest, "Template '%s' missing specification", tmpl.Name)
		node = woc.initializeNode(workNodeName, wfv1.NodeTypeSkipped, templateName, boundaryID, wfv1.NodeError, err.Error())
	}

	// Swap the node back to retry node and add worker node as child.
	if retryNodeName != "" {
		woc.addChildNode(retryNodeName, workNodeName)
		node = woc.getNodeByName(retryNodeName)
	}

	// Set the input values to the node. This is presented in the UI
	if tmpl.Inputs.HasInputs() && node.Inputs == nil {
		node.Inputs = &tmpl.Inputs
		woc.wf.Status.Nodes[node.ID] = *node
		woc.updated = true
	}
	return node, nil
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

// stepsOrDagSeparator identifies if a node name starts with our naming convention separator from
// DAG or steps templates. Will match stings with prefix like: [0]. or .
var stepsOrDagSeparator = regexp.MustCompile(`^(\[\d+\])?\.`)

func (woc *wfOperationCtx) initializeNode(nodeName string, nodeType wfv1.NodeType, templateName string, boundaryID string, phase wfv1.NodePhase, messages ...string) *wfv1.NodeStatus {
	nodeID := woc.wf.NodeID(nodeName)
	_, ok := woc.wf.Status.Nodes[nodeID]
	if ok {
		panic(fmt.Sprintf("node %s already initialized", nodeName))
	}
	node := wfv1.NodeStatus{
		ID:           nodeID,
		Name:         nodeName,
		TemplateName: templateName,
		Type:         nodeType,
		BoundaryID:   boundaryID,
		Phase:        phase,
		StartedAt:    metav1.Time{Time: time.Now().UTC()},
	}
	if boundaryNode, ok := woc.wf.Status.Nodes[boundaryID]; ok {
		node.DisplayName = strings.TrimPrefix(node.Name, boundaryNode.Name)
		if stepsOrDagSeparator.MatchString(node.DisplayName) {
			node.DisplayName = stepsOrDagSeparator.ReplaceAllString(node.DisplayName, "")
		}
	} else {
		node.DisplayName = nodeName
	}

	if node.Completed() && node.FinishedAt.IsZero() {
		node.FinishedAt = node.StartedAt
	}
	var message string
	if len(messages) > 0 {
		message = fmt.Sprintf(" (message: %s)", messages[0])
		node.Message = messages[0]
	}
	woc.wf.Status.Nodes[nodeID] = node
	woc.log.Infof("%s node %s initialized %s%s", node.Type, node, node.Phase, message)
	woc.updated = true
	return &node
}

// markNodePhase marks a node with the given phase, creating the node if necessary and handles timestamps
func (woc *wfOperationCtx) markNodePhase(nodeName string, phase wfv1.NodePhase, message ...string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node == nil {
		panic(fmt.Sprintf("node %s uninitialized", nodeName))
	}
	if node.Phase != phase {
		woc.log.Infof("node %s phase %s -> %s", node, node.Phase, phase)
		node.Phase = phase
		woc.updated = true
	}
	if len(message) > 0 {
		if message[0] != node.Message {
			woc.log.Infof("node %s message: %s", node, message[0])
			node.Message = message[0]
			woc.updated = true
		}
	}
	if node.Completed() && node.FinishedAt.IsZero() {
		node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
		woc.log.Infof("node %s finished: %s", node, node.FinishedAt)
		woc.updated = true
	}
	woc.wf.Status.Nodes[node.ID] = *node
	return node
}

// markNodeError is a convenience method to mark a node with an error and set the message from the error
func (woc *wfOperationCtx) markNodeError(nodeName string, err error) *wfv1.NodeStatus {
	return woc.markNodePhase(nodeName, wfv1.NodeError, err.Error())
}

// checkParallelism checks if the given template is able to be executed, considering the current active pods and workflow/template parallelism
func (woc *wfOperationCtx) checkParallelism(tmpl *wfv1.Template, node *wfv1.NodeStatus, boundaryID string) error {
	if woc.wf.Spec.Parallelism != nil && woc.activePods >= *woc.wf.Spec.Parallelism {
		woc.log.Infof("workflow active pod spec parallelism reached %d/%d", woc.activePods, *woc.wf.Spec.Parallelism)
		return ErrParallelismReached
	}
	// TODO: repeated calls to countActivePods is not optimal
	switch tmpl.GetType() {
	case wfv1.TemplateTypeDAG, wfv1.TemplateTypeSteps:
		// if we are about to execute a DAG/Steps template, make sure we havent already reached our limit
		if tmpl.Parallelism != nil && node != nil {
			templateActivePods := woc.countActivePods(node.ID)
			if templateActivePods >= *tmpl.Parallelism {
				woc.log.Infof("template (node %s) active pod parallelism reached %d/%d", node.ID, templateActivePods, *tmpl.Parallelism)
				return ErrParallelismReached
			}
		}
	default:
		// if we are about to execute a pod, make our parent hasn't reached it's limit
		if boundaryID != "" {
			boundaryNode := woc.wf.Status.Nodes[boundaryID]
			boundaryTemplate := woc.wf.GetTemplate(boundaryNode.TemplateName)
			if boundaryTemplate.Parallelism != nil {
				templateActivePods := woc.countActivePods(boundaryID)
				woc.log.Debugf("counted %d/%d active pods in boundary %s", templateActivePods, *boundaryTemplate.Parallelism, boundaryID)
				if templateActivePods >= *boundaryTemplate.Parallelism {
					woc.log.Infof("template (node %s) active pod parallelism reached %d/%d", boundaryID, templateActivePods, *boundaryTemplate.Parallelism)
					return ErrParallelismReached
				}
			}
		}

	}
	return nil
}

func (woc *wfOperationCtx) executeContainer(nodeName string, tmpl *wfv1.Template, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node != nil {
		return node
	}
	woc.log.Debugf("Executing node %s with container template: %v\n", nodeName, tmpl)
	_, err := woc.createWorkflowPod(nodeName, *tmpl.Container, tmpl)
	if err != nil {
		return woc.initializeNode(nodeName, wfv1.NodeTypePod, tmpl.Name, boundaryID, wfv1.NodeError, err.Error())
	}
	return woc.initializeNode(nodeName, wfv1.NodeTypePod, tmpl.Name, boundaryID, wfv1.NodeRunning)
}

func (woc *wfOperationCtx) getOutboundNodes(nodeID string) []string {
	node := woc.wf.Status.Nodes[nodeID]
	switch node.Type {
	case wfv1.NodeTypePod, wfv1.NodeTypeSkipped, wfv1.NodeTypeSuspend:
		return []string{node.ID}
	case wfv1.NodeTypeRetry:
		numChildren := len(node.Children)
		if numChildren > 0 {
			return []string{node.Children[numChildren-1]}
		}
	}
	outbound := make([]string, 0)
	for _, outboundNodeID := range node.OutboundNodes {
		outNode := woc.wf.Status.Nodes[outboundNodeID]
		if outNode.Type == wfv1.NodeTypePod {
			outbound = append(outbound, outboundNodeID)
		} else {
			subOutIDs := woc.getOutboundNodes(outboundNodeID)
			for _, subOutID := range subOutIDs {
				outbound = append(outbound, subOutID)
			}
		}
	}
	return outbound
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

func (woc *wfOperationCtx) executeScript(nodeName string, tmpl *wfv1.Template, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node != nil {
		return node
	}
	mainCtr := tmpl.Script.Container
	mainCtr.Args = append(mainCtr.Args, common.ExecutorScriptSourcePath)
	_, err := woc.createWorkflowPod(nodeName, mainCtr, tmpl)
	if err != nil {
		return woc.initializeNode(nodeName, wfv1.NodeTypePod, tmpl.Name, boundaryID, wfv1.NodeError, err.Error())
	}
	return woc.initializeNode(nodeName, wfv1.NodeTypePod, tmpl.Name, boundaryID, wfv1.NodeRunning)
}

// processNodeOutputs adds all of a nodes outputs to the local scope with the given prefix, as well
// as the global scope, if specified with a globalName
func (woc *wfOperationCtx) processNodeOutputs(wfs *wfScope, prefix string, node *wfv1.NodeStatus) {
	if node.PodIP != "" {
		key := fmt.Sprintf("%s.ip", prefix)
		wfs.addParamToScope(key, node.PodIP)
	}
	if node.Outputs == nil {
		return
	}
	if node.Outputs.Result != nil {
		key := fmt.Sprintf("%s.outputs.result", prefix)
		wfs.addParamToScope(key, *node.Outputs.Result)
	}
	for _, outParam := range node.Outputs.Parameters {
		key := fmt.Sprintf("%s.outputs.parameters.%s", prefix, outParam.Name)
		wfs.addParamToScope(key, *outParam.Value)
		woc.addParamToGlobalScope(outParam)
	}
	for _, outArt := range node.Outputs.Artifacts {
		key := fmt.Sprintf("%s.outputs.artifacts.%s", prefix, outArt.Name)
		wfs.addArtifactToScope(key, outArt)
		woc.addArtifactToGlobalScope(outArt)
	}
}

type loopNodes []wfv1.NodeStatus

func (n loopNodes) Len() int {
	return len(n)
}

func parseLoopIndex(s string) int {
	s = strings.SplitN(s, "(", 2)[1]
	s = strings.SplitN(s, ":", 2)[0]
	val, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse '%s' as int: %v", s, err))
	}
	return val
}
func (n loopNodes) Less(i, j int) bool {
	left := parseLoopIndex(n[i].DisplayName)
	right := parseLoopIndex(n[j].DisplayName)
	return left < right
}
func (n loopNodes) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

// processAggregateNodeOutputs adds the aggregated outputs of a withItems/withParam template as a
// parameter in the form of a JSON list
func (woc *wfOperationCtx) processAggregateNodeOutputs(stepsCtx *stepsContext, prefix, childNodePrefix string) {
	paramList := make([]map[string]string, 0)
	var childNodes []wfv1.NodeStatus
	for _, node := range woc.wf.Status.Nodes {
		if node.BoundaryID == stepsCtx.boundaryID && strings.HasPrefix(node.Name, childNodePrefix) {
			childNodes = append(childNodes, node)
		}
	}
	if len(childNodes) == 0 {
		return
	}

	// need to sort the child node list so that the order of outputs are preserved
	sort.Sort(loopNodes(childNodes))

	for _, node := range childNodes {
		if node.Outputs == nil || len(node.Outputs.Parameters) == 0 {
			continue
		}
		param := make(map[string]string)
		for _, p := range node.Outputs.Parameters {
			param[p.Name] = *p.Value
		}
		paramList = append(paramList, param)
	}
	outputsJSON, _ := json.Marshal(paramList)
	key := fmt.Sprintf("%s.outputs.parameters", prefix)
	stepsCtx.scope.addParamToScope(key, string(outputsJSON))
}

// addParamToGlobalScope exports any desired node outputs to the global scope, and adds it to the global outputs.
func (woc *wfOperationCtx) addParamToGlobalScope(param wfv1.Parameter) {
	if param.GlobalName == "" {
		return
	}
	index := -1
	if woc.wf.Status.Outputs != nil {
		for i, gParam := range woc.wf.Status.Outputs.Parameters {
			if gParam.Name == param.GlobalName {
				index = i
				break
			}
		}
	} else {
		woc.wf.Status.Outputs = &wfv1.Outputs{}
	}
	woc.updated = true
	paramName := fmt.Sprintf("workflow.outputs.parameters.%s", param.GlobalName)
	woc.globalParams[paramName] = *param.Value
	if index == -1 {
		woc.log.Infof("setting %s: '%s'", paramName, *param.Value)
		gParam := wfv1.Parameter{Name: param.GlobalName, Value: param.Value}
		woc.wf.Status.Outputs.Parameters = append(woc.wf.Status.Outputs.Parameters, gParam)
	} else {
		woc.log.Infof("overwriting %s: '%s' -> '%s'", paramName, *woc.wf.Status.Outputs.Parameters[index].Value, *param.Value)
		woc.wf.Status.Outputs.Parameters[index].Value = param.Value
	}
}

// addArtifactToGlobalScope exports any desired node outputs to the global scope
// Optionally adds to a local scope if supplied
func (woc *wfOperationCtx) addArtifactToGlobalScope(art wfv1.Artifact) {
	if art.GlobalName == "" {
		return
	}
	woc.updated = true
	globalArtName := fmt.Sprintf("workflow.outputs.artifacts.%s", art.GlobalName)
	if woc.wf.Status.Outputs != nil {
		for i, gArt := range woc.wf.Status.Outputs.Artifacts {
			if gArt.Name == art.GlobalName {
				// global output already exists. overwrite the value
				art.Name = art.GlobalName
				art.GlobalName = ""
				art.Path = ""
				woc.wf.Status.Outputs.Artifacts[i] = art
				woc.log.Infof("overwriting %s: %s", globalArtName, art)
				return
			}
		}
	} else {
		woc.wf.Status.Outputs = &wfv1.Outputs{}
	}
	// global output does not yet exist
	art.Name = art.GlobalName
	art.GlobalName = ""
	art.Path = ""
	woc.log.Infof("setting %s: %s", globalArtName, art)
	woc.wf.Status.Outputs.Artifacts = append(woc.wf.Status.Outputs.Artifacts, art)
}

// replaceMap returns a replacement map of strings intended to be used simple string substitution
func (wfs *wfScope) replaceMap() map[string]string {
	replaceMap := make(map[string]string)
	for key, val := range wfs.scope {
		valStr, ok := val.(string)
		if ok {
			replaceMap[key] = valStr
		}
	}
	return replaceMap
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
	if strings.HasPrefix(v, "steps.") || strings.HasPrefix(v, "tasks.") {
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
// parent and child are both node names
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
func (woc *wfOperationCtx) executeResource(nodeName string, tmpl *wfv1.Template, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node != nil {
		return node
	}
	mainCtr := apiv1.Container{
		Image:   woc.controller.Config.ExecutorImage,
		Command: []string{"argoexec"},
		Args:    []string{"resource", tmpl.Resource.Action},
		VolumeMounts: []apiv1.VolumeMount{
			volumeMountPodMetadata,
		},
		Env: execEnvVars,
	}
	_, err := woc.createWorkflowPod(nodeName, mainCtr, tmpl)
	if err != nil {
		return woc.initializeNode(nodeName, wfv1.NodeTypePod, tmpl.Name, boundaryID, wfv1.NodeError, err.Error())
	}
	return woc.initializeNode(nodeName, wfv1.NodeTypePod, tmpl.Name, boundaryID, wfv1.NodeRunning)
}
