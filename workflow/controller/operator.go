package controller

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/argoproj/pkg/humanize"
	argokubeerr "github.com/argoproj/pkg/kube/errors"
	"github.com/argoproj/pkg/strftime"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
	apiv1 "k8s.io/api/core/v1"
	policyv1beta "k8s.io/api/policy/v1beta1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
	envutil "github.com/argoproj/argo/util/env"
	errorsutil "github.com/argoproj/argo/util/errors"
	"github.com/argoproj/argo/util/intstr"
	"github.com/argoproj/argo/util/resource"
	"github.com/argoproj/argo/util/retry"
	"github.com/argoproj/argo/workflow/common"
	controllercache "github.com/argoproj/argo/workflow/controller/cache"
	"github.com/argoproj/argo/workflow/controller/estimation"
	"github.com/argoproj/argo/workflow/controller/indexes"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/argoproj/argo/workflow/progress"
	argosync "github.com/argoproj/argo/workflow/sync"
	"github.com/argoproj/argo/workflow/templateresolution"
	wfutil "github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

// wfOperationCtx is the context for evaluation and operation of a single workflow
type wfOperationCtx struct {
	// wf is the workflow object. It should not be used in execution logic. woc.wfSpec should be used instead
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
	// estimate duration
	estimator estimation.Estimator
	// globalParams holds any parameters that are available to be referenced
	// in the global scope (e.g. workflow.parameters.XXX).
	globalParams common.Parameters
	// volumes holds a DeepCopy of wf.Spec.Volumes to perform substitutions.
	// It is then used in addVolumeReferences() when creating a pod.
	volumes []apiv1.Volume
	// ArtifactRepository contains the default location of an artifact repository for container artifacts
	artifactRepository *config.ArtifactRepository
	// map of pods which need to be labeled with completed=true
	completedPods map[string]bool
	// map of pods which is identified as succeeded=true
	succeededPods map[string]bool
	// deadline is the dealine time in which this operation should relinquish
	// its hold on the workflow so that an operation does not run for too long
	// and starve other workqueue items. It also enables workflow progress to
	// be periodically synced to the database.
	deadline time.Time
	// activePods tracks the number of active (Running/Pending) pods for controlling
	// parallelism
	activePods int64
	// workflowDeadline is the deadline which the workflow is expected to complete before we
	// terminate the workflow.
	workflowDeadline *time.Time
	eventRecorder    record.EventRecorder
	// preExecutionNodePhases contains the phases of all the nodes before the current operation. Necessary to infer
	// changes in phase for metric emission
	preExecutionNodePhases map[string]wfv1.NodePhase

	// execWf holds the Workflow for use in execution.
	// In Normal workflow scenario: It holds copy of workflow object
	// In Submit From WorkflowTemplate: It holds merged workflow with WorkflowDefault, Workflow and WorkflowTemplate
	// 'execWf.Spec' should usually be used instead `wf.Spec`, with two exceptions for user editable fields:
	// 1. `wf.Spec.Suspend`
	// 2. `wf.Spec.Shutdown`
	execWf *wfv1.Workflow
}

var (
	// ErrDeadlineExceeded indicates the operation exceeded its deadline for execution
	ErrDeadlineExceeded = errors.New(errors.CodeTimeout, "Deadline exceeded")
	// ErrParallelismReached indicates this workflow reached its parallelism limit
	ErrParallelismReached = errors.New(errors.CodeForbidden, "Max parallelism reached")
	// ErrTimeout indicates a specific template timed out
	ErrTimeout = errors.New(errors.CodeTimeout, "timeout")
)

// maxOperationTime is the maximum time a workflow operation is allowed to run
// for before requeuing the workflow onto the workqueue.
var (
	maxOperationTime   = envutil.LookupEnvDurationOr("MAX_OPERATION_TIME", 30*time.Second)
	defaultRequeueTime = envutil.LookupEnvDurationOr("DEFAULT_REQUEUE_TIME", maxOperationTime/2)
)

// failedNodeStatus is a subset of NodeStatus that is only used to Marshal certain fields into a JSON of failed nodes
type failedNodeStatus struct {
	DisplayName  string      `json:"displayName"`
	Message      string      `json:"message"`
	TemplateName string      `json:"templateName"`
	Phase        string      `json:"phase"`
	PodName      string      `json:"podName"`
	FinishedAt   metav1.Time `json:"finishedAt"`
}

// newWorkflowOperationCtx creates and initializes a new wfOperationCtx object.
func newWorkflowOperationCtx(wf *wfv1.Workflow, wfc *WorkflowController) *wfOperationCtx {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	wfCopy := wf.DeepCopyObject().(*wfv1.Workflow)
	woc := wfOperationCtx{
		wf:      wfCopy,
		orig:    wf,
		execWf:  wfCopy,
		updated: false,
		log: log.WithFields(log.Fields{
			"workflow":  wf.ObjectMeta.Name,
			"namespace": wf.ObjectMeta.Namespace,
		}),
		controller:             wfc,
		globalParams:           make(map[string]string),
		volumes:                wf.Spec.DeepCopy().Volumes,
		completedPods:          make(map[string]bool),
		succeededPods:          make(map[string]bool),
		deadline:               time.Now().UTC().Add(maxOperationTime),
		eventRecorder:          wfc.eventRecorderManager.Get(wf.Namespace),
		preExecutionNodePhases: make(map[string]wfv1.NodePhase),
	}

	if woc.wf.Status.Nodes == nil {
		woc.wf.Status.Nodes = make(map[string]wfv1.NodeStatus)
	}

	if woc.wf.Status.StoredTemplates == nil {
		woc.wf.Status.StoredTemplates = make(map[string]wfv1.Template)
	}
	return &woc
}

// operate is the main operator logic of a workflow. It evaluates the current state of the workflow,
// and its pods and decides how to proceed down the execution path.
// TODO: an error returned by this method should result in requeuing the workflow to be retried at a
// later time
// As you must not call `persistUpdates` twice, you must not call `operate` twice.
func (woc *wfOperationCtx) operate() {
	defer func() {
		if woc.wf.Status.Fulfilled() {
			_ = woc.killDaemonedChildren("")
		}
		woc.persistUpdates()
	}()
	defer func() {
		if r := recover(); r != nil {
			woc.log.WithFields(log.Fields{"stack": string(debug.Stack()), "r": r}).Errorf("Recovered from panic")
			if rerr, ok := r.(error); ok {
				woc.markWorkflowError(rerr)
			} else {
				woc.markWorkflowPhase(wfv1.NodeError, fmt.Sprintf("%v", r))
			}
			woc.controller.metrics.OperationPanic()
		}
	}()

	woc.log.Infof("Processing workflow")

	// Set the Execute workflow spec for execution
	// ExecWF is a runtime execution spec which merged from Wf, WFT and Wfdefault
	err := woc.setExecWorkflow()
	if err != nil {
		woc.log.WithError(err).Errorf("Unable to get Workflow Template Reference for workflow")
		woc.markWorkflowError(err)
		return
	}

	if woc.wf.Status.ArtifactRepositoryRef == nil {
		ref, err := woc.controller.artifactRepositories.Resolve(woc.execWf.Spec.ArtifactRepositoryRef, woc.wf.Namespace)
		if err != nil {
			woc.markWorkflowError(fmt.Errorf("failed to resolve artifact repository: %w", err))
			return
		}
		woc.wf.Status.ArtifactRepositoryRef = ref
		woc.updated = true
	}

	repo, err := woc.controller.artifactRepositories.Get(woc.wf.Status.ArtifactRepositoryRef)
	if err != nil {
		woc.markWorkflowError(fmt.Errorf("failed to get artifact repository: %v", err))
		return
	}
	woc.artifactRepository = repo

	// Workflow Level Synchronization lock
	if woc.execWf.Spec.Synchronization != nil {
		acquired, wfUpdate, msg, err := woc.controller.syncManager.TryAcquire(woc.wf, "", woc.execWf.Spec.Synchronization)
		if err != nil {
			woc.log.Warn("Failed to acquire the lock")
			woc.markWorkflowFailed(fmt.Sprintf("Failed to acquire the synchronization lock. %s", err.Error()))
			return
		}
		woc.updated = wfUpdate
		if !acquired {
			woc.log.Warn("Workflow processing has been postponed due to concurrency limit")
			woc.wf.Status.Message = msg
			return
		}
	}

	// Update workflow duration variable
	woc.globalParams[common.GlobalVarWorkflowDuration] = fmt.Sprintf("%f", time.Since(woc.wf.Status.StartedAt.Time).Seconds())

	// Populate the phase of all the nodes prior to execution
	for _, node := range woc.wf.Status.Nodes {
		woc.preExecutionNodePhases[node.ID] = node.Phase
	}

	woc.setGlobalParameters(woc.execWf.Spec.Arguments)

	// Perform one-time workflow validation
	if woc.wf.Status.Phase == "" {
		woc.markWorkflowRunning()
		err := woc.createPDBResource()
		if err != nil {
			msg := fmt.Sprintf("Unable to create PDB resource for workflow, %s error: %s", woc.wf.Name, err)
			woc.markWorkflowFailed(msg)
			return
		}
		validateOpts := validate.ValidateOpts{ContainerRuntimeExecutor: woc.controller.GetContainerRuntimeExecutor()}
		wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(woc.wf.Namespace))
		cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(woc.controller.wfclientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())

		// Validate the execution wfSpec
		wfConditions, err := validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, woc.wf, validateOpts)

		if err != nil {
			msg := fmt.Sprintf("invalid spec: %s", err.Error())
			woc.markWorkflowFailed(msg)
			return
		}
		// If we received conditions during validation (such as SpecWarnings), add them to the Workflow object
		if len(*wfConditions) > 0 {
			woc.wf.Status.Conditions.JoinConditions(wfConditions)
			woc.updated = true
		}

		woc.workflowDeadline = woc.getWorkflowDeadline()

		// Workflow will not be requeued if workflow steps are in pending state.
		// Workflow needs to requeue on its deadline,
		if woc.workflowDeadline != nil {
			woc.requeue(time.Until(*woc.workflowDeadline))
		}

		if woc.execWf.Spec.Metrics != nil {
			realTimeScope := map[string]func() float64{common.GlobalVarWorkflowDuration: func() float64 {
				return time.Since(woc.wf.Status.StartedAt.Time).Seconds()
			}}
			woc.computeMetrics(woc.execWf.Spec.Metrics.Prometheus, woc.globalParams, realTimeScope, true)
		}
		woc.wf.Status.EstimatedDuration = woc.estimateWorkflowDuration()
	} else {
		woc.workflowDeadline = woc.getWorkflowDeadline()
		err := woc.podReconciliation()
		if err == nil {
			err = woc.failSuspendedAndPendingNodesAfterDeadlineOrShutdown()
		}
		if err != nil {
			woc.log.WithError(err).WithField("workflow", woc.wf.ObjectMeta.Name).Error("workflow timeout")
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowTimedOut", "Workflow timed out")
			// TODO: we need to re-add to the workqueue, but should happen in caller
			return
		}
	}

	if woc.wf.Spec.Suspend != nil && *woc.wf.Spec.Suspend {
		woc.log.Infof("workflow suspended")
		return
	}
	if woc.execWf.Spec.Parallelism != nil {
		woc.activePods = woc.countActivePods()
	}

	// Create a starting template context.
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	if err != nil {
		woc.log.WithError(err).Error("Failed to create a template context")
		woc.markWorkflowError(err)
		return
	}

	err = woc.substituteParamsInVolumes(woc.globalParams)
	if err != nil {
		woc.log.WithError(err).Error("volumes global param substitution error")
		woc.markWorkflowError(err)
		return
	}

	err = woc.createPVCs()
	if err != nil {
		if errorsutil.IsTransientErr(err) {
			// Error was most likely caused by a lack of resources.
			// In this case, Workflow will be in pending state and requeue.
			woc.markWorkflowPhase(wfv1.NodePending, fmt.Sprintf("Waiting for a PVC to be created. %v", err))
			woc.requeue(defaultRequeueTime)
			return
		}
		err = fmt.Errorf("pvc create error: %w", err)
		woc.log.WithError(err).Error("pvc create error")
		woc.markWorkflowError(err)
		return
	} else if woc.wf.Status.Phase == wfv1.NodePending {
		// Workflow might be in pending state if previous PVC creation is forbidden
		woc.markWorkflowRunning()
	}

	node, err := woc.executeTemplate(woc.wf.ObjectMeta.Name, &wfv1.WorkflowStep{Template: woc.execWf.Spec.Entrypoint}, tmplCtx, woc.execWf.Spec.Arguments, &executeTemplateOpts{})
	if err != nil {
		// the error are handled in the callee so just log it.
		msg := "error in entry template execution"
		woc.log.WithError(err).Error(msg)
		msg = fmt.Sprintf("%s %s: %+v", woc.wf.Name, msg, err)
		switch err {
		case ErrDeadlineExceeded:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowTimedOut", msg)
		}
		return
	}

	if node == nil || !node.Fulfilled() {
		// node can be nil if a workflow created immediately in a parallelism == 0 state
		return
	}

	workflowStatus := node.Phase
	var onExitNode *wfv1.NodeStatus
	if woc.execWf.Spec.OnExit != "" && woc.wf.Spec.Shutdown.ShouldExecute(true) {
		if workflowStatus == wfv1.NodeSkipped {
			// treat skipped the same as Succeeded for workflow.status
			woc.globalParams[common.GlobalVarWorkflowStatus] = string(wfv1.NodeSucceeded)
		} else {
			woc.globalParams[common.GlobalVarWorkflowStatus] = string(workflowStatus)
		}

		var failures []failedNodeStatus
		for _, node := range woc.wf.Status.Nodes {
			if node.Phase == wfv1.NodeFailed || node.Phase == wfv1.NodeError {
				failures = append(failures,
					failedNodeStatus{
						DisplayName:  node.DisplayName,
						Message:      node.Message,
						TemplateName: node.TemplateName,
						Phase:        string(node.Phase),
						PodName:      node.ID,
						FinishedAt:   node.FinishedAt,
					})
			}
		}
		failedNodeBytes, err := json.Marshal(failures)
		if err != nil {
			woc.log.Errorf("Error marshalling failed nodes list: %+v", err)
			// No need to return here
		}
		// This strconv.Quote is necessary so that the escaped quotes are not removed during parameter substitution
		woc.globalParams[common.GlobalVarWorkflowFailures] = strconv.Quote(string(failedNodeBytes))

		woc.log.Infof("Running OnExit handler: %s", woc.execWf.Spec.OnExit)
		onExitNodeName := common.GenerateOnExitNodeName(woc.wf.ObjectMeta.Name)
		onExitNode, err = woc.executeTemplate(onExitNodeName, &wfv1.WorkflowStep{Template: woc.execWf.Spec.OnExit}, tmplCtx, woc.execWf.Spec.Arguments, &executeTemplateOpts{onExitTemplate: true})
		if err != nil {
			// the error are handled in the callee so just log it.
			woc.log.WithError(err).Error("error in exit template execution")
			return
		}
		if onExitNode == nil || !onExitNode.Fulfilled() {
			return
		}
	}

	var workflowMessage string
	if node.FailedOrError() && woc.execWf.Spec.Shutdown != "" {
		workflowMessage = fmt.Sprintf("Stopped with strategy '%s'", woc.execWf.Spec.Shutdown)
	} else {
		workflowMessage = node.Message
	}

	// If we get here, the workflow completed, all PVCs were deleted successfully, and
	// exit handlers were executed. We now need to infer the workflow phase from the
	// node phase.
	switch workflowStatus {
	case wfv1.NodeSucceeded, wfv1.NodeSkipped:
		if onExitNode != nil && onExitNode.FailedOrError() {
			// if main workflow succeeded, but the exit node was unsuccessful
			// the workflow is now considered unsuccessful.
			woc.markWorkflowPhase(onExitNode.Phase, onExitNode.Message)
		} else {
			woc.markWorkflowSuccess()
		}
	case wfv1.NodeFailed:
		woc.markWorkflowFailed(workflowMessage)
	case wfv1.NodeError:
		woc.markWorkflowPhase(wfv1.NodeError, workflowMessage)
	default:
		// NOTE: we should never make it here because if the node was 'Running' we should have
		// returned earlier.
		err = errors.InternalErrorf("Unexpected node phase %s: %+v", woc.wf.ObjectMeta.Name, err)
		woc.markWorkflowError(err)
	}

	if woc.execWf.Spec.Metrics != nil {
		realTimeScope := map[string]func() float64{common.GlobalVarWorkflowDuration: func() float64 {
			return node.FinishedAt.Sub(node.StartedAt.Time).Seconds()
		}}
		woc.globalParams[common.GlobalVarWorkflowStatus] = string(workflowStatus)
		woc.computeMetrics(woc.execWf.Spec.Metrics.Prometheus, woc.globalParams, realTimeScope, false)
	}

	err = woc.deletePVCs()
	if err != nil {
		woc.log.WithError(err).Warn("failed to delete PVCs")
	}
}

func (woc *wfOperationCtx) getWorkflowDeadline() *time.Time {
	if woc.execWf.Spec.ActiveDeadlineSeconds == nil {
		return nil
	}
	if woc.wf.Status.StartedAt.IsZero() {
		return nil
	}
	startedAt := woc.wf.Status.StartedAt.Truncate(time.Second)
	deadline := startedAt.Add(time.Duration(*woc.execWf.Spec.ActiveDeadlineSeconds) * time.Second).UTC()
	return &deadline
}

// setGlobalParameters sets the globalParam map with global parameters
func (woc *wfOperationCtx) setGlobalParameters(executionParameters wfv1.Arguments) {
	woc.globalParams[common.GlobalVarWorkflowName] = woc.wf.ObjectMeta.Name
	woc.globalParams[common.GlobalVarWorkflowNamespace] = woc.wf.ObjectMeta.Namespace
	woc.globalParams[common.GlobalVarWorkflowServiceAccountName] = woc.execWf.Spec.ServiceAccountName
	woc.globalParams[common.GlobalVarWorkflowUID] = string(woc.wf.ObjectMeta.UID)
	woc.globalParams[common.GlobalVarWorkflowCreationTimestamp] = woc.wf.ObjectMeta.CreationTimestamp.Format(time.RFC3339)
	if woc.execWf.Spec.Priority != nil {
		woc.globalParams[common.GlobalVarWorkflowPriority] = strconv.Itoa(int(*woc.execWf.Spec.Priority))
	}
	for char := range strftime.FormatChars {
		cTimeVar := fmt.Sprintf("%s.%s", common.GlobalVarWorkflowCreationTimestamp, string(char))
		woc.globalParams[cTimeVar] = strftime.Format("%"+string(char), woc.wf.ObjectMeta.CreationTimestamp.Time)
	}

	if workflowParameters, err := json.Marshal(woc.execWf.Spec.Arguments.Parameters); err == nil {
		woc.globalParams[common.GlobalVarWorkflowParameters] = string(workflowParameters)
	}
	for _, param := range executionParameters.Parameters {
		woc.globalParams["workflow.parameters."+param.Name] = param.Value.String()
	}
	for k, v := range woc.wf.ObjectMeta.Annotations {
		woc.globalParams["workflow.annotations."+k] = v
	}
	for k, v := range woc.wf.ObjectMeta.Labels {
		woc.globalParams["workflow.labels."+k] = v
	}
	if woc.wf.Status.Outputs != nil {
		for _, param := range woc.wf.Status.Outputs.Parameters {
			woc.globalParams["workflow.outputs.parameters."+param.Name] = param.Value.String()
		}
	}
}

// persistUpdates will update a workflow with any updates made during workflow operation.
// It also labels any pods as completed if we have extracted everything we need from it.
// NOTE: a previous implementation used Patch instead of Update, but Patch does not work with
// the fake CRD clientset which makes unit testing extremely difficult.
func (woc *wfOperationCtx) persistUpdates() {
	if !woc.updated {
		return
	}
	resource.UpdateResourceDurations(woc.wf)
	progress.UpdateProgress(woc.wf)
	// You MUST not call `persistUpdates` twice.
	// * Fails the `reapplyUpdate` cannot work unless resource versions are different.
	// * It will double the number of Kubernetes API requests.
	if woc.orig.ResourceVersion != woc.wf.ResourceVersion {
		woc.log.Panic("cannot persist updates with mismatched resource versions")
	}
	wfClient := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows(woc.wf.ObjectMeta.Namespace)
	// try and compress nodes if needed
	nodes := woc.wf.Status.Nodes
	err := woc.controller.hydrator.Dehydrate(woc.wf)
	if err != nil {
		woc.log.Warnf("Failed to dehydrate: %v", err)
		woc.markWorkflowError(err)
	}

	// Release all acquired lock for completed workflow
	if woc.wf.Status.Synchronization != nil && woc.wf.Status.Fulfilled() {
		if woc.controller.syncManager.ReleaseAll(woc.wf) {
			log.WithFields(log.Fields{"key": woc.wf.Name}).Info("Released all acquired locks")
		}
	}

	wf, err := wfClient.Update(woc.wf)
	if err != nil {
		woc.log.Warnf("Error updating workflow: %v %s", err, apierr.ReasonForError(err))
		if argokubeerr.IsRequestEntityTooLargeErr(err) {
			woc.persistWorkflowSizeLimitErr(wfClient, err)
			return
		}
		if !apierr.IsConflict(err) {
			return
		}
		woc.log.Info("Re-applying updates on latest version and retrying update")
		wf, err := woc.reapplyUpdate(wfClient, nodes)
		if err != nil {
			woc.log.Infof("Failed to re-apply update: %+v", err)
			return
		}
		woc.wf = wf
	} else {
		woc.wf = wf
		woc.controller.hydrator.HydrateWithNodes(woc.wf, nodes)
	}

	if !woc.controller.hydrator.IsHydrated(woc.wf) {
		panic("workflow should be hydrated")
	}

	woc.log.WithFields(log.Fields{"resourceVersion": woc.wf.ResourceVersion, "phase": woc.wf.Status.Phase}).Info("Workflow update successful")

	if os.Getenv("INFORMER_WRITE_BACK") != "false" {
		if err := woc.writeBackToInformer(); err != nil {
			woc.markWorkflowError(err)
			return
		}
	} else {
		time.Sleep(enoughTimeForInformerSync)
	}

	// It is important that we *never* label pods as completed until we successfully updated the workflow
	// Failing to do so means we can have inconsistent state.
	// TODO: The completedPods will be labeled multiple times. I think it would be improved in the future.
	// Send succeeded pods or completed pods to gcPods channel to delete it later depend on the PodGCStrategy.
	// Notice we do not need to label the pod if we will delete it later for GC. Otherwise, that may even result in
	// errors if we label a pod that was deleted already.
	if woc.execWf.Spec.PodGC != nil {
		switch woc.execWf.Spec.PodGC.Strategy {
		case wfv1.PodGCOnPodSuccess:
			for podName := range woc.succeededPods {
				woc.controller.gcPods <- fmt.Sprintf("%s/%s", woc.wf.ObjectMeta.Namespace, podName)
			}
		case wfv1.PodGCOnPodCompletion:
			for podName := range woc.completedPods {
				woc.controller.gcPods <- fmt.Sprintf("%s/%s", woc.wf.ObjectMeta.Namespace, podName)
			}
		}
	} else {
		// label pods which will not be deleted
		for podName := range woc.completedPods {
			woc.controller.completedPods <- fmt.Sprintf("%s/%s", woc.wf.ObjectMeta.Namespace, podName)
		}
	}
}

func (woc *wfOperationCtx) writeBackToInformer() error {
	un, err := wfutil.ToUnstructured(woc.wf)
	if err != nil {
		return fmt.Errorf("failed to convert workflow to unstructured: %w", err)
	}
	err = woc.controller.wfInformer.GetStore().Update(un)
	if err != nil {
		return fmt.Errorf("failed to update informer store: %w", err)
	}
	return nil
}

// persistWorkflowSizeLimitErr will fail a the workflow with an error when we hit the resource size limit
// See https://github.com/argoproj/argo/issues/913
func (woc *wfOperationCtx) persistWorkflowSizeLimitErr(wfClient v1alpha1.WorkflowInterface, err error) {
	woc.wf = woc.orig.DeepCopy()
	woc.markWorkflowError(err)
	_, err = wfClient.Update(woc.wf)
	if err != nil {
		woc.log.Warnf("Error updating workflow with size error: %v", err)
	}
}

// reapplyUpdate GETs the latest version of the workflow, re-applies the updates and
// retries the UPDATE multiple times. For reasoning behind this technique, see:
// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
func (woc *wfOperationCtx) reapplyUpdate(wfClient v1alpha1.WorkflowInterface, nodes wfv1.Nodes) (*wfv1.Workflow, error) {
	// if this condition is true, then this func will always error
	if woc.orig.ResourceVersion != woc.wf.ResourceVersion {
		woc.log.Panic("cannot re-apply update with mismatched resource versions")
	}
	err := woc.controller.hydrator.Hydrate(woc.orig)
	if err != nil {
		return nil, err
	}
	// First generate the patch
	oldData, err := json.Marshal(woc.orig)
	if err != nil {
		return nil, err
	}
	woc.controller.hydrator.HydrateWithNodes(woc.wf, nodes)
	newData, err := json.Marshal(woc.wf)
	if err != nil {
		return nil, err
	}
	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return nil, err
	}
	// Next get latest version of the workflow, apply the patch and retry the update
	attempt := 1
	for {
		currWf, err := wfClient.Get(woc.wf.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		err = woc.controller.hydrator.Hydrate(currWf)
		if err != nil {
			return nil, err
		}
		currWfBytes, err := json.Marshal(currWf)
		if err != nil {
			return nil, err
		}
		newWfBytes, err := jsonpatch.MergePatch(currWfBytes, patchBytes)
		if err != nil {
			return nil, err
		}
		var newWf wfv1.Workflow
		err = json.Unmarshal(newWfBytes, &newWf)
		if err != nil {
			return nil, err
		}
		err = woc.controller.hydrator.Dehydrate(&newWf)
		if err != nil {
			return nil, err
		}
		wf, err := wfClient.Update(&newWf)
		if err == nil {
			woc.log.Infof("Update retry attempt %d successful", attempt)
			woc.controller.hydrator.HydrateWithNodes(wf, nodes)
			return wf, nil
		}
		attempt++
		woc.log.Warnf("Update retry attempt %d failed: %v", attempt, err)
		if attempt > 5 {
			return nil, err
		}
	}
}

// requeue this workflow onto the workqueue for later processing
func (woc *wfOperationCtx) requeue(afterDuration time.Duration) {
	key, err := cache.MetaNamespaceKeyFunc(woc.wf)
	if err != nil {
		woc.log.Errorf("Failed to requeue workflow %s: %v", woc.wf.ObjectMeta.Name, err)
		return
	}
	woc.controller.wfQueue.AddAfter(key, afterDuration)
}

// processNodeRetries updates the retry node state based on the child node state and the retry strategy and returns the node.
func (woc *wfOperationCtx) processNodeRetries(node *wfv1.NodeStatus, retryStrategy wfv1.RetryStrategy, opts *executeTemplateOpts) (*wfv1.NodeStatus, bool, error) {
	if node.Fulfilled() {
		return node, true, nil
	}
	lastChildNode := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)

	if lastChildNode == nil {
		return node, true, nil
	}

	if !lastChildNode.Fulfilled() {
		// last child node is still running.
		return node, true, nil
	}

	if !lastChildNode.FailedOrError() {
		node.Outputs = lastChildNode.Outputs.DeepCopy()
		woc.wf.Status.Nodes[node.ID] = *node
		return woc.markNodePhase(node.Name, wfv1.NodeSucceeded), true, nil
	}

	if woc.execWf.Spec.Shutdown != "" || (woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline)) {
		var message string
		if woc.execWf.Spec.Shutdown != "" {
			message = fmt.Sprintf("Stopped with strategy '%s'", woc.execWf.Spec.Shutdown)
		} else {
			message = fmt.Sprintf("retry exceeded workflow deadline %s", *woc.workflowDeadline)
		}
		woc.log.Infoln(message)
		return woc.markNodePhase(node.Name, lastChildNode.Phase, message), true, nil
	}

	if retryStrategy.Backoff != nil {
		maxDurationDeadline := time.Time{}
		// Process max duration limit
		if retryStrategy.Backoff.MaxDuration != "" && len(node.Children) > 0 {
			maxDuration, err := parseStringToDuration(retryStrategy.Backoff.MaxDuration)
			if err != nil {
				return nil, false, err
			}
			firstChildNode := getChildNodeIndex(node, woc.wf.Status.Nodes, 0)
			maxDurationDeadline = firstChildNode.StartedAt.Add(maxDuration)
			if time.Now().After(maxDurationDeadline) {
				woc.log.Infoln("Max duration limit exceeded. Failing...")
				return woc.markNodePhase(node.Name, lastChildNode.Phase, "Max duration limit exceeded"), true, nil
			}
		}

		// Max duration limit hasn't been exceeded, process back off
		if retryStrategy.Backoff.Duration == "" {
			return nil, false, fmt.Errorf("no base duration specified for retryStrategy")
		}

		baseDuration, err := parseStringToDuration(retryStrategy.Backoff.Duration)
		if err != nil {
			return nil, false, err
		}

		timeToWait := baseDuration
		retryStrategyBackoffFactor, err := intstr.Int32(retryStrategy.Backoff.Factor)
		if err != nil {
			return nil, false, err
		}
		if retryStrategyBackoffFactor != nil && *retryStrategyBackoffFactor > 0 {
			// Formula: timeToWait = duration * factor^retry_number
			// Note that timeToWait should equal to duration for the first retry attempt.
			timeToWait = baseDuration * time.Duration(math.Pow(float64(*retryStrategyBackoffFactor), float64(len(node.Children)-1)))
		}
		waitingDeadline := lastChildNode.FinishedAt.Add(timeToWait)

		// If the waiting deadline is after the max duration deadline, then it's futile to wait until then. Stop early
		if !maxDurationDeadline.IsZero() && waitingDeadline.After(maxDurationDeadline) {
			woc.log.Infoln("Backoff would exceed max duration limit. Failing...")
			return woc.markNodePhase(node.Name, lastChildNode.Phase, "Backoff would exceed max duration limit"), true, nil
		}

		// See if we have waited past the deadline
		if time.Now().Before(waitingDeadline) {
			woc.requeue(timeToWait)
			retryMessage := fmt.Sprintf("Backoff for %s", humanize.Duration(timeToWait))
			return woc.markNodePhase(node.Name, node.Phase, retryMessage), false, nil
		}

		woc.log.WithField("node", node.Name).Infof("node has maxDuration set, setting executionDeadline to: %s", humanize.Timestamp(maxDurationDeadline))
		opts.executionDeadline = maxDurationDeadline

		node = woc.markNodePhase(node.Name, node.Phase, "")
	}

	var retryOnFailed bool
	var retryOnError bool
	switch retryStrategy.RetryPolicy {
	case wfv1.RetryPolicyAlways:
		retryOnFailed = true
		retryOnError = true
	case wfv1.RetryPolicyOnError:
		retryOnFailed = false
		retryOnError = true
	case wfv1.RetryPolicyOnFailure, "":
		retryOnFailed = true
		retryOnError = false
	default:
		return nil, false, fmt.Errorf("%s is not a valid RetryPolicy", retryStrategy.RetryPolicy)
	}

	if (lastChildNode.Phase == wfv1.NodeFailed && !retryOnFailed) || (lastChildNode.Phase == wfv1.NodeError && !retryOnError) {
		woc.log.Infof("Node not set to be retried after status: %s", lastChildNode.Phase)
		return woc.markNodePhase(node.Name, lastChildNode.Phase, lastChildNode.Message), true, nil
	}

	if !lastChildNode.CanRetry() {
		woc.log.Infof("Node cannot be retried. Marking it failed")
		return woc.markNodePhase(node.Name, lastChildNode.Phase, lastChildNode.Message), true, nil
	}

	limit, err := intstr.Int32(retryStrategy.Limit)
	if err != nil {
		return nil, false, err
	}
	if retryStrategy.Limit != nil && limit != nil && int32(len(node.Children)) > *limit {
		woc.log.Infoln("No more retries left. Failing...")
		return woc.markNodePhase(node.Name, lastChildNode.Phase, "No more retries left"), true, nil
	}

	woc.log.Infof("%d child nodes of %s failed. Trying again...", len(node.Children), node.Name)
	return node, true, nil
}

// podReconciliation is the process by which a workflow will examine all its related
// pods and update the node state before continuing the evaluation of the workflow.
// Records all pods which were observed completed, which will be labeled completed=true
// after successful persist of the workflow.
func (woc *wfOperationCtx) podReconciliation() error {
	podList, err := woc.getAllWorkflowPods()
	if err != nil {
		return err
	}
	seenPods := make(map[string]*apiv1.Pod)
	seenPodLock := &sync.Mutex{}
	wfNodesLock := &sync.RWMutex{}

	performAssessment := func(pod *apiv1.Pod) {
		if pod == nil {
			return
		}
		nodeNameForPod := pod.Annotations[common.AnnotationKeyNodeName]
		nodeID := woc.wf.NodeID(nodeNameForPod)
		seenPodLock.Lock()
		seenPods[nodeID] = pod
		seenPodLock.Unlock()

		wfNodesLock.Lock()
		defer wfNodesLock.Unlock()
		if node, ok := woc.wf.Status.Nodes[nodeID]; ok {
			if newState := woc.assessNodeStatus(pod, &node); newState != nil {
				woc.wf.Status.Nodes[nodeID] = *newState
				woc.addOutputsToGlobalScope(node.Outputs)
				if node.MemoizationStatus != nil {
					c := woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, node.MemoizationStatus.CacheName)
					err := c.Save(node.MemoizationStatus.Key, node.ID, node.Outputs)
					if err != nil {
						woc.log.WithFields(log.Fields{"nodeID": node.ID}).WithError(err).Error("Failed to save node outputs to cache")
						node.Phase = wfv1.NodeError
					}
				}
				woc.updated = true
			}
			node := woc.wf.Status.Nodes[pod.ObjectMeta.Name]
			if node.Fulfilled() && !node.IsDaemoned() {
				if tmpVal, tmpOk := pod.Labels[common.LabelKeyCompleted]; tmpOk {
					if tmpVal == "true" {
						return
					}
				}
				woc.completedPods[pod.ObjectMeta.Name] = true
				if woc.shouldPrintPodSpec(node) {
					printPodSpecLog(pod, woc.wf.Name)
				}
				if !woc.orig.Status.Nodes[node.ID].Fulfilled() {
					woc.onNodeComplete(&node)
				}
			}
			if node.Succeeded() {
				woc.succeededPods[pod.ObjectMeta.Name] = true
			}
		}
	}

	parallelPodNum := make(chan string, 500)
	var wg sync.WaitGroup

	for _, pod := range podList {
		parallelPodNum <- pod.Name
		wg.Add(1)
		go func(pod *apiv1.Pod) {
			defer wg.Done()
			performAssessment(pod)
			err = woc.applyExecutionControl(pod, wfNodesLock)
			if err != nil {
				woc.log.Warnf("Failed to apply execution control to pod %s", pod.Name)
			}
			<-parallelPodNum
		}(pod)
	}

	wg.Wait()

	// Now check for deleted pods. Iterate our nodes. If any one of our nodes does not show up in
	// the seen list it implies that the pod was deleted without the controller seeing the event.
	// It is now impossible to infer pod status. We can do at this point is to mark the node with Error, or
	// we can re-submit it.
	for nodeID, node := range woc.wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod || node.Fulfilled() || node.StartedAt.IsZero() {
			// node is not a pod, it is already complete, or it can be re-run.
			continue
		}
		if _, ok := seenPods[nodeID]; !ok {

			// If the node is pending and the pod does not exist, it could be the case that we want to try to submit it
			// again instead of marking it as an error. Check if that's the case.
			if node.Pending() {
				continue
			}

			node.Message = "pod deleted"
			node.Phase = wfv1.NodeError
			// FinishedAt must be set since retry strategy depends on it to determine the backoff duration.
			// See processNodeRetries for more details.
			node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
			woc.wf.Status.Nodes[nodeID] = node
			woc.log.WithField("displayName", node.DisplayName).WithField("templateName", node.TemplateName).
				WithField("node", node.Name).Error("Pod for node deleted")
			woc.updated = true
		} else {
			// At this point we are certain that the pod associated with our node is running or has been run;
			// it is safe to extract the k8s-node information given this knowledge.
			if node.HostNodeName != seenPods[nodeID].Spec.NodeName {
				node.HostNodeName = seenPods[nodeID].Spec.NodeName
				woc.wf.Status.Nodes[nodeID] = node
				woc.updated = true
			}
		}
	}
	return nil
}

// shouldPrintPodSpec return eligible to print to the pod spec
func (woc *wfOperationCtx) shouldPrintPodSpec(node wfv1.NodeStatus) bool {
	return woc.controller.Config.PodSpecLogStrategy.AllPods ||
		(woc.controller.Config.PodSpecLogStrategy.FailedPod && node.FailedOrError())
}

//fails any suspended and pending nodes if the workflow deadline has passed
func (woc *wfOperationCtx) failSuspendedAndPendingNodesAfterDeadlineOrShutdown() error {
	deadlineExceeded := woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline)
	if woc.execWf.Spec.Shutdown != "" || deadlineExceeded {
		for _, node := range woc.wf.Status.Nodes {
			if node.IsActiveSuspendNode() || (node.Phase == wfv1.NodePending && deadlineExceeded) {
				var message string
				if woc.execWf.Spec.Shutdown != "" {
					message = fmt.Sprintf("Stopped with strategy '%s'", woc.execWf.Spec.Shutdown)
				} else {
					message = "Step exceeded its deadline"
				}
				woc.markNodePhase(node.Name, wfv1.NodeFailed, message)
			}
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
		if node.Type != wfv1.NodeTypePod {
			continue
		}
		if boundaryID != "" && node.BoundaryID != boundaryID {
			continue
		}
		switch node.Phase {
		case wfv1.NodePending, wfv1.NodeRunning:
			if node.SynchronizationStatus != nil && node.SynchronizationStatus.Waiting != "" {
				// Do not include pending nodes that are waiting for a lock
				continue
			}
			activePods++
		}
	}
	return activePods
}

// countActiveChildren counts the number of active (Pending/Running) children nodes of parent parentName
func (woc *wfOperationCtx) countActiveChildren(boundaryIDs ...string) int64 {
	var boundaryID = ""
	if len(boundaryIDs) > 0 {
		boundaryID = boundaryIDs[0]
	}
	var activeChildren int64
	// if we care about parallelism, count the active pods at the template level
	for _, node := range woc.wf.Status.Nodes {
		if boundaryID != "" && node.BoundaryID != boundaryID {
			continue
		}
		switch node.Type {
		case wfv1.NodeTypePod, wfv1.NodeTypeSteps, wfv1.NodeTypeDAG:
		default:
			continue
		}
		switch node.Phase {
		case wfv1.NodePending, wfv1.NodeRunning:
			activeChildren++
		}
	}
	return activeChildren
}

// getAllWorkflowPods returns all pods related to the current workflow
func (woc *wfOperationCtx) getAllWorkflowPods() ([]*apiv1.Pod, error) {
	objs, err := woc.controller.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, indexes.WorkflowIndexValue(woc.wf.Namespace, woc.wf.Name))
	if err != nil {
		return nil, err
	}
	pods := make([]*apiv1.Pod, len(objs))
	for i, obj := range objs {
		pod, ok := obj.(*apiv1.Pod)
		if !ok {
			return nil, fmt.Errorf("expected \"*apiv1.Pod\", got \"%v\"", reflect.TypeOf(obj).String())
		}
		pods[i] = pod
	}
	return pods, nil
}

func printPodSpecLog(pod *apiv1.Pod, wfName string) {
	podSpecByte, err := json.Marshal(pod)
	if err != nil {
		log.WithField("workflow", wfName).WithField("nodename", pod.Name).WithField("namespace", pod.Namespace).Warnf("Unable to mashal pod spec. %v", err)
	}
	log.WithField("workflow", wfName).WithField("nodename", pod.Name).WithField("namespace", pod.Namespace).Infof("Pod Spec: %s", string(podSpecByte))
}

// assessNodeStatus compares the current state of a pod with its corresponding node
// and returns the new node status if something changed
func (woc *wfOperationCtx) assessNodeStatus(pod *apiv1.Pod, node *wfv1.NodeStatus) *wfv1.NodeStatus {
	var newPhase wfv1.NodePhase
	var newDaemonStatus *bool
	var message string
	updated := false
	switch pod.Status.Phase {
	case apiv1.PodPending:
		newPhase = wfv1.NodePending
		newDaemonStatus = pointer.BoolPtr(false)
		message = getPendingReason(pod)
	case apiv1.PodSucceeded:
		newPhase = wfv1.NodeSucceeded
		newDaemonStatus = pointer.BoolPtr(false)
	case apiv1.PodFailed:
		// ignore pod failure for daemoned steps
		if node.IsDaemoned() {
			newPhase = wfv1.NodeSucceeded
		} else {
			newPhase, message = inferFailedReason(pod)
			woc.log.WithField("displayName", node.DisplayName).WithField("templateName", node.TemplateName).
				WithField("pod", pod.Name).Infof("Pod failed")
		}
		newDaemonStatus = pointer.BoolPtr(false)
	case apiv1.PodRunning:
		if pod.DeletionTimestamp != nil {
			// pod is being terminated
			newPhase = wfv1.NodeError
			message = "pod deleted during operation"
			woc.log.WithField("displayName", node.DisplayName).WithField("templateName", node.TemplateName).
				WithField("pod", pod.Name).Error(message)
		} else {
			newPhase = wfv1.NodeRunning
			tmplStr, ok := pod.Annotations[common.AnnotationKeyTemplate]
			if !ok {
				log.WithField("pod", pod.ObjectMeta.Name).Warn("missing template annotation")
				return nil
			}
			var tmpl wfv1.Template
			err := json.Unmarshal([]byte(tmplStr), &tmpl)
			if err != nil {
				log.WithError(err).WithField("pod", pod.ObjectMeta.Name).Warn("template annotation unreadable")
				return nil
			}
			if tmpl.Daemon != nil && *tmpl.Daemon {
				// pod is running and template is marked daemon. check if everything is ready
				for _, ctrStatus := range pod.Status.ContainerStatuses {
					if !ctrStatus.Ready {
						return nil
					}
				}
				// proceed to mark node status as running (and daemoned)
				newPhase = wfv1.NodeRunning
				newDaemonStatus = pointer.BoolPtr(true)
				log.Infof("Processing ready daemon pod: %v", pod.ObjectMeta.SelfLink)
			}
		}
	default:
		newPhase = wfv1.NodeError
		message = fmt.Sprintf("Unexpected pod phase for %s: %s", pod.ObjectMeta.Name, pod.Status.Phase)
		woc.log.WithField("displayName", node.DisplayName).WithField("templateName", node.TemplateName).
			WithField("pod", pod.Name).Error(message)
	}

	if newDaemonStatus != nil {
		if !*newDaemonStatus {
			// if the daemon status switched to false, we prefer to just unset daemoned status field
			// (as opposed to setting it to false)
			newDaemonStatus = nil
		}
		if (newDaemonStatus != nil && node.Daemoned == nil) || (newDaemonStatus == nil && node.Daemoned != nil) {
			log.Infof("Setting node %v daemoned: %v -> %v", node.ID, node.Daemoned, newDaemonStatus)
			node.Daemoned = newDaemonStatus
			updated = true
			if pod.Status.PodIP != "" && pod.Status.PodIP != node.PodIP {
				// only update Pod IP for daemoned nodes to reduce number of updates
				log.Infof("Updating daemon node %s IP %s -> %s", node.ID, node.PodIP, pod.Status.PodIP)
				node.PodIP = pod.Status.PodIP
			}
		}
	}
	outputStr, ok := pod.Annotations[common.AnnotationKeyOutputs]
	if ok && node.Outputs == nil {
		updated = true
		log.Infof("Setting node %v outputs", node.ID)
		var outputs wfv1.Outputs
		err := json.Unmarshal([]byte(outputStr), &outputs)
		if err != nil {
			woc.log.WithField("displayName", node.DisplayName).WithField("templateName", node.TemplateName).
				WithField("pod", pod.Name).Errorf("Failed to unmarshal %s outputs from pod annotation: %v", pod.Name, err)
			node.Phase = wfv1.NodeError
		} else {
			node.Outputs = &outputs
		}
	}
	if node.Phase != newPhase {
		log.Infof("Updating node %s status %s -> %s", node.ID, node.Phase, newPhase)
		// if we are transitioning from Pending to a different state, clear out pending message
		if node.Phase == wfv1.NodePending {
			node.Message = ""
		}
		updated = true
		node.Phase = newPhase
	}
	if message != "" && node.Message != message {
		log.Infof("Updating node %s message: %s", node.ID, message)
		updated = true
		node.Message = message
	}

	if node.Fulfilled() && node.FinishedAt.IsZero() {
		updated = true
		if !node.IsDaemoned() {
			node.FinishedAt = getLatestFinishedAt(pod)
		}
		if node.FinishedAt.IsZero() {
			// If we get here, the container is daemoned so the
			// finishedAt might not have been set.
			node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
		}
		node.ResourcesDuration = resource.DurationForPod(pod)
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

func getPendingReason(pod *apiv1.Pod) string {
	for _, ctrStatus := range pod.Status.ContainerStatuses {
		if ctrStatus.State.Waiting != nil {
			if ctrStatus.State.Waiting.Message != "" {
				return fmt.Sprintf("%s: %s", ctrStatus.State.Waiting.Reason, ctrStatus.State.Waiting.Message)
			}
			return ctrStatus.State.Waiting.Reason
		}
	}
	// Example:
	// - lastProbeTime: null
	//   lastTransitionTime: 2018-08-29T06:38:36Z
	//   message: '0/3 nodes are available: 2 Insufficient cpu, 3 MatchNodeSelector.'
	//   reason: Unschedulable
	//   status: "False"
	//   type: PodScheduled
	for _, cond := range pod.Status.Conditions {
		if cond.Reason == apiv1.PodReasonUnschedulable {
			if cond.Message != "" {
				return fmt.Sprintf("%s: %s", cond.Reason, cond.Message)
			}
			return cond.Reason
		}
	}
	return ""
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
		// Virtual Kubelet environment will not set the terminate on waiting container
		// https://github.com/argoproj/argo/issues/3879
		// https://github.com/virtual-kubelet/virtual-kubelet/blob/7f2a02291530d2df14905702e6d51500dd57640a/node/sync.go#L195-L208
		if ctr.State.Waiting != nil {
			return wfv1.NodeError, fmt.Sprintf("Pod failed before %s container starts", ctr.Name)
		}
		if ctr.State.Terminated == nil {
			// We should never get here
			log.Warnf("Pod %s phase was Failed but %s did not have terminated state", pod.ObjectMeta.Name, ctr.Name)
			continue
		}
		if ctr.State.Terminated.ExitCode == 0 {
			continue
		}
		errMsg := "failed to load artifacts"
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
		// Virtual Kubelet environment will not set the terminate on waiting container
		// https://github.com/argoproj/argo/issues/3879
		// https://github.com/virtual-kubelet/virtual-kubelet/blob/7f2a02291530d2df14905702e6d51500dd57640a/node/sync.go#L195-L208

		if ctr.State.Waiting != nil {
			return wfv1.NodeError, fmt.Sprintf("Pod failed before %s container starts", ctr.Name)
		}
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
			continue
		}
		if ctr.State.Terminated.Message != "" {
			errMsg := ctr.State.Terminated.Message
			if ctr.Name != common.MainContainerName {
				errMsg = fmt.Sprintf("sidecar '%s' %s", ctr.Name, errMsg)
			}
			failMessages[ctr.Name] = errMsg
			continue
		}
		if ctr.State.Terminated.Reason == "OOMKilled" {
			failMessages[ctr.Name] = ctr.State.Terminated.Reason
			continue
		}
		errMsg := fmt.Sprintf("failed with exit code %d", ctr.State.Terminated.ExitCode)
		if ctr.Name != common.MainContainerName {
			if ctr.State.Terminated.ExitCode == 137 || ctr.State.Terminated.ExitCode == 143 {
				// if the sidecar was SIGKILL'd (exit code 137) assume it was because argoexec
				// forcibly killed the container, which we ignore the error for.
				// Java code 143 is a normal exit 128 + 15 https://github.com/elastic/elasticsearch/issues/31847
				log.Infof("Ignoring %d exit code of sidecar '%s'", ctr.State.Terminated.ExitCode, ctr.Name)
				continue
			}
			errMsg = fmt.Sprintf("sidecar '%s' %s", ctr.Name, errMsg)
		}
		failMessages[ctr.Name] = errMsg
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

	// If we get here, both the main and wait container succeeded. Iterate the fail messages to
	// identify the sidecar which failed and return the message.
	for _, failMsg := range failMessages {
		return wfv1.NodeFailed, failMsg
	}
	// If we get here, we have detected that the main/wait containers succeed but the sidecar(s)
	// were  SIGKILL'd. The executor may have had to forcefully terminate the sidecar (kill -9),
	// resulting in a 137 exit code (which we had ignored earlier). If failMessages is empty, it
	// indicates that this is the case and we return Success instead of Failure.
	return wfv1.NodeSucceeded, ""
}

func (woc *wfOperationCtx) createPVCs() error {
	if !(woc.wf.Status.Phase == wfv1.NodePending || woc.wf.Status.Phase == wfv1.NodeRunning) {
		// Only attempt to create PVCs if workflow is in Pending or Running state
		// (e.g. passed validation, or didn't already complete)
		return nil
	}
	if len(woc.execWf.Spec.VolumeClaimTemplates) == len(woc.wf.Status.PersistentVolumeClaims) {
		// If we have already created the PVCs, then there is nothing to do.
		// This will also handle the case where workflow has no volumeClaimTemplates.
		return nil
	}
	pvcClient := woc.controller.kubeclientset.CoreV1().PersistentVolumeClaims(woc.wf.ObjectMeta.Namespace)
	for i, pvcTmpl := range woc.execWf.Spec.VolumeClaimTemplates {
		if pvcTmpl.ObjectMeta.Name == "" {
			return errors.Errorf(errors.CodeBadRequest, "volumeClaimTemplates[%d].metadata.name is required", i)
		}
		pvcTmpl = *pvcTmpl.DeepCopy()
		// PVC name will be <workflowname>-<volumeclaimtemplatename>
		refName := pvcTmpl.ObjectMeta.Name
		pvcName := fmt.Sprintf("%s-%s", woc.wf.ObjectMeta.Name, pvcTmpl.ObjectMeta.Name)
		woc.log.Infof("Creating pvc %s", pvcName)
		pvcTmpl.ObjectMeta.Name = pvcName
		if pvcTmpl.ObjectMeta.Labels == nil {
			pvcTmpl.ObjectMeta.Labels = make(map[string]string)
		}
		pvcTmpl.ObjectMeta.Labels[common.LabelKeyWorkflow] = woc.wf.ObjectMeta.Name
		pvcTmpl.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
		}
		pvc, err := pvcClient.Create(&pvcTmpl)
		if err != nil && apierr.IsAlreadyExists(err) {
			woc.log.WithField("pvc", pvcTmpl.Name).Info("pvc already exists. Workflow is re-using it")
			pvc, err = pvcClient.Get(pvcTmpl.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			hasOwnerReference := false
			for i := range pvc.OwnerReferences {
				ownerRef := pvc.OwnerReferences[i]
				if ownerRef.UID == woc.wf.UID {
					hasOwnerReference = true
					break
				}
			}
			if !hasOwnerReference {
				return errors.Errorf(errors.CodeForbidden, "%s pvc already exists with different ownerreference", pvcTmpl.Name)
			}
		}

		//continue
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
		woc.wf.Status.PersistentVolumeClaims = append(woc.wf.Status.PersistentVolumeClaims, vol)
		woc.updated = true
	}
	return nil
}

func (woc *wfOperationCtx) deletePVCs() error {
	gcStrategy := woc.wf.Spec.GetVolumeClaimGC().GetStrategy()

	switch gcStrategy {
	case wfv1.VolumeClaimGCOnSuccess:
		if woc.wf.Status.Phase == wfv1.NodeError || woc.wf.Status.Phase == wfv1.NodeFailed {
			// Skip deleting PVCs to reuse them for retried failed/error workflows.
			// PVCs are automatically deleted when corresponded owner workflows get deleted.
			return nil
		}
	case wfv1.VolumeClaimGCOnCompletion:
	default:
		return fmt.Errorf("unknown volume gc strategy: %s", gcStrategy)
	}

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

func getChildNodeIndex(node *wfv1.NodeStatus, nodes wfv1.Nodes, index int) *wfv1.NodeStatus {
	if len(node.Children) <= 0 {
		return nil
	}

	nodeIndex := index
	if index < 0 {
		nodeIndex = len(node.Children) + index // This actually subtracts, since index is negative
		if nodeIndex < 0 {
			panic(fmt.Sprintf("child index '%d' out of bounds", index))
		}
	}

	lastChildNodeName := node.Children[nodeIndex]
	lastChildNode, ok := nodes[lastChildNodeName]
	if !ok {
		panic("could not find child node")
	}

	return &lastChildNode
}

type executeTemplateOpts struct {
	// boundaryID is an ID for node grouping
	boundaryID string
	// onExitTemplate signifies that executeTemplate was called as part of an onExit handler.
	// Necessary for graceful shutdowns
	onExitTemplate bool
	// activeDeadlineSeconds is a deadline to set to any pods executed. This is necessary for pods to inherit backoff.maxDuration
	executionDeadline time.Time
}

// executeTemplate executes the template with the given arguments and returns the created NodeStatus
// for the created node (if created). Nodes may not be created if parallelism or deadline exceeded.
// nodeName is the name to be used as the name of the node, and boundaryID indicates which template
// boundary this node belongs to.
func (woc *wfOperationCtx) executeTemplate(nodeName string, orgTmpl wfv1.TemplateReferenceHolder, tmplCtx *templateresolution.Context, args wfv1.Arguments, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	woc.log.Debugf("Evaluating node %s: template: %s, boundaryID: %s", nodeName, common.GetTemplateHolderString(orgTmpl), opts.boundaryID)

	node := woc.wf.GetNodeByName(nodeName)

	// Set templateScope from which the template resolution starts.
	templateScope := tmplCtx.GetTemplateScope()
	newTmplCtx, resolvedTmpl, templateStored, err := tmplCtx.ResolveTemplate(orgTmpl)
	if err != nil {
		return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
	}
	// A new template was stored during resolution, persist it
	if templateStored {
		woc.updated = true
	}

	localParams := make(map[string]string)
	// Inject the pod name. If the pod has a retry strategy, the pod name will be changed and will be injected when it
	// is determined
	if resolvedTmpl.IsPodType() && woc.retryStrategy(resolvedTmpl) == nil {
		localParams[common.LocalVarPodName] = woc.wf.NodeID(nodeName)
	}

	// Inputs has been processed with arguments already, so pass empty arguments.
	processedTmpl, err := common.ProcessArgs(resolvedTmpl, &args, woc.globalParams, localParams, false)
	if err != nil {
		return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
	}

	// If memoization is on, check if node output exists in cache
	if node == nil && processedTmpl.Memoize != nil {
		memoizationCache := woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, processedTmpl.Memoize.Cache.ConfigMap.Name)
		if memoizationCache == nil {
			err := fmt.Errorf("cache could not be found or created")
			woc.log.WithFields(log.Fields{"cacheName": processedTmpl.Memoize.Cache.ConfigMap.Name}).WithError(err)
			return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
		}

		entry, err := memoizationCache.Load(processedTmpl.Memoize.Key)
		if err != nil {
			return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
		}

		hit := entry.Hit()
		var outputs *wfv1.Outputs
		if processedTmpl.Memoize.MaxAge != "" {
			maxAge, err := time.ParseDuration(processedTmpl.Memoize.MaxAge)
			if err != nil {
				err := fmt.Errorf("invalid maxAge: %s", err)
				return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
			}
			maxAgeOutputs, ok := entry.GetOutputsWithMaxAge(maxAge)
			if !ok {
				// The outputs are expired, so this cache entry is not hit
				hit = false
			}
			outputs = maxAgeOutputs
		} else {
			outputs = entry.GetOutputs()
		}

		memoizationStatus := &wfv1.MemoizationStatus{
			Hit:       hit,
			Key:       processedTmpl.Memoize.Key,
			CacheName: processedTmpl.Memoize.Cache.ConfigMap.Name,
		}
		if hit {
			node = woc.initializeCacheHitNode(nodeName, processedTmpl, templateScope, orgTmpl, opts.boundaryID, outputs, memoizationStatus)
		} else {
			node = woc.initializeCacheNode(nodeName, processedTmpl, templateScope, orgTmpl, opts.boundaryID, memoizationStatus)
		}
		woc.wf.Status.Nodes[node.ID] = *node
		woc.updated = true
	}

	if node != nil {
		if node.Fulfilled() {
			if processedTmpl.Synchronization != nil {
				woc.controller.syncManager.Release(woc.wf, node.ID, processedTmpl.Synchronization)
			}
			woc.log.Debugf("Node %s already completed", nodeName)
			if processedTmpl.Metrics != nil {
				// Check if this node completed between executions. If it did, emit metrics. If a node completes within
				// the same execution, its metrics are emitted below.
				// We can infer that this node completed during the current operation, emit metrics
				if prevNodeStatus, ok := woc.preExecutionNodePhases[node.ID]; ok && !prevNodeStatus.Fulfilled() {
					localScope, realTimeScope := woc.prepareMetricScope(node)
					woc.computeMetrics(processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
				}
			}
			return node, nil
		}
		woc.log.Debugf("Executing node %s of %s is %s", nodeName, node.Type, node.Phase)
		// Memoized nodes don't have StartedAt.
		if node.StartedAt.IsZero() {
			node.StartedAt = metav1.Time{Time: time.Now().UTC()}
			node.EstimatedDuration = woc.estimateNodeDuration(node.Name)
			woc.wf.Status.Nodes[node.ID] = *node
			woc.updated = true
		}
	}

	// Check if we took too long operating on this workflow and immediately return if we did
	if time.Now().UTC().After(woc.deadline) {
		woc.log.Warnf("Deadline exceeded")
		woc.requeue(defaultRequeueTime)
		return node, ErrDeadlineExceeded
	}

	// Check the template deadline for Pending nodes
	// This check will cover the resource forbidden, synchronization scenario,
	// In above scenario, only Node will be created in pending state
	_, err = woc.checkTemplateTimeout(processedTmpl, node)
	if err != nil {
		woc.log.Warnf("Template %s exceeded its deadline", processedTmpl.Name)
		return woc.markNodePhase(nodeName, wfv1.NodeFailed, err.Error()), err
	}

	// Check if we exceeded template or workflow parallelism and immediately return if we did
	if err := woc.checkParallelism(processedTmpl, node, opts.boundaryID); err != nil {
		return node, err
	}

	if processedTmpl.Synchronization != nil {
		lockAcquired, wfUpdated, msg, err := woc.controller.syncManager.TryAcquire(woc.wf, woc.wf.NodeID(nodeName), processedTmpl.Synchronization)
		if err != nil {
			return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
		}
		if !lockAcquired {
			if node == nil {
				node = woc.initializeExecutableNode(nodeName, wfutil.GetNodeType(processedTmpl), templateScope, processedTmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, msg)
			}
			lockName, err := argosync.GetLockName(processedTmpl.Synchronization, woc.wf.Namespace)
			if err != nil {
				// If an error were to be returned here, it would have been caught by TryAcquire. If it didn't, then it is
				// unexpected behavior and is a bug.
				panic("bug: GetLockName should not return an error after a call to TryAcquire")
			}
			return woc.markNodeWaitingForLock(node.Name, lockName.EncodeName()), nil
		} else {
			woc.log.Infof("Node %s acquired synchronization lock", nodeName)
			if node != nil {
				node.Message = ""
				node = woc.markNodeWaitingForLock(node.Name, "")
			}
		}

		woc.updated = wfUpdated
	}
	// If the user has specified retries, node becomes a special retry node.
	// This node acts as a parent of all retries that will be done for
	// the container. The status of this node should be "Success" if any
	// of the retries succeed. Otherwise, it is "Failed".
	retryNodeName := ""
	if woc.retryStrategy(processedTmpl) != nil {
		retryNodeName = nodeName
		retryParentNode := node
		if retryParentNode == nil {
			woc.log.Debugf("Inject a retry node for node %s", retryNodeName)
			retryParentNode = woc.initializeExecutableNode(retryNodeName, wfv1.NodeTypeRetry, templateScope, processedTmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning)
		}
		processedRetryParentNode, continueExecution, err := woc.processNodeRetries(retryParentNode, *woc.retryStrategy(processedTmpl), opts)
		if err != nil {
			return woc.markNodeError(retryNodeName, err), err
		} else if !continueExecution {
			// We are still waiting for a retry delay to finish
			return retryParentNode, nil
		}
		retryParentNode = processedRetryParentNode
		// The retry node might have completed by now.
		if retryParentNode.Fulfilled() {
			if processedTmpl.Metrics != nil {
				// In this check, a completed node may or may not have existed prior to this execution. If it did exist, ensure that it wasn't
				// completed before this execution. If it did not exist prior, then we can infer that it was completed during this execution.
				// The statement "(!ok || !prevNodeStatus.Fulfilled())" checks for this behavior and represents the material conditional
				// "ok -> !prevNodeStatus.Fulfilled()" (https://en.wikipedia.org/wiki/Material_conditional)
				if prevNodeStatus, ok := woc.preExecutionNodePhases[retryParentNode.ID]; (!ok || !prevNodeStatus.Fulfilled()) && retryParentNode.Fulfilled() {
					localScope, realTimeScope := woc.prepareMetricScope(node)
					woc.computeMetrics(processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
				}
			}
			return retryParentNode, nil
		}
		lastChildNode := getChildNodeIndex(retryParentNode, woc.wf.Status.Nodes, -1)
		if lastChildNode != nil && !lastChildNode.Fulfilled() {
			// Last child node is still running.
			nodeName = lastChildNode.Name
			node = lastChildNode
		} else {
			// Create a new child node and append it to the retry node.
			nodeName = fmt.Sprintf("%s(%d)", retryNodeName, len(retryParentNode.Children))
			woc.addChildNode(retryNodeName, nodeName)
			node = nil

			localParams := make(map[string]string)
			// Change the `pod.name` variable to the new retry node name
			if processedTmpl.IsPodType() {
				localParams[common.LocalVarPodName] = woc.wf.NodeID(nodeName)
			}
			// Inject the retryAttempt number
			localParams[common.LocalVarRetries] = strconv.Itoa(len(retryParentNode.Children))

			processedTmpl, err = common.SubstituteParams(processedTmpl, map[string]string{}, localParams)
			if err != nil {
				return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
			}
		}
	}

	switch processedTmpl.GetType() {
	case wfv1.TemplateTypeContainer:
		node, err = woc.executeContainer(nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeSteps:
		node, err = woc.executeSteps(nodeName, newTmplCtx, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeScript:
		node, err = woc.executeScript(nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeResource:
		node, err = woc.executeResource(nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeDAG:
		node, err = woc.executeDAG(nodeName, newTmplCtx, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeSuspend:
		node, err = woc.executeSuspend(nodeName, templateScope, processedTmpl, orgTmpl, opts)
	default:
		err = errors.Errorf(errors.CodeBadRequest, "Template '%s' missing specification", processedTmpl.Name)
		return woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, templateScope, orgTmpl, opts.boundaryID, wfv1.NodeError, err.Error()), err
	}
	if err != nil {
		node = woc.markNodeError(nodeName, err)

		if processedTmpl.Synchronization != nil {
			woc.controller.syncManager.Release(woc.wf, node.ID, processedTmpl.Synchronization)
		}

		// If retry policy is not set, or if it is not set to Always or OnError, we won't attempt to retry an errored container
		// and we return instead.
		retryStrategy := woc.retryStrategy(processedTmpl)
		if retryStrategy == nil ||
			(retryStrategy.RetryPolicy != wfv1.RetryPolicyAlways &&
				retryStrategy.RetryPolicy != wfv1.RetryPolicyOnError) {
			return node, err
		}
	}

	if processedTmpl.Metrics != nil {
		// Check if the node was just created, if it was emit realtime metrics.
		// If the node did not previously exist, we can infer that it was created during the current operation, emit real time metrics.
		if _, ok := woc.preExecutionNodePhases[node.ID]; !ok {
			localScope, realTimeScope := woc.prepareMetricScope(node)
			woc.computeMetrics(processedTmpl.Metrics.Prometheus, localScope, realTimeScope, true)
		}
		// Check if the node completed during this execution, if it did emit metrics
		//
		// This check is necessary because sometimes a node will be marked completed during the current execution and will
		// not be considered again. The best example of this is the entrypoint steps/dag template (once completed, the
		// workflow ends and it's not reconsidered). This checks makes sure that its metrics also get emitted.
		//
		// In this check, a completed node may or may not have existed prior to this execution. If it did exist, ensure that it wasn't
		// completed before this execution. If it did not exist prior, then we can infer that it was completed during this execution.
		// The statement "(!ok || !prevNodeStatus.Fulfilled())" checks for this behavior and represents the material conditional
		// "ok -> !prevNodeStatus.Fulfilled()" (https://en.wikipedia.org/wiki/Material_conditional)
		if prevNodeStatus, ok := woc.preExecutionNodePhases[node.ID]; (!ok || !prevNodeStatus.Fulfilled()) && node.Fulfilled() {
			localScope, realTimeScope := woc.prepareMetricScope(node)
			woc.computeMetrics(processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
		}
	}

	node = woc.wf.GetNodeByName(node.Name)

	// Swap the node back to retry node
	if retryNodeName != "" {
		retryNode := woc.wf.GetNodeByName(retryNodeName)
		if !retryNode.Fulfilled() && node.Fulfilled() { //if the retry child has completed we need to update outself
			node, err = woc.executeTemplate(retryNodeName, orgTmpl, tmplCtx, args, opts)
			if err != nil {
				return woc.markNodeError(node.Name, err), err
			}
		}
		node = retryNode
	}

	return node, nil
}

// Checks if the template has exceeded its deadline
func (woc *wfOperationCtx) checkTemplateTimeout(tmpl *wfv1.Template, node *wfv1.NodeStatus) (*time.Time, error) {
	if node == nil {
		return nil, nil
	}

	if tmpl.Timeout != "" {
		tmplTimeout, err := time.ParseDuration(tmpl.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format. %v", err)
		}

		deadline := node.StartedAt.Add(tmplTimeout)

		if node.Phase == wfv1.NodePending && time.Now().After(deadline) {
			return nil, ErrTimeout
		}
		return &deadline, nil
	}

	return nil, nil
}

// markWorkflowPhase is a convenience method to set the phase of the workflow with optional message
// optionally marks the workflow completed, which sets the finishedAt timestamp and completed label
func (woc *wfOperationCtx) markWorkflowPhase(phase wfv1.NodePhase, message string) {
	markCompleted := false
	if woc.wf.Status.Phase != phase {
		if woc.wf.Status.Phase.Fulfilled() {
			woc.log.WithFields(log.Fields{"fromPhase": woc.wf.Status.Phase, "toPhase": phase}).
				Panic("workflow is already fulfilled")
		}
		woc.log.Infof("Updated phase %s -> %s", woc.wf.Status.Phase, phase)
		woc.updated = true
		woc.wf.Status.Phase = phase
		if woc.wf.ObjectMeta.Labels == nil {
			woc.wf.ObjectMeta.Labels = make(map[string]string)
		}
		woc.wf.ObjectMeta.Labels[common.LabelKeyPhase] = string(phase)
		switch phase {
		case wfv1.NodeRunning:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeNormal, "WorkflowRunning", "Workflow Running")
		case wfv1.NodeSucceeded:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeNormal, "WorkflowSucceeded", "Workflow completed")
		case wfv1.NodeFailed, wfv1.NodeError:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowFailed", message)
		}
		markCompleted = phase.Completed()
	}
	if woc.wf.Status.StartedAt.IsZero() {
		woc.updated = true
		woc.wf.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
		woc.wf.Status.EstimatedDuration = woc.estimateWorkflowDuration()
	}
	if woc.wf.Status.Message != message {
		woc.log.Infof("Updated message %s -> %s", woc.wf.Status.Message, message)
		woc.updated = true
		woc.wf.Status.Message = message
	}

	if phase == wfv1.NodeError {
		entryNode, ok := woc.wf.Status.Nodes[woc.wf.ObjectMeta.Name]
		if ok && entryNode.Phase == wfv1.NodeRunning {
			entryNode.Phase = wfv1.NodeError
			entryNode.Message = "Workflow operation error"
			woc.wf.Status.Nodes[woc.wf.ObjectMeta.Name] = entryNode
			woc.updated = true
		}
	}

	switch phase {
	case wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeError:
		// wait for all daemon nodes to get terminated before marking workflow completed
		if markCompleted && !woc.hasDaemonNodes() {
			woc.log.Infof("Marking workflow completed")
			woc.wf.Status.FinishedAt = metav1.Time{Time: time.Now().UTC()}
			woc.globalParams[common.GlobalVarWorkflowDuration] = fmt.Sprintf("%f", woc.wf.Status.FinishedAt.Sub(woc.wf.Status.StartedAt.Time).Seconds())
			if woc.wf.ObjectMeta.Labels == nil {
				woc.wf.ObjectMeta.Labels = make(map[string]string)
			}
			woc.wf.ObjectMeta.Labels[common.LabelKeyCompleted] = "true"
			woc.wf.Status.Conditions.UpsertCondition(wfv1.Condition{Status: metav1.ConditionTrue, Type: wfv1.ConditionTypeCompleted})
			err := woc.deletePDBResource()
			if err != nil {
				woc.wf.Status.Phase = wfv1.NodeError
				woc.wf.ObjectMeta.Labels[common.LabelKeyPhase] = string(wfv1.NodeError)
				woc.updated = true
				woc.wf.Status.Message = err.Error()
			}
			if woc.controller.wfArchive.IsEnabled() {
				if woc.controller.isArchivable(woc.wf) {
					woc.log.Infof("Marking workflow as pending archiving")
					woc.wf.Labels[common.LabelKeyWorkflowArchivingStatus] = "Pending"
				} else {
					woc.log.Infof("Doesn't match with archive label selector. Skipping Archive")
				}
			}
			woc.updated = true
		}
	}
}

// get a predictor, this maybe null implementation in the case of rare error
func (woc *wfOperationCtx) getEstimator() estimation.Estimator {
	if woc.estimator == nil {
		woc.estimator, _ = woc.controller.estimatorFactory.NewEstimator(woc.wf)
	}
	return woc.estimator
}

func (woc *wfOperationCtx) estimateWorkflowDuration() wfv1.EstimatedDuration {
	return woc.getEstimator().EstimateWorkflowDuration()
}

func (woc *wfOperationCtx) estimateNodeDuration(nodeName string) wfv1.EstimatedDuration {
	return woc.getEstimator().EstimateNodeDuration(nodeName)
}

func (woc *wfOperationCtx) hasDaemonNodes() bool {
	for _, node := range woc.wf.Status.Nodes {
		if node.IsDaemoned() {
			return true
		}
	}
	return false
}

func (woc *wfOperationCtx) markWorkflowRunning() {
	woc.markWorkflowPhase(wfv1.NodeRunning, "")
}

func (woc *wfOperationCtx) markWorkflowSuccess() {
	woc.markWorkflowPhase(wfv1.NodeSucceeded, "")
}

func (woc *wfOperationCtx) markWorkflowFailed(message string) {
	woc.markWorkflowPhase(wfv1.NodeFailed, message)
}

func (woc *wfOperationCtx) markWorkflowError(err error) {
	woc.markWorkflowPhase(wfv1.NodeError, err.Error())
}

// stepsOrDagSeparator identifies if a node name starts with our naming convention separator from
// DAG or steps templates. Will match stings with prefix like: [0]. or .
var stepsOrDagSeparator = regexp.MustCompile(`^(\[\d+\])?\.`)

// initializeExecutableNode initializes a node and stores the template.
func (woc *wfOperationCtx) initializeExecutableNode(nodeName string, nodeType wfv1.NodeType, templateScope string, executeTmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, phase wfv1.NodePhase, messages ...string) *wfv1.NodeStatus {
	node := woc.initializeNode(nodeName, nodeType, templateScope, orgTmpl, boundaryID, phase)

	// Set the input values to the node.
	if executeTmpl.Inputs.HasInputs() {
		node.Inputs = executeTmpl.Inputs.DeepCopy()
	}

	if nodeType == wfv1.NodeTypeSuspend {
		node = addRawOutputFields(node, executeTmpl)
	}

	if len(messages) > 0 {
		node.Message = messages[0]
	}

	// Update the node
	woc.wf.Status.Nodes[node.ID] = *node
	woc.updated = true

	return node
}

// initializeNodeOrMarkError initializes an error node or mark a node if it already exists.
func (woc *wfOperationCtx) initializeNodeOrMarkError(node *wfv1.NodeStatus, nodeName string, templateScope string, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, err error) *wfv1.NodeStatus {
	if node != nil {
		return woc.markNodeError(nodeName, err)
	}
	return woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, templateScope, orgTmpl, boundaryID, wfv1.NodeError, err.Error())
}

// Creates a node status that is or will be chaced
func (woc *wfOperationCtx) initializeCacheNode(nodeName string, resolvedTmpl *wfv1.Template, templateScope string, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, memStat *wfv1.MemoizationStatus, messages ...string) *wfv1.NodeStatus {
	if resolvedTmpl.Memoize == nil {
		err := fmt.Errorf("cannot initialize a cached node from a non-memoized template")
		woc.log.WithFields(log.Fields{"namespace": woc.wf.Namespace, "wfName": woc.wf.Name}).WithError(err)
		panic(err)
	}
	woc.log.Debug("Initializing cached node ", nodeName, common.GetTemplateHolderString(orgTmpl), boundaryID)
	node := woc.initializeExecutableNode(nodeName, wfutil.GetNodeType(resolvedTmpl), templateScope, resolvedTmpl, orgTmpl, boundaryID, wfv1.NodePending, messages...)
	node.MemoizationStatus = memStat
	return node
}

// Creates a node status that has been cached, completely initialized, and marked as finished
func (woc *wfOperationCtx) initializeCacheHitNode(nodeName string, resolvedTmpl *wfv1.Template, templateScope string, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, outputs *wfv1.Outputs, memStat *wfv1.MemoizationStatus, messages ...string) *wfv1.NodeStatus {
	node := woc.initializeCacheNode(nodeName, resolvedTmpl, templateScope, orgTmpl, boundaryID, memStat, messages...)
	node.Phase = wfv1.NodeSucceeded
	node.Outputs = outputs
	node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
	return node
}

func (woc *wfOperationCtx) initializeNode(nodeName string, nodeType wfv1.NodeType, templateScope string, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, phase wfv1.NodePhase, messages ...string) *wfv1.NodeStatus {
	woc.log.Debugf("Initializing node %s: template: %s, boundaryID: %s", nodeName, common.GetTemplateHolderString(orgTmpl), boundaryID)

	nodeID := woc.wf.NodeID(nodeName)
	_, ok := woc.wf.Status.Nodes[nodeID]
	if ok {
		panic(fmt.Sprintf("node %s already initialized", nodeName))
	}

	node := wfv1.NodeStatus{
		ID:                nodeID,
		Name:              nodeName,
		TemplateName:      orgTmpl.GetTemplateName(),
		TemplateRef:       orgTmpl.GetTemplateRef(),
		TemplateScope:     templateScope,
		Type:              nodeType,
		BoundaryID:        boundaryID,
		Phase:             phase,
		StartedAt:         metav1.Time{Time: time.Now().UTC()},
		EstimatedDuration: woc.estimateNodeDuration(nodeName),
	}

	if boundaryNode, ok := woc.wf.Status.Nodes[boundaryID]; ok {
		node.DisplayName = strings.TrimPrefix(node.Name, boundaryNode.Name)
		if stepsOrDagSeparator.MatchString(node.DisplayName) {
			node.DisplayName = stepsOrDagSeparator.ReplaceAllString(node.DisplayName, "")
		}
	} else {
		node.DisplayName = nodeName
	}

	if node.Fulfilled() && node.FinishedAt.IsZero() {
		node.FinishedAt = node.StartedAt
	}
	var message string
	if len(messages) > 0 {
		message = fmt.Sprintf(" (message: %s)", messages[0])
		node.Message = messages[0]
	}
	woc.wf.Status.Nodes[nodeID] = node
	woc.log.Infof("%s node %v initialized %s%s", node.Type, node.ID, node.Phase, message)
	woc.updated = true
	return &node
}

// markNodePhase marks a node with the given phase, creating the node if necessary and handles timestamps
func (woc *wfOperationCtx) markNodePhase(nodeName string, phase wfv1.NodePhase, message ...string) *wfv1.NodeStatus {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		panic(fmt.Sprintf("workflow '%s' node '%s' uninitialized when marking as %v: %s", woc.wf.Name, nodeName, phase, message))
	}
	if node.Phase != phase {
		if node.Phase.Fulfilled() {
			woc.log.WithFields(log.Fields{"nodeName": node.Name, "fromPhase": node.Phase, "toPhase": phase}).
				Error("node is already fulfilled")
		}
		woc.log.Infof("node %s phase %s -> %s", node.ID, node.Phase, phase)
		node.Phase = phase
		woc.updated = true
	}
	if len(message) > 0 {
		if message[0] != node.Message {
			woc.log.Infof("node %s message: %s", node.ID, message[0])
			node.Message = message[0]
			woc.updated = true
		}
	}
	if node.Fulfilled() && node.FinishedAt.IsZero() {
		node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
		woc.log.Infof("node %s finished: %s", node.ID, node.FinishedAt)
		woc.updated = true
	}
	if !woc.orig.Status.Nodes[node.ID].Fulfilled() && node.Fulfilled() {
		woc.onNodeComplete(node)
	}
	woc.wf.Status.Nodes[node.ID] = *node
	return node
}

func (woc *wfOperationCtx) onNodeComplete(node *wfv1.NodeStatus) {
	if !woc.controller.Config.NodeEvents.IsEnabled() {
		return
	}
	message := fmt.Sprintf("%v node %s", node.Phase, node.Name)
	if node.Message != "" {
		message = message + ": " + node.Message
	}
	eventType := apiv1.EventTypeWarning
	if node.Phase == wfv1.NodeSucceeded {
		eventType = apiv1.EventTypeNormal
	}
	woc.eventRecorder.AnnotatedEventf(
		woc.wf,
		map[string]string{
			common.AnnotationKeyNodeType: string(node.Type),
			common.AnnotationKeyNodeName: node.Name,
		},
		eventType,
		fmt.Sprintf("WorkflowNode%s", node.Phase),
		message,
	)
}

// markNodeError is a convenience method to mark a node with an error and set the message from the error
func (woc *wfOperationCtx) markNodeError(nodeName string, err error) *wfv1.NodeStatus {
	woc.log.WithError(err).WithField("nodeName", nodeName).Error("Mark error node")
	return woc.markNodePhase(nodeName, wfv1.NodeError, err.Error())
}

// markNodePending is a convenience method to mark a node and set the message from the error
func (woc *wfOperationCtx) markNodePending(nodeName string, err error) *wfv1.NodeStatus {
	woc.log.Infof("Mark node %s as Pending, due to: %+v", nodeName, err)
	node := woc.wf.GetNodeByName(nodeName)
	return woc.markNodePhase(nodeName, wfv1.NodePending, fmt.Sprintf("Pending %s", time.Since(node.StartedAt.Time)))
}

// markNodeWaitingForLock is a convenience method to mark that a node is waiting for a lock
func (woc *wfOperationCtx) markNodeWaitingForLock(nodeName string, lockName string) *wfv1.NodeStatus {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		return node
	}

	if node.SynchronizationStatus == nil {
		node.SynchronizationStatus = &wfv1.NodeSynchronizationStatus{}
	}

	if lockName == "" {
		// If we are no longer waiting for a lock, nil out the sync status
		node.SynchronizationStatus = nil
	} else {
		node.SynchronizationStatus.Waiting = lockName
	}

	woc.wf.Status.Nodes[node.ID] = *node
	woc.updated = true
	return node
}

// checkParallelism checks if the given template is able to be executed, considering the current active pods and workflow/template parallelism
func (woc *wfOperationCtx) checkParallelism(tmpl *wfv1.Template, node *wfv1.NodeStatus, boundaryID string) error {
	if woc.execWf.Spec.Parallelism != nil && woc.activePods >= *woc.execWf.Spec.Parallelism {
		woc.log.Infof("workflow active pod spec parallelism reached %d/%d", woc.activePods, *woc.execWf.Spec.Parallelism)
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
		fallthrough
	default:
		// if we are about to execute a pod, make our parent hasn't reached it's limit
		if boundaryID != "" && (node == nil || (node.Phase != wfv1.NodePending && node.Phase != wfv1.NodeRunning)) {
			boundaryNode, ok := woc.wf.Status.Nodes[boundaryID]
			if !ok {
				return errors.InternalError("boundaryNode not found")
			}
			tmplCtx, err := woc.createTemplateContext(boundaryNode.GetTemplateScope())
			if err != nil {
				return err
			}
			_, boundaryTemplate, templateStored, err := tmplCtx.ResolveTemplate(&boundaryNode)
			if err != nil {
				return err
			}
			// A new template was stored during resolution, persist it
			if templateStored {
				woc.updated = true
			}

			if boundaryTemplate != nil && boundaryTemplate.Parallelism != nil {
				activeSiblings := woc.countActiveChildren(boundaryID)
				woc.log.Debugf("counted %d/%d active children in boundary %s", activeSiblings, *boundaryTemplate.Parallelism, boundaryID)
				if activeSiblings >= *boundaryTemplate.Parallelism {
					woc.log.Infof("template (node %s) active children parallelism reached %d/%d", boundaryID, activeSiblings, *boundaryTemplate.Parallelism)
					return ErrParallelismReached
				}
			}
		}
	}
	return nil
}

func (woc *wfOperationCtx) executeContainer(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}

	// Check if the output of this container is referenced elsewhere in the Workflow. If so, make sure to include it during
	// execution.
	includeScriptOutput, err := woc.includeScriptOutput(nodeName, opts.boundaryID)
	if err != nil {
		return node, err
	}

	woc.log.Debugf("Executing node %s with container template: %v\n", nodeName, tmpl)
	_, err = woc.createWorkflowPod(nodeName, *tmpl.Container, tmpl, &createWorkflowPodOpts{
		includeScriptOutput: includeScriptOutput,
		onExitPod:           opts.onExitTemplate,
		executionDeadline:   opts.executionDeadline,
	})

	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}

	return node, err
}

func (woc *wfOperationCtx) getOutboundNodes(nodeID string) []string {
	node := woc.wf.Status.Nodes[nodeID]
	switch node.Type {
	case wfv1.NodeTypePod, wfv1.NodeTypeSkipped, wfv1.NodeTypeSuspend:
		return []string{node.ID}
	case wfv1.NodeTypeTaskGroup:
		if len(node.Children) == 0 {
			return []string{node.ID}
		}
		outboundNodes := make([]string, 0)
		for _, child := range node.Children {
			outboundNodes = append(outboundNodes, woc.getOutboundNodes(child)...)
		}
		return outboundNodes
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
			outbound = append(outbound, subOutIDs...)
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
			if param.ValueFrom == nil {
				return nil, fmt.Errorf("output parameters must have a valueFrom specified")
			}
			val, err := scope.resolveParameter(param.ValueFrom.Parameter)
			if err != nil {
				// We have a default value to use instead of returning an error
				if param.ValueFrom.Default != nil {
					val = param.ValueFrom.Default.String()
				} else {
					return nil, err
				}
			}
			param.Value = wfv1.AnyStringPtr(val)
			param.ValueFrom = nil
			outputs.Parameters = append(outputs.Parameters, param)
		}
	}
	if len(tmpl.Outputs.Artifacts) > 0 {
		outputs.Artifacts = make([]wfv1.Artifact, 0)
		for _, art := range tmpl.Outputs.Artifacts {
			resolvedArt, err := scope.resolveArtifact(art.From, art.SubPath)
			if err != nil {
				// If the artifact was not found and is optional, don't mark an error
				if strings.Contains(err.Error(), "Unable to resolve") && art.Optional {
					log.Warnf("Optional artifact '%s' was not found; it won't be available as an output", art.Name)
					continue
				}
				return nil, fmt.Errorf("unable to resolve outputs from scope: %s", err)
			}
			resolvedArt.Name = art.Name
			outputs.Artifacts = append(outputs.Artifacts, *resolvedArt)
		}
	}
	return &outputs, nil
}

// hasOutputResultRef will check given template output has any reference
func hasOutputResultRef(name string, parentTmpl *wfv1.Template) bool {

	var variableRefName string
	if parentTmpl.DAG != nil {
		variableRefName = "{{tasks." + name + ".outputs.result}}"
	} else if parentTmpl.Steps != nil {
		variableRefName = "{{steps." + name + ".outputs.result}}"
	}

	jsonValue, err := json.Marshal(parentTmpl)
	if err != nil {
		log.Warnf("Unable to marshal the template. %v, %v", parentTmpl, err)
	}

	return strings.Contains(string(jsonValue), variableRefName)
}

// getStepOrDAGTaskName will extract the node from NodeStatus Name
func getStepOrDAGTaskName(nodeName string) string {
	if strings.Contains(nodeName, ".") {
		name := nodeName[strings.LastIndex(nodeName, ".")+1:]
		// Retry, withItems and withParam scenario
		if indx := strings.Index(name, "("); indx > 0 {
			return name[0:indx]
		}
		return name
	}
	return nodeName
}

func extractMainCtrFromScriptTemplate(tmpl *wfv1.Template) apiv1.Container {
	mainCtr := tmpl.Script.Container
	// If script source is provided then pass all container args to the
	// script instead of passing them to the container command directly
	if tmpl.Script.Source != "" {
		mainCtr.Args = append([]string{common.ExecutorScriptSourcePath}, mainCtr.Args...)
	}
	return mainCtr
}

func (woc *wfOperationCtx) executeScript(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	} else if !node.Pending() {
		return node, nil
	}

	// Check if the output of this script is referenced elsewhere in the Workflow. If so, make sure to include it during
	// execution.
	includeScriptOutput, err := woc.includeScriptOutput(nodeName, opts.boundaryID)
	if err != nil {
		return node, err
	}

	mainCtr := extractMainCtrFromScriptTemplate(tmpl)
	_, err = woc.createWorkflowPod(nodeName, mainCtr, tmpl, &createWorkflowPodOpts{
		includeScriptOutput: includeScriptOutput,
		onExitPod:           opts.onExitTemplate,
		executionDeadline:   opts.executionDeadline,
	})
	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}
	return node, err
}

func (woc *wfOperationCtx) requeueIfTransientErr(err error, nodeName string) (*wfv1.NodeStatus, error) {
	if errorsutil.IsTransientErr(err) {
		// Our error was most likely caused by a lack of resources.
		woc.requeue(defaultRequeueTime)
		return woc.markNodePending(nodeName, err), nil
	}
	return nil, err
}

// buildLocalScope adds all of a nodes outputs to the local scope with the given prefix, as well
// as the global scope, if specified with a globalName
func (woc *wfOperationCtx) buildLocalScope(scope *wfScope, prefix string, node *wfv1.NodeStatus) {
	// It may be that the node is a retry node, in which case we want to get the outputs of the last node
	// in the retry group instead of the retry node itself.
	if node.Type == wfv1.NodeTypeRetry {
		node = getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	}

	if node.ID != "" {
		key := fmt.Sprintf("%s.id", prefix)
		scope.addParamToScope(key, node.ID)
	}

	if !node.StartedAt.Time.IsZero() {
		key := fmt.Sprintf("%s.startedAt", prefix)
		scope.addParamToScope(key, node.StartedAt.Time.Format(time.RFC3339))
	}

	if !node.FinishedAt.Time.IsZero() {
		key := fmt.Sprintf("%s.finishedAt", prefix)
		scope.addParamToScope(key, node.FinishedAt.Time.Format(time.RFC3339))
	}

	if node.PodIP != "" {
		key := fmt.Sprintf("%s.ip", prefix)
		scope.addParamToScope(key, node.PodIP)
	}
	if node.Phase != "" {
		key := fmt.Sprintf("%s.status", prefix)
		scope.addParamToScope(key, string(node.Phase))
	}
	woc.addOutputsToLocalScope(prefix, node.Outputs, scope)
}

func (woc *wfOperationCtx) addOutputsToLocalScope(prefix string, outputs *wfv1.Outputs, scope *wfScope) {
	if outputs == nil || scope == nil {
		return
	}
	if prefix != "workflow" && outputs.Result != nil {
		scope.addParamToScope(fmt.Sprintf("%s.outputs.result", prefix), *outputs.Result)
	}
	if prefix != "workflow" && outputs.ExitCode != nil {
		scope.addParamToScope(fmt.Sprintf("%s.exitCode", prefix), *outputs.ExitCode)
	}
	for _, param := range outputs.Parameters {
		if param.Value != nil {
			scope.addParamToScope(fmt.Sprintf("%s.outputs.parameters.%s", prefix, param.Name), param.Value.String())
		}
	}
	for _, art := range outputs.Artifacts {
		scope.addArtifactToScope(fmt.Sprintf("%s.outputs.artifacts.%s", prefix, art.Name), art)
	}
}

func (woc *wfOperationCtx) addOutputsToGlobalScope(outputs *wfv1.Outputs) {
	if outputs == nil {
		return
	}
	for _, param := range outputs.Parameters {
		woc.addParamToGlobalScope(param)
	}
	for _, art := range outputs.Artifacts {
		woc.addArtifactToGlobalScope(art, nil)
	}
}

// loopNodes is a node list which supports sorting by loop index
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
func (woc *wfOperationCtx) processAggregateNodeOutputs(tmpl *wfv1.Template, scope *wfScope, prefix string, childNodes []wfv1.NodeStatus) error {
	if len(childNodes) == 0 {
		return nil
	}
	// need to sort the child node list so that the order of outputs are preserved
	sort.Sort(loopNodes(childNodes))
	paramList := make([]map[string]string, 0)
	outputParamValueLists := make(map[string][]string)
	resultsList := make([]wfv1.Item, 0)
	for _, node := range childNodes {
		if node.Outputs == nil {
			continue
		}
		if len(node.Outputs.Parameters) > 0 {
			param := make(map[string]string)
			for _, p := range node.Outputs.Parameters {
				param[p.Name] = p.Value.String()
				outputParamValueList := outputParamValueLists[p.Name]
				outputParamValueList = append(outputParamValueList, p.Value.String())
				outputParamValueLists[p.Name] = outputParamValueList
			}
			paramList = append(paramList, param)
		}
		if node.Outputs.Result != nil {
			// Support the case where item may be a map
			var item wfv1.Item
			err := json.Unmarshal([]byte(*node.Outputs.Result), &item)
			if err != nil {
				return err
			}
			resultsList = append(resultsList, item)
		}
	}
	if tmpl.GetType() == wfv1.TemplateTypeScript {
		resultsJSON, err := json.Marshal(resultsList)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("%s.outputs.result", prefix)
		scope.addParamToScope(key, string(resultsJSON))
	}
	outputsJSON, err := json.Marshal(paramList)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s.outputs.parameters", prefix)
	scope.addParamToScope(key, string(outputsJSON))
	// Adding per-output aggregated value placeholders
	for outputName, valueList := range outputParamValueLists {
		key = fmt.Sprintf("%s.outputs.parameters.%s", prefix, outputName)
		valueListJSON, err := json.Marshal(valueList)
		if err != nil {
			return err
		}
		scope.addParamToScope(key, string(valueListJSON))
	}
	return nil
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
	paramName := fmt.Sprintf("workflow.outputs.parameters.%s", param.GlobalName)
	woc.globalParams[paramName] = param.Value.String()
	if index == -1 {
		woc.log.Infof("setting %s: '%s'", paramName, param.Value)
		gParam := wfv1.Parameter{Name: param.GlobalName, Value: param.Value}
		woc.wf.Status.Outputs.Parameters = append(woc.wf.Status.Outputs.Parameters, gParam)
		woc.updated = true
	} else {
		prevVal := *woc.wf.Status.Outputs.Parameters[index].Value
		if prevVal != *param.Value {
			woc.log.Infof("overwriting %s: '%s' -> '%s'", paramName, woc.wf.Status.Outputs.Parameters[index].Value, param.Value)
			woc.wf.Status.Outputs.Parameters[index].Value = param.Value
			woc.updated = true
		}
	}
}

// addArtifactToGlobalScope exports any desired node outputs to the global scope
// Optionally adds to a local scope if supplied
func (woc *wfOperationCtx) addArtifactToGlobalScope(art wfv1.Artifact, scope *wfScope) {
	if art.GlobalName == "" {
		return
	}
	globalArtName := fmt.Sprintf("workflow.outputs.artifacts.%s", art.GlobalName)
	if woc.wf.Status.Outputs != nil {
		for i, gArt := range woc.wf.Status.Outputs.Artifacts {
			if gArt.Name == art.GlobalName {
				// global output already exists. overwrite the value if different
				art.Name = art.GlobalName
				art.GlobalName = ""
				art.Path = ""
				if !reflect.DeepEqual(woc.wf.Status.Outputs.Artifacts[i], art) {
					woc.wf.Status.Outputs.Artifacts[i] = art
					if scope != nil {
						scope.addArtifactToScope(globalArtName, art)
					}
					woc.log.Infof("overwriting %s: %v", globalArtName, art)
					woc.updated = true
				}
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
	woc.log.Infof("setting %s: %v", globalArtName, art)
	woc.wf.Status.Outputs.Artifacts = append(woc.wf.Status.Outputs.Artifacts, art)
	if scope != nil {
		scope.addArtifactToScope(globalArtName, art)
	}
	woc.updated = true
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
func (woc *wfOperationCtx) executeResource(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {

	node := woc.wf.GetNodeByName(nodeName)

	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	} else if !node.Pending() {
		return node, nil
	}

	tmpl = tmpl.DeepCopy()

	// Try to unmarshal the given manifest.
	obj := unstructured.Unstructured{}
	err := yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &obj)
	if err != nil {
		return node, err
	}

	if tmpl.Resource.SetOwnerReference {
		ownerReferences := obj.GetOwnerReferences()
		obj.SetOwnerReferences(append(ownerReferences, *metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind))))
		bytes, err := yaml.Marshal(obj.Object)
		if err != nil {
			return node, err
		}
		tmpl.Resource.Manifest = string(bytes)
	}

	mainCtr := woc.newExecContainer(common.MainContainerName, tmpl)
	mainCtr.Command = []string{"argoexec", "resource", tmpl.Resource.Action}
	_, err = woc.createWorkflowPod(nodeName, *mainCtr, tmpl, &createWorkflowPodOpts{onExitPod: opts.onExitTemplate, executionDeadline: opts.executionDeadline})
	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}

	return node, err
}

func (woc *wfOperationCtx) executeSuspend(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeSuspend, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}
	woc.log.Infof("node %s suspended", nodeName)

	// If there is either an active workflow deadline, or if this node is suspended with a duration, then the workflow
	// will need to be requeued after a certain amount of time
	var requeueTime *time.Time

	if tmpl.Suspend.Duration != "" {
		node := woc.wf.GetNodeByName(nodeName)
		suspendDuration, err := parseStringToDuration(tmpl.Suspend.Duration)
		if err != nil {
			return node, err
		}
		suspendDeadline := node.StartedAt.Add(suspendDuration)
		requeueTime = &suspendDeadline
		if time.Now().UTC().After(suspendDeadline) {
			// Suspension is expired, node can be resumed
			woc.log.Infof("auto resuming node %s", nodeName)
			_ = woc.markNodePhase(nodeName, wfv1.NodeSucceeded)
			return node, nil
		}
	}

	// workflowDeadline is the time when the workflow will be timed out, if any
	if workflowDeadline := woc.getWorkflowDeadline(); workflowDeadline != nil {
		// There is an active workflow deadline. If this node is suspended with a duration, choose the earlier time
		// between the two, otherwise choose the deadline time.
		if requeueTime == nil || workflowDeadline.Before(*requeueTime) {
			requeueTime = workflowDeadline
		}
	}

	if requeueTime != nil {
		woc.requeue(time.Until(*requeueTime))
	}

	_ = woc.markNodePhase(nodeName, wfv1.NodeRunning)
	return node, nil
}

func addRawOutputFields(node *wfv1.NodeStatus, tmpl *wfv1.Template) *wfv1.NodeStatus {
	if tmpl.GetType() != wfv1.TemplateTypeSuspend || node.Type != wfv1.NodeTypeSuspend {
		panic("addRawOutputFields should only be used for nodes and templates of type suspend")
	}
	for _, param := range tmpl.Outputs.Parameters {
		if param.ValueFrom.Supplied != nil {
			if node.Outputs == nil {
				node.Outputs = &wfv1.Outputs{Parameters: []wfv1.Parameter{}}
			}
			node.Outputs.Parameters = append(node.Outputs.Parameters, param)
		}
	}
	return node
}

func parseStringToDuration(durationString string) (time.Duration, error) {
	var suspendDuration time.Duration
	// If no units are attached, treat as seconds
	if val, err := strconv.Atoi(durationString); err == nil {
		suspendDuration = time.Duration(val) * time.Second
	} else if duration, err := time.ParseDuration(durationString); err == nil {
		suspendDuration = duration
	} else {
		return 0, fmt.Errorf("unable to parse %s as a duration", durationString)
	}
	return suspendDuration, nil
}

func processItem(fstTmpl *fasttemplate.Template, name string, index int, item wfv1.Item, obj interface{}) (string, error) {
	replaceMap := make(map[string]string)
	var newName string

	switch item.GetType() {
	case wfv1.String, wfv1.Number, wfv1.Bool:
		replaceMap["item"] = fmt.Sprintf("%v", item)
		newName = generateNodeName(name, index, item)
	case wfv1.Map:
		// Handle the case when withItems is a list of maps.
		// vals holds stringified versions of the map items which are incorporated as part of the step name.
		// For example if the item is: {"name": "jesse","group":"developer"}
		// the vals would be: ["name:jesse", "group:developer"]
		// This would eventually be part of the step name (group:developer,name:jesse)
		vals := make([]string, 0)
		mapVal := item.GetMapVal()
		for itemKey, itemVal := range mapVal {
			replaceMap[fmt.Sprintf("item.%s", itemKey)] = fmt.Sprintf("%v", itemVal)
			vals = append(vals, fmt.Sprintf("%s:%v", itemKey, itemVal))

		}
		jsonByteVal, err := json.Marshal(mapVal)
		if err != nil {
			return "", errors.InternalWrapError(err)
		}
		replaceMap["item"] = string(jsonByteVal)

		// sort the values so that the name is deterministic
		sort.Strings(vals)
		newName = generateNodeName(name, index, strings.Join(vals, ","))
	case wfv1.List:
		listVal := item.GetListVal()
		byteVal, err := json.Marshal(listVal)
		if err != nil {
			return "", errors.InternalWrapError(err)
		}
		replaceMap["item"] = string(byteVal)
		newName = generateNodeName(name, index, listVal)
	default:
		return "", errors.Errorf(errors.CodeBadRequest, "withItems[%d] expected string, number, list, or map. received: %v", index, item)
	}
	newStepStr, err := common.Replace(fstTmpl, replaceMap, false)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal([]byte(newStepStr), &obj)
	if err != nil {
		return "", errors.InternalWrapError(err)
	}
	return newName, nil
}

func generateNodeName(name string, index int, desc interface{}) string {
	newName := fmt.Sprintf("%s(%d:%v)", name, index, desc)
	if out := util.RecoverIndexFromNodeName(newName); out != index {
		panic(fmt.Sprintf("unrecoverable digit in generateName; wanted '%d' and got '%d'", index, out))
	}
	return newName
}

func expandSequence(seq *wfv1.Sequence) ([]wfv1.Item, error) {
	var start, end int
	var err error
	if seq.Start != nil {
		start, err = strconv.Atoi(seq.Start.String())
		if err != nil {
			return nil, err
		}
	}
	if seq.End != nil {
		end, err = strconv.Atoi(seq.End.String())
		if err != nil {
			return nil, err
		}
	} else if seq.Count != nil {
		count, err := strconv.Atoi(seq.Count.String())
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return []wfv1.Item{}, nil
		}
		end = start + count - 1
	} else {
		return nil, errors.InternalError("neither end nor count was specified in withSequence")
	}
	items := make([]wfv1.Item, 0)
	format := "%d"
	if seq.Format != "" {
		format = seq.Format
	}
	if start <= end {
		for i := start; i <= end; i++ {
			item, err := wfv1.ParseItem(`"` + fmt.Sprintf(format, i) + `"`)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	} else {
		for i := start; i >= end; i-- {
			item, err := wfv1.ParseItem(`"` + fmt.Sprintf(format, i) + `"`)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (woc *wfOperationCtx) substituteParamsInVolumes(params map[string]string) error {
	if woc.volumes == nil {
		return nil
	}

	volumes := woc.volumes
	volumesBytes, err := json.Marshal(volumes)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	fstTmpl, err := fasttemplate.NewTemplate(string(volumesBytes), "{{", "}}")
	if err != nil {
		return fmt.Errorf("unable to parse argo varaible: %w", err)
	}
	newVolumesStr, err := common.Replace(fstTmpl, params, true)
	if err != nil {
		return err
	}
	var newVolumes []apiv1.Volume
	err = json.Unmarshal([]byte(newVolumesStr), &newVolumes)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	woc.volumes = newVolumes
	return nil
}

// createTemplateContext creates a new template context.
func (woc *wfOperationCtx) createTemplateContext(scope wfv1.ResourceScope, resourceName string) (*templateresolution.Context, error) {
	var clusterWorkflowTemplateGetter templateresolution.ClusterWorkflowTemplateGetter
	if woc.controller.cwftmplInformer != nil {
		clusterWorkflowTemplateGetter = woc.controller.cwftmplInformer.Lister()
	} else {
		clusterWorkflowTemplateGetter = &templateresolution.NullClusterWorkflowTemplateGetter{}
	}
	ctx := templateresolution.NewContext(woc.controller.wftmplInformer.Lister().WorkflowTemplates(woc.wf.Namespace), clusterWorkflowTemplateGetter, woc.execWf, woc.wf)

	switch scope {
	case wfv1.ResourceScopeNamespaced:
		return ctx.WithWorkflowTemplate(resourceName)
	case wfv1.ResourceScopeCluster:
		return ctx.WithClusterWorkflowTemplate(resourceName)
	default:
		return ctx, nil
	}
}

func (woc *wfOperationCtx) runOnExitNode(templateRef, parentDisplayName, parentNodeName, boundaryID string, tmplCtx *templateresolution.Context) (bool, *wfv1.NodeStatus, error) {
	if templateRef != "" && woc.wf.Spec.Shutdown.ShouldExecute(true) {
		woc.log.Infof("Running OnExit handler: %s", templateRef)
		onExitNodeName := common.GenerateOnExitNodeName(parentDisplayName)
		onExitNode, err := woc.executeTemplate(onExitNodeName, &wfv1.WorkflowStep{Template: templateRef}, tmplCtx, woc.execWf.Spec.Arguments, &executeTemplateOpts{
			boundaryID:     boundaryID,
			onExitTemplate: true,
		})
		woc.addChildNode(parentNodeName, onExitNodeName)
		return true, onExitNode, err
	}
	return false, nil, nil
}

func (woc *wfOperationCtx) computeMetrics(metricList []*wfv1.Prometheus, localScope map[string]string, realTimeScope map[string]func() float64, realTimeOnly bool) {
	for _, metricTmpl := range metricList {

		// Don't process real time metrics after execution
		if realTimeOnly && !metricTmpl.IsRealtime() {
			continue
		}

		if metricTmpl.Help == "" {
			woc.reportMetricEmissionError(fmt.Sprintf("metric '%s' must contain a help string under 'help: ' field", metricTmpl.Name))
			continue
		}

		// Substitute parameters in non-value fields of the template to support variables in places such as labels,
		// name, and help. We do not substitute value fields here (i.e. gauge, histogram, counter) here because they
		// might be realtime ({{workflow.duration}} will not be substituted the same way if it's realtime or if it isn't).
		metricTmplBytes, err := json.Marshal(metricTmpl)
		if err != nil {
			woc.reportMetricEmissionError(fmt.Sprintf("unable to substitute parameters for metric '%s' (marshal): %s", metricTmpl.Name, err))
			continue
		}
		fstTmpl, err := fasttemplate.NewTemplate(string(metricTmplBytes), "{{", "}}")
		if err != nil {
			woc.reportMetricEmissionError(fmt.Sprintf("unable to parse argo varaible for metric '%s': %s", metricTmpl.Name, err))
			continue
		}
		replacedValue, err := common.Replace(fstTmpl, localScope, false)
		if err != nil {
			woc.reportMetricEmissionError(fmt.Sprintf("unable to substitute parameters for metric '%s': %s", metricTmpl.Name, err))
			continue
		}

		var metricTmplSubstituted wfv1.Prometheus
		err = json.Unmarshal([]byte(replacedValue), &metricTmplSubstituted)
		if err != nil {
			woc.reportMetricEmissionError(fmt.Sprintf("unable to substitute parameters for metric '%s' (unmarshal): %s", metricTmpl.Name, err))
			continue
		}
		// Only substitute non-value fields here. Value field substitution happens below
		metricTmpl.Name = metricTmplSubstituted.Name
		metricTmpl.Help = metricTmplSubstituted.Help
		metricTmpl.Labels = metricTmplSubstituted.Labels
		metricTmpl.When = metricTmplSubstituted.When

		proceed, err := shouldExecute(metricTmpl.When)
		if err != nil {
			woc.reportMetricEmissionError(fmt.Sprintf("unable to compute 'when' clause for metric '%s': %s", woc.wf.ObjectMeta.Name, err))
			continue
		}
		if !proceed {
			continue
		}

		if metricTmpl.IsRealtime() {
			// Finally substitute value parameters
			value := metricTmpl.Gauge.Value
			if !(strings.HasPrefix(value, "{{") && strings.HasSuffix(value, "}}")) {
				woc.reportMetricEmissionError("real time metrics can only be used with metric variables")
				continue
			}
			value = strings.TrimSuffix(strings.TrimPrefix(value, "{{"), "}}")
			valueFunc, ok := realTimeScope[value]
			if !ok {
				woc.reportMetricEmissionError(fmt.Sprintf("'%s' is not available as a real time metric", value))
				continue
			}
			updatedMetric, err := metrics.ConstructRealTimeGaugeMetric(metricTmpl, valueFunc)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("could not construct metric '%s': %s", metricTmpl.Name, err))
				continue
			}
			err = woc.controller.metrics.UpsertCustomMetric(metricTmpl.GetDesc(), string(woc.wf.UID), updatedMetric, true)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("could not construct metric '%s': %s", metricTmpl.Name, err))
				continue
			}
			continue
		} else {
			metricSpec := metricTmpl.DeepCopy()

			// Finally substitute value parameters
			fstTmpl, err = fasttemplate.NewTemplate(metricSpec.GetValueString(), "{{", "}}")
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("unable to parse argo varaible for metric '%s': %s", metricTmpl.Name, err))
				continue
			}
			replacedValue, err := common.Replace(fstTmpl, localScope, false)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("unable to substitute parameters for metric '%s': %s", metricSpec.Name, err))
				continue
			}
			metricSpec.SetValueString(replacedValue)

			metric := woc.controller.metrics.GetCustomMetric(metricSpec.GetDesc())
			// It is valid to pass a nil metric to ConstructOrUpdateMetric, in that case the metric will be created for us
			updatedMetric, err := metrics.ConstructOrUpdateMetric(metric, metricSpec)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("could not construct metric '%s': %s", metricSpec.Name, err))
				continue
			}
			err = woc.controller.metrics.UpsertCustomMetric(metricSpec.GetDesc(), string(woc.wf.UID), updatedMetric, false)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("could not construct metric '%s': %s", metricSpec.Name, err))
				continue
			}
			continue
		}
	}
}

func (woc *wfOperationCtx) reportMetricEmissionError(errorString string) {
	woc.wf.Status.Conditions.UpsertConditionMessage(
		wfv1.Condition{
			Status:  metav1.ConditionTrue,
			Type:    wfv1.ConditionTypeMetricsError,
			Message: errorString,
		})
	woc.updated = true
	woc.log.Error(errorString)
}

func (woc *wfOperationCtx) createPDBResource() error {

	if woc.execWf.Spec.PodDisruptionBudget == nil {
		return nil
	}

	pdb, err := woc.controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets(woc.wf.Namespace).Get(woc.wf.Name, metav1.GetOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return err
	}
	if pdb != nil && pdb.Name != "" {
		return nil
	}

	pdbSpec := *woc.execWf.Spec.PodDisruptionBudget
	if pdbSpec.Selector == nil {
		pdbSpec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{common.LabelKeyWorkflow: woc.wf.Name},
		}
	}

	newPDB := policyv1beta.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:   woc.wf.Name,
			Labels: map[string]string{common.LabelKeyWorkflow: woc.wf.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: pdbSpec,
	}
	_, err = woc.controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets(woc.wf.Namespace).Create(&newPDB)
	if err != nil {
		return err
	}
	woc.log.Infof("Created PDB resource for workflow.")
	woc.updated = true
	return nil
}

func (woc *wfOperationCtx) deletePDBResource() error {
	if woc.execWf.Spec.PodDisruptionBudget == nil {
		return nil
	}
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		err := woc.controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets(woc.wf.Namespace).Delete(woc.wf.Name, &metav1.DeleteOptions{})
		if err != nil && !apierr.IsNotFound(err) {
			woc.log.WithField("err", err).Warn("Failed to delete PDB.")
			if !errorsutil.IsTransientErr(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		woc.log.WithField("err", err).Error("Unable to delete PDB resource for workflow.")
		return err
	}
	woc.log.Infof("Deleted PDB resource for workflow.")
	return nil
}

// Check if the output of this node is referenced elsewhere in the Workflow. If so, make sure to include it during
// execution.
func (woc *wfOperationCtx) includeScriptOutput(nodeName, boundaryID string) (bool, error) {
	if boundaryNode, ok := woc.wf.Status.Nodes[boundaryID]; ok {
		tmplCtx, err := woc.createTemplateContext(boundaryNode.GetTemplateScope())
		if err != nil {
			return false, err
		}
		_, parentTemplate, templateStored, err := tmplCtx.ResolveTemplate(&boundaryNode)
		if err != nil {
			return false, err
		}
		// A new template was stored during resolution, persist it
		if templateStored {
			woc.updated = true
		}

		name := getStepOrDAGTaskName(nodeName)
		return hasOutputResultRef(name, parentTemplate), nil
	}
	return false, nil
}

func (woc *wfOperationCtx) fetchWorkflowSpec() (wfv1.WorkflowSpecHolder, error) {
	if woc.wf.Spec.WorkflowTemplateRef == nil {
		return nil, fmt.Errorf("cannot fetch workflow spec without workflowTemplateRef")
	}

	var specHolder wfv1.WorkflowSpecHolder
	var err error
	// Logic for workflow refers Workflow template
	if woc.wf.Spec.WorkflowTemplateRef.ClusterScope {
		specHolder, err = woc.controller.cwftmplInformer.Lister().Get(woc.wf.Spec.WorkflowTemplateRef.Name)
	} else {
		specHolder, err = woc.controller.wftmplInformer.Lister().WorkflowTemplates(woc.wf.Namespace).Get(woc.wf.Spec.WorkflowTemplateRef.Name)
	}
	if err != nil {
		return nil, err
	}
	return specHolder, nil
}

func (woc *wfOperationCtx) retryStrategy(tmpl *wfv1.Template) *wfv1.RetryStrategy {
	if tmpl != nil && tmpl.RetryStrategy != nil {
		return tmpl.RetryStrategy
	}
	return woc.execWf.Spec.RetryStrategy
}

func (woc *wfOperationCtx) setExecWorkflow() error {
	if woc.wf.Spec.WorkflowTemplateRef != nil {
		err := woc.setStoredWfSpec()
		if err != nil {
			return err
		}
		woc.execWf = &wfv1.Workflow{Spec: *woc.wf.Status.StoredWorkflowSpec.DeepCopy()}
		woc.volumes = woc.execWf.Spec.DeepCopy().Volumes
	} else if woc.controller.Config.WorkflowRestrictions.MustUseReference() {
		return fmt.Errorf("workflows must use workflowTemplateRef to be executed when the controller is in reference mode")
	} else {
		err := woc.controller.setWorkflowDefaults(woc.wf)
		if err != nil {
			return err
		}
		woc.volumes = woc.wf.Spec.DeepCopy().Volumes
	}
	return nil
}

func (woc *wfOperationCtx) setStoredWfSpec() error {
	wfDefault := woc.controller.Config.WorkflowDefaults
	if wfDefault == nil {
		wfDefault = &wfv1.Workflow{}
	}
	if woc.wf.Status.StoredWorkflowSpec == nil {
		wftHolder, err := woc.fetchWorkflowSpec()
		if err != nil {
			return err
		}

		// Join WFT and WfDefault metadata to Workflow metadata.
		wfutil.JoinWorkflowMetaData(&woc.wf.ObjectMeta, wftHolder.GetWorkflowMetadata(), &wfDefault.ObjectMeta)

		// Join workflow, workflow template, and workflow default metadata to workflow spec.
		mergedWf, err := wfutil.JoinWorkflowSpec(&woc.wf.Spec, wftHolder.GetWorkflowSpec(), &wfDefault.Spec)
		if err != nil {
			return err
		}

		woc.wf.Status.StoredWorkflowSpec = &mergedWf.Spec
		woc.updated = true
	} else if woc.controller.Config.WorkflowRestrictions.MustNotChangeSpec() {
		wftHolder, err := woc.fetchWorkflowSpec()
		if err != nil {
			return err
		}
		mergedWf, err := wfutil.JoinWorkflowSpec(&woc.wf.Spec, wftHolder.GetWorkflowSpec(), &wfDefault.Spec)
		if err != nil {
			return err
		}
		if mergedWf.Spec.String() != woc.wf.Status.StoredWorkflowSpec.String() {
			return fmt.Errorf("workflowTemplateRef reference may not change during execution when the controller is in reference mode")

		}
	}
	return nil
}
