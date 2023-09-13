package controller

import (
	"context"
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

	"github.com/argoproj/argo-workflows/v3/util/secrets"

	"github.com/antonmedv/expr"
	"github.com/argoproj/pkg/humanize"
	argokubeerr "github.com/argoproj/pkg/kube/errors"
	"github.com/argoproj/pkg/strftime"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/diff"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/argoproj/argo-workflows/v3/util/help"
	"github.com/argoproj/argo-workflows/v3/util/intstr"
	"github.com/argoproj/argo-workflows/v3/util/resource"
	"github.com/argoproj/argo-workflows/v3/util/retry"
	argoruntime "github.com/argoproj/argo-workflows/v3/util/runtime"
	"github.com/argoproj/argo-workflows/v3/util/slice"
	"github.com/argoproj/argo-workflows/v3/util/template"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/estimation"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/progress"
	argosync "github.com/argoproj/argo-workflows/v3/workflow/sync"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	wfutil "github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

// wfOperationCtx is the context for evaluation and operation of a single workflow
type wfOperationCtx struct {
	// wf is the workflow object. It should not be used in execution logic. woc.execWf.Spec should be used instead
	wf *wfv1.Workflow
	// orig is the original workflow object for purposes of creating a patch
	orig *wfv1.Workflow
	// updated indicates whether or not the workflow object itself was updated
	// and needs to be persisted back to kubernetes
	updated bool
	// log is an logrus logging context to correlate logs with a workflow
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
	artifactRepository *wfv1.ArtifactRepository
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
	// 'execWf.Spec' should usually be used instead `wf.Spec`
	execWf *wfv1.Workflow

	taskSet map[string]wfv1.Template

	// currentStackDepth tracks the depth of the "stack", increased with every nested call to executeTemplate and decreased
	// when such calls return. This is used to prevent infinite recursion
	currentStackDepth int
}

var (
	// ErrDeadlineExceeded indicates the operation exceeded its deadline for execution
	ErrDeadlineExceeded = errors.New(errors.CodeTimeout, "Deadline exceeded")
	// ErrParallelismReached indicates this workflow reached its parallelism limit
	ErrParallelismReached       = errors.New(errors.CodeForbidden, "Max parallelism reached")
	ErrResourceRateLimitReached = errors.New(errors.CodeForbidden, "resource creation rate-limit reached")
	// ErrTimeout indicates a specific template timed out
	ErrTimeout = errors.New(errors.CodeTimeout, "timeout")
	// ErrMaxDepthExceeded indicates that the maximum recursion depth was exceeded
	ErrMaxDepthExceeded = errors.New(errors.CodeTimeout, fmt.Sprintf("Maximum recursion depth exceeded. See %s", help.ConfigureMaximumRecursionDepth))
)

// maxOperationTime is the maximum time a workflow operation is allowed to run
// for before requeuing the workflow onto the workqueue.
var (
	maxOperationTime = envutil.LookupEnvDurationOr("MAX_OPERATION_TIME", 30*time.Second)
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
		deadline:               time.Now().UTC().Add(maxOperationTime),
		eventRecorder:          wfc.eventRecorderManager.Get(wf.Namespace),
		preExecutionNodePhases: make(map[string]wfv1.NodePhase),
		taskSet:                make(map[string]wfv1.Template),
		currentStackDepth:      0,
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
func (woc *wfOperationCtx) operate(ctx context.Context) {
	defer argoruntime.RecoverFromPanic(woc.log)

	defer func() {
		if woc.wf.Status.Fulfilled() {
			woc.killDaemonedChildren("")
		}
		woc.persistUpdates(ctx)
	}()
	defer func() {
		if r := recover(); r != nil {
			woc.log.WithFields(log.Fields{"stack": string(debug.Stack()), "r": r}).Errorf("Recovered from panic")
			if rerr, ok := r.(error); ok {
				woc.markWorkflowError(ctx, rerr)
			} else {
				woc.markWorkflowError(ctx, fmt.Errorf("%v", r))
			}
			woc.controller.metrics.OperationPanic()
		}
	}()

	woc.log.Info("Processing workflow")

	// Set the Execute workflow spec for execution
	// ExecWF is a runtime execution spec which merged from Wf, WFT and Wfdefault
	err := woc.setExecWorkflow(ctx)
	if err != nil {
		woc.log.WithError(err).Errorf("Unable to set ExecWorkflow")
		return
	}

	if woc.wf.Status.ArtifactRepositoryRef == nil {
		ref, err := woc.controller.artifactRepositories.Resolve(ctx, woc.execWf.Spec.ArtifactRepositoryRef, woc.wf.Namespace)
		if err != nil {
			woc.markWorkflowError(ctx, fmt.Errorf("failed to resolve artifact repository: %w", err))
			return
		}
		woc.wf.Status.ArtifactRepositoryRef = ref
		woc.updated = true
	}

	repo, err := woc.controller.artifactRepositories.Get(ctx, woc.wf.Status.ArtifactRepositoryRef)
	if err != nil {
		woc.markWorkflowError(ctx, fmt.Errorf("failed to get artifact repository: %v", err))
		return
	}
	woc.artifactRepository = repo

	// check to see if we can do garbage collection of Artifacts; this is the only functionality in this method which can be called for 'Completed' Workflows,
	// so we can check for Completed Workflows after and return
	if err := woc.garbageCollectArtifacts(ctx); err != nil {
		woc.log.WithError(err).Error("failed to GC artifacts")
		return
	}
	if woc.wf.Labels[common.LabelKeyCompleted] == "true" { // abort now, we do not want to perform any more processing on a complete workflow because we could corrupt it
		return
	}

	// Workflow Level Synchronization lock
	if woc.execWf.Spec.Synchronization != nil {
		acquired, wfUpdate, msg, err := woc.controller.syncManager.TryAcquire(woc.wf, "", woc.execWf.Spec.Synchronization)
		if err != nil {
			woc.log.Warn("Failed to acquire the lock")
			woc.markWorkflowFailed(ctx, fmt.Sprintf("Failed to acquire the synchronization lock. %s", err.Error()))
			return
		}
		woc.updated = wfUpdate
		if !acquired {
			if !woc.releaseLocksForPendingShuttingdownWfs(ctx) {
				woc.log.Warn("Workflow processing has been postponed due to concurrency limit")
				phase := woc.wf.Status.Phase
				if phase == wfv1.WorkflowUnknown {
					phase = wfv1.WorkflowPending
				}
				woc.markWorkflowPhase(ctx, phase, msg)
				return
			}
		}
	}

	// Populate the phase of all the nodes prior to execution
	for _, node := range woc.wf.Status.Nodes {
		woc.preExecutionNodePhases[node.ID] = node.Phase
	}

	if woc.execWf.Spec.Metrics != nil {
		localScope, realTimeScope := woc.prepareDefaultMetricScope()
		woc.computeMetrics(woc.execWf.Spec.Metrics.Prometheus, localScope, realTimeScope, true)
	}

	if woc.wf.Status.Phase == wfv1.WorkflowUnknown {
		woc.markWorkflowRunning(ctx)
		setWfPodNamesAnnotation(woc.wf)

		err := woc.createPDBResource(ctx)
		if err != nil {
			msg := fmt.Sprintf("Unable to create PDB resource for workflow, %s error: %s", woc.wf.Name, err)
			woc.markWorkflowFailed(ctx, msg)
			return
		}

		woc.workflowDeadline = woc.getWorkflowDeadline()

		// Workflow will not be requeued if workflow steps are in pending state.
		// Workflow needs to requeue on its deadline,
		if woc.workflowDeadline != nil {
			woc.requeueAfter(time.Until(*woc.workflowDeadline))
		}

		woc.wf.Status.EstimatedDuration = woc.estimateWorkflowDuration()
	} else {
		woc.workflowDeadline = woc.getWorkflowDeadline()
		woc.taskResultReconciliation()
		err = woc.podReconciliation(ctx)
		if err == nil {
			woc.failSuspendedAndPendingNodesAfterDeadlineOrShutdown()
		}

		if err != nil {
			woc.log.WithError(err).WithField("workflow", woc.wf.ObjectMeta.Name).Error("workflow timeout")
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowTimedOut", "Workflow timed out")
			// TODO: we need to re-add to the workqueue, but should happen in caller
			return
		}
	}

	if woc.ShouldSuspend() {
		woc.log.Info("workflow suspended")
		return
	}
	if woc.execWf.Spec.Parallelism != nil {
		woc.activePods = woc.getActivePods("")
	}

	// Create a starting template context.
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	if err != nil {
		woc.log.WithError(err).Error("Failed to create a template context")
		woc.markWorkflowError(ctx, err)
		return
	}

	err = woc.substituteParamsInVolumes(woc.globalParams)
	if err != nil {
		woc.log.WithError(err).Error("volumes global param substitution error")
		woc.markWorkflowError(ctx, err)
		return
	}

	err = woc.createPVCs(ctx)
	if err != nil {
		if errorsutil.IsTransientErr(err) {
			// Error was most likely caused by a lack of resources.
			// In this case, Workflow will be in pending state and requeue.
			woc.markWorkflowPhase(ctx, wfv1.WorkflowPending, fmt.Sprintf("Waiting for a PVC to be created. %v", err))
			woc.requeue()
			return
		}
		err = fmt.Errorf("pvc create error: %w", err)
		woc.log.WithError(err).Error("pvc create error")
		woc.markWorkflowError(ctx, err)
		return
	} else if woc.wf.Status.Phase == wfv1.WorkflowPending {
		// Workflow might be in pending state if previous PVC creation is forbidden
		woc.markWorkflowRunning(ctx)
	}

	node, err := woc.executeTemplate(ctx, woc.wf.ObjectMeta.Name, &wfv1.WorkflowStep{Template: woc.execWf.Spec.Entrypoint}, tmplCtx, woc.execWf.Spec.Arguments, &executeTemplateOpts{})
	if err != nil {
		woc.log.WithError(err).Error("error in entry template execution")
		// we wrap this error up to report a clear message
		x := fmt.Errorf("error in entry template execution: %w", err)
		switch err {
		case ErrDeadlineExceeded:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowTimedOut", x.Error())
		case ErrParallelismReached:
		default:
			if !errorsutil.IsTransientErr(err) && !woc.wf.Status.Phase.Completed() && os.Getenv("BUBBLE_ENTRY_TEMPLATE_ERR") != "false" {
				woc.markWorkflowError(ctx, x)

				// Garbage collect PVCs if Entrypoint template execution returns error
				if err := woc.deletePVCs(ctx); err != nil {
					woc.log.WithError(err).Warn("failed to delete PVCs")
				}
			}
		}
		return
	}

	workflowStatus := map[wfv1.NodePhase]wfv1.WorkflowPhase{
		wfv1.NodePending:   wfv1.WorkflowPending,
		wfv1.NodeRunning:   wfv1.WorkflowRunning,
		wfv1.NodeSucceeded: wfv1.WorkflowSucceeded,
		wfv1.NodeSkipped:   wfv1.WorkflowSucceeded,
		wfv1.NodeFailed:    wfv1.WorkflowFailed,
		wfv1.NodeError:     wfv1.WorkflowError,
		wfv1.NodeOmitted:   wfv1.WorkflowSucceeded,
	}[node.Phase]

	woc.globalParams[common.GlobalVarWorkflowStatus] = string(workflowStatus)

	var failures []failedNodeStatus
	for _, node := range woc.wf.Status.Nodes {
		if node.Phase == wfv1.NodeFailed || node.Phase == wfv1.NodeError {
			failures = append(failures,
				failedNodeStatus{
					DisplayName:  node.DisplayName,
					Message:      node.Message,
					TemplateName: node.TemplateName,
					Phase:        string(node.Phase),
					PodName:      wfutil.GeneratePodName(woc.wf.Name, node.Name, node.TemplateName, node.ID, wfutil.GetPodNameVersion()),
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

	hookCompleted, err := woc.executeWfLifeCycleHook(ctx, tmplCtx)
	if err != nil {
		woc.markNodeError(node.Name, err)
	}
	// Reconcile TaskSet and Agent for HTTP templates
	woc.taskSetReconciliation(ctx)

	// Check all hooks are completes
	if !hookCompleted {
		return
	}

	if !node.Fulfilled() {
		// node can be nil if a workflow created immediately in a parallelism == 0 state
		return
	}

	var onExitNode *wfv1.NodeStatus
	if woc.execWf.Spec.HasExitHook() && woc.GetShutdownStrategy().ShouldExecute(true) {
		woc.log.Infof("Running OnExit handler: %s", woc.execWf.Spec.OnExit)
		onExitNodeName := common.GenerateOnExitNodeName(woc.wf.ObjectMeta.Name)
		exitHook := woc.execWf.Spec.GetExitHook(woc.execWf.Spec.Arguments)
		onExitNode, err = woc.executeTemplate(ctx, onExitNodeName, &wfv1.WorkflowStep{Template: exitHook.Template, TemplateRef: exitHook.TemplateRef}, tmplCtx, exitHook.Arguments, &executeTemplateOpts{onExitTemplate: true})
		if err != nil {
			x := fmt.Errorf("error in exit template execution : %w", err)
			switch err {
			case ErrDeadlineExceeded:
				woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowTimedOut", x.Error())
			case ErrParallelismReached:
			default:
				if !errorsutil.IsTransientErr(err) && !woc.wf.Status.Phase.Completed() && os.Getenv("BUBBLE_ENTRY_TEMPLATE_ERR") != "false" {
					woc.markWorkflowError(ctx, x)

					// Garbage collect PVCs if Onexit template execution returns error
					if err := woc.deletePVCs(ctx); err != nil {
						woc.log.WithError(err).Warn("failed to delete PVCs")
					}
				}
			}
			return
		}

		// If the onExit node (or any child of the onExit node) requires HTTP reconciliation, do it here
		if onExitNode != nil && woc.nodeRequiresTaskSetReconciliation(onExitNode.Name) {
			woc.taskSetReconciliation(ctx)
		}

		if onExitNode == nil || !onExitNode.Fulfilled() {
			return
		}
	}

	var workflowMessage string
	if node.FailedOrError() && woc.GetShutdownStrategy().Enabled() {
		workflowMessage = fmt.Sprintf("Stopped with strategy '%s'", woc.GetShutdownStrategy())
	} else {
		workflowMessage = node.Message
	}

	// If we get here, the workflow completed, all PVCs were deleted successfully, and
	// exit handlers were executed. We now need to infer the workflow phase from the
	// node phase.
	switch workflowStatus {
	case wfv1.WorkflowSucceeded:
		if onExitNode != nil && onExitNode.FailedOrError() {
			// if main workflow succeeded, but the exit node was unsuccessful
			// the workflow is now considered unsuccessful.
			switch onExitNode.Phase {
			case wfv1.NodeFailed:
				woc.markWorkflowFailed(ctx, onExitNode.Message)
			default:
				woc.markWorkflowError(ctx, fmt.Errorf(onExitNode.Message))
			}
		} else {
			woc.markWorkflowSuccess(ctx)
		}
	case wfv1.WorkflowFailed:
		woc.markWorkflowFailed(ctx, workflowMessage)
	case wfv1.WorkflowError:
		woc.markWorkflowPhase(ctx, wfv1.WorkflowError, workflowMessage)
	default:
		// NOTE: we should never make it here because if the node was 'Running' we should have
		// returned earlier.
		err = errors.InternalErrorf("Unexpected node phase %s: %+v", woc.wf.ObjectMeta.Name, err)
		woc.markWorkflowError(ctx, err)
	}

	if woc.execWf.Spec.Metrics != nil {
		woc.globalParams[common.GlobalVarWorkflowStatus] = string(workflowStatus)
		localScope, realTimeScope := woc.prepareMetricScope(node)
		woc.computeMetrics(woc.execWf.Spec.Metrics.Prometheus, localScope, realTimeScope, false)
	}

	if err := woc.deletePVCs(ctx); err != nil {
		woc.log.WithError(err).Warn("failed to delete PVCs")
	}
}

func (woc *wfOperationCtx) releaseLocksForPendingShuttingdownWfs(ctx context.Context) bool {
	if woc.GetShutdownStrategy().Enabled() && woc.wf.Status.Phase == wfv1.WorkflowPending && woc.GetShutdownStrategy() == wfv1.ShutdownStrategyTerminate {
		if woc.controller.syncManager.ReleaseAll(woc.execWf) {
			woc.log.WithFields(log.Fields{"key": woc.execWf.Name}).Info("Released all locks since this pending workflow is being shutdown")
			woc.markWorkflowSuccess(ctx)
			return true
		}
	}
	return false
}

// set Labels and Annotations for the Workflow
// Also, since we're setting Labels and Annotations we need to find any
// parameters formatted as "workflow.labels.<param>" or "workflow.annotations.<param>"
// and perform substitution
func (woc *wfOperationCtx) updateWorkflowMetadata() error {
	updatedParams := make(common.Parameters)
	if md := woc.execWf.Spec.WorkflowMetadata; md != nil {
		if woc.wf.Labels == nil {
			woc.wf.Labels = make(map[string]string)
		}
		for n, v := range md.Labels {
			if errs := validation.IsValidLabelValue(v); errs != nil {
				return errors.Errorf(errors.CodeBadRequest, "invalid label value %q for label %q: %s", v, n, strings.Join(errs, ";"))
			}
			woc.wf.Labels[n] = v
			woc.globalParams["workflow.labels."+n] = v
			updatedParams["workflow.labels."+n] = v
		}
		if woc.wf.Annotations == nil {
			woc.wf.Annotations = make(map[string]string)
		}
		for n, v := range md.Annotations {
			woc.wf.Annotations[n] = v
			woc.globalParams["workflow.annotations."+n] = v
			updatedParams["workflow.annotations."+n] = v
		}

		env := env.GetFuncMap(template.EnvMap(woc.globalParams))
		for n, f := range md.LabelsFrom {
			r, err := expr.Eval(f.Expression, env)
			if err != nil {
				return fmt.Errorf("failed to evaluate label %q expression %q: %w", n, f.Expression, err)
			}
			v, ok := r.(string)
			if !ok {
				return fmt.Errorf("failed to evaluate label %q expression %q evaluted to %T but must be a string", n, f.Expression, r)
			}
			if errs := validation.IsValidLabelValue(v); errs != nil {
				return errors.Errorf(errors.CodeBadRequest, "invalid label value %q for label %q and expression %q: %s", v, n, f.Expression, strings.Join(errs, ";"))
			}
			woc.wf.Labels[n] = v
			woc.globalParams["workflow.labels."+n] = v
			updatedParams["workflow.labels."+n] = v
		}
		woc.updated = true

		// Now we need to do any substitution that involves these labels
		err := woc.substituteGlobalVariables(updatedParams)
		if err != nil {
			return err
		}

	}
	return nil
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
func (woc *wfOperationCtx) setGlobalParameters(executionParameters wfv1.Arguments) error {
	woc.globalParams[common.GlobalVarWorkflowName] = woc.wf.ObjectMeta.Name
	woc.globalParams[common.GlobalVarWorkflowNamespace] = woc.wf.ObjectMeta.Namespace
	woc.globalParams[common.GlobalVarWorkflowMainEntrypoint] = woc.execWf.Spec.Entrypoint
	woc.globalParams[common.GlobalVarWorkflowServiceAccountName] = woc.execWf.Spec.ServiceAccountName
	woc.globalParams[common.GlobalVarWorkflowUID] = string(woc.wf.ObjectMeta.UID)
	woc.globalParams[common.GlobalVarWorkflowCreationTimestamp] = woc.wf.ObjectMeta.CreationTimestamp.Format(time.RFC3339)
	if annotation := woc.wf.ObjectMeta.GetAnnotations(); annotation != nil {
		val, ok := annotation[common.AnnotationKeyCronWfScheduledTime]
		if ok {
			woc.globalParams[common.GlobalVarWorkflowCronScheduleTime] = val
		}
	}

	if woc.execWf.Spec.Priority != nil {
		woc.globalParams[common.GlobalVarWorkflowPriority] = strconv.Itoa(int(*woc.execWf.Spec.Priority))
	}
	for char := range strftime.FormatChars {
		cTimeVar := fmt.Sprintf("%s.%s", common.GlobalVarWorkflowCreationTimestamp, string(char))
		woc.globalParams[cTimeVar] = strftime.Format("%"+string(char), woc.wf.ObjectMeta.CreationTimestamp.Time)
	}
	woc.globalParams[common.GlobalVarWorkflowCreationTimestamp+".s"] = strconv.FormatInt(woc.wf.ObjectMeta.CreationTimestamp.Time.Unix(), 10)
	woc.globalParams[common.GlobalVarWorkflowCreationTimestamp+".RFC3339"] = woc.wf.ObjectMeta.CreationTimestamp.Format(time.RFC3339)

	if workflowParameters, err := json.Marshal(woc.execWf.Spec.Arguments.Parameters); err == nil {
		woc.globalParams[common.GlobalVarWorkflowParameters] = string(workflowParameters)
		woc.globalParams[common.GlobalVarWorkflowParametersJSON] = string(workflowParameters)
	}
	for _, param := range executionParameters.Parameters {
		if param.ValueFrom != nil && param.ValueFrom.ConfigMapKeyRef != nil {
			cmValue, err := common.GetConfigMapValue(woc.controller.configMapInformer.GetIndexer(), woc.wf.ObjectMeta.Namespace, param.ValueFrom.ConfigMapKeyRef.Name, param.ValueFrom.ConfigMapKeyRef.Key)
			if err != nil {
				if param.ValueFrom.Default != nil {
					woc.globalParams["workflow.parameters."+param.Name] = param.ValueFrom.Default.String()
				} else {
					return fmt.Errorf("failed to set global parameter %s from configmap with name %s and key %s: %w",
						param.Name, param.ValueFrom.ConfigMapKeyRef.Name, param.ValueFrom.ConfigMapKeyRef.Key, err)
				}
			} else {
				woc.globalParams["workflow.parameters."+param.Name] = cmValue
			}
		} else if param.Value != nil {
			woc.globalParams["workflow.parameters."+param.Name] = param.Value.String()
		} else {
			return fmt.Errorf("either value or valueFrom must be specified in order to set global parameter %s", param.Name)
		}
	}
	if woc.wf.Status.Outputs != nil {
		for _, param := range woc.wf.Status.Outputs.Parameters {
			if param.HasValue() {
				woc.globalParams["workflow.outputs.parameters."+param.Name] = param.GetValue()
			}
		}
	}

	/////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Set global parameters based on Labels and Annotations, both those that are defined in the execWf.ObjectMeta
	// and those that are defined in the execWf.Spec.WorkflowMetadata
	// Note: we no longer set globalParams based on LabelsFrom expressions here since they may themselves use parameters
	// and thus will need to be evaluated later based on the evaluation of those parameters
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////

	md := woc.execWf.Spec.WorkflowMetadata

	if workflowAnnotations, err := json.Marshal(woc.wf.ObjectMeta.Annotations); err == nil {
		woc.globalParams[common.GlobalVarWorkflowAnnotations] = string(workflowAnnotations)
		woc.globalParams[common.GlobalVarWorkflowAnnotationsJSON] = string(workflowAnnotations)
	}
	for k, v := range woc.wf.ObjectMeta.Annotations {
		woc.globalParams["workflow.annotations."+k] = v
	}
	if workflowLabels, err := json.Marshal(woc.wf.ObjectMeta.Labels); err == nil {
		woc.globalParams[common.GlobalVarWorkflowLabels] = string(workflowLabels)
		woc.globalParams[common.GlobalVarWorkflowLabelsJSON] = string(workflowLabels)
	}
	for k, v := range woc.wf.ObjectMeta.Labels {
		// if the Label will get overridden by a LabelsFrom expression later, don't set it now
		if md != nil {
			_, existsLabelsFrom := md.LabelsFrom[k]
			if !existsLabelsFrom {
				woc.globalParams["workflow.labels."+k] = v
			}
		} else {
			woc.globalParams["workflow.labels."+k] = v
		}
	}

	if md != nil {
		for n, v := range md.Labels {
			// if the Label will get overridden by a LabelsFrom expression later, don't set it now
			_, existsLabelsFrom := md.LabelsFrom[n]
			if !existsLabelsFrom {
				woc.globalParams["workflow.labels."+n] = v
			}
		}
		for n, v := range md.Annotations {
			woc.globalParams["workflow.annotations."+n] = v
		}
	}

	return nil
}

// persistUpdates will update a workflow with any updates made during workflow operation.
// It also labels any pods as completed if we have extracted everything we need from it.
// NOTE: a previous implementation used Patch instead of Update, but Patch does not work with
// the fake CRD clientset which makes unit testing extremely difficult.
func (woc *wfOperationCtx) persistUpdates(ctx context.Context) {
	if !woc.updated {
		return
	}

	diff.LogChanges(woc.orig, woc.wf)

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
		woc.markWorkflowError(ctx, err)
	}

	// Release all acquired lock for completed workflow
	if woc.wf.Status.Synchronization != nil && woc.wf.Status.Fulfilled() {
		if woc.controller.syncManager.ReleaseAll(woc.wf) {
			woc.log.WithFields(log.Fields{"key": woc.wf.Name}).Info("Released all acquired locks")
		}
	}

	wf, err := wfClient.Update(ctx, woc.wf, metav1.UpdateOptions{})
	if err != nil {
		woc.log.Warnf("Error updating workflow: %v %s", err, apierr.ReasonForError(err))
		if argokubeerr.IsRequestEntityTooLargeErr(err) {
			woc.persistWorkflowSizeLimitErr(ctx, wfClient, err)
			return
		}
		if !apierr.IsConflict(err) {
			return
		}
		woc.log.Info("Re-applying updates on latest version and retrying update")
		wf, err := woc.reapplyUpdate(ctx, wfClient, nodes)
		if err != nil {
			woc.log.WithError(err).Info("Failed to re-apply update")
			return
		}
		woc.wf = wf
	} else {
		woc.wf = wf
		woc.controller.hydrator.HydrateWithNodes(woc.wf, nodes)
	}

	// The workflow returned from wfClient.Update doesn't have a TypeMeta associated
	// with it, so copy from the original workflow.
	woc.wf.TypeMeta = woc.orig.TypeMeta

	// Create WorkflowNode* events for nodes that have changed phase
	woc.recordNodePhaseChangeEvents(woc.orig.Status.Nodes, woc.wf.Status.Nodes)

	if !woc.controller.hydrator.IsHydrated(woc.wf) {
		panic("workflow should be hydrated")
	}

	woc.log.WithFields(log.Fields{"resourceVersion": woc.wf.ResourceVersion, "phase": woc.wf.Status.Phase}).Info("Workflow update successful")

	switch os.Getenv("INFORMER_WRITE_BACK") {
	// By default we write back (as per v2.11), this does not reduce errors, but does reduce
	// conflicts and therefore we log fewer warning messages.
	case "", "true":
		if err := woc.writeBackToInformer(); err != nil {
			woc.markWorkflowError(ctx, err)
			return
		}
	case "false":
		time.Sleep(1 * time.Second)
	}

	err = woc.removeCompletedTaskSetStatus(ctx)

	if err != nil {
		woc.log.WithError(err).Warn("error updating taskset")
	}

	if woc.wf.Status.Phase.Completed() {
		if err := woc.deleteTaskResults(ctx); err != nil {
			woc.log.WithError(err).Warn("failed to delete task-results")
		}
	}

	// It is important that we *never* label pods as completed until we successfully updated the workflow
	// Failing to do so means we can have inconsistent state.
	// Pods may be be labeled multiple times.
	woc.queuePodsForCleanup()
}

func (woc *wfOperationCtx) deleteTaskResults(ctx context.Context) error {
	deletePropagationBackground := metav1.DeletePropagationBackground
	return woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(woc.wf.Namespace).
		DeleteCollection(
			ctx,
			metav1.DeleteOptions{PropagationPolicy: &deletePropagationBackground},
			metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + woc.wf.Name},
		)
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
// See https://github.com/argoproj/argo-workflows/issues/913
func (woc *wfOperationCtx) persistWorkflowSizeLimitErr(ctx context.Context, wfClient v1alpha1.WorkflowInterface, err error) {
	woc.wf = woc.orig.DeepCopy()
	woc.markWorkflowError(ctx, err)
	_, err = wfClient.Update(ctx, woc.wf, metav1.UpdateOptions{})
	if err != nil {
		woc.log.Warnf("Error updating workflow with size error: %v", err)
	}
}

// reapplyUpdate GETs the latest version of the workflow, re-applies the updates and
// retries the UPDATE multiple times. For reasoning behind this technique, see:
// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
func (woc *wfOperationCtx) reapplyUpdate(ctx context.Context, wfClient v1alpha1.WorkflowInterface, nodes wfv1.Nodes) (*wfv1.Workflow, error) {
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
		currWf, err := wfClient.Get(ctx, woc.wf.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		// There is something about having informer indexers (introduced in v2.12) that means we are more likely to operate on the
		// previous version of the workflow. This means under high load, a previously successful workflow could
		// be operated on again. This can error (e.g. if any pod was deleted as part of clean-up). This check prevents that.
		// https://github.com/argoproj/argo-workflows/issues/4798
		if currWf.Status.Fulfilled() {
			return nil, fmt.Errorf("must never update completed workflows")
		}
		err = woc.controller.hydrator.Hydrate(currWf)
		if err != nil {
			return nil, err
		}
		for id, node := range woc.wf.Status.Nodes {
			currNode, err := currWf.Status.Nodes.Get(id)
			if (err == nil) && currNode.Fulfilled() && node.Phase != currNode.Phase {
				return nil, fmt.Errorf("must never update completed node %s", id)
			}
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
		wf, err := wfClient.Update(ctx, &newWf, metav1.UpdateOptions{})
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
func (woc *wfOperationCtx) requeueAfter(afterDuration time.Duration) {
	key, _ := cache.MetaNamespaceKeyFunc(woc.wf)
	woc.controller.wfQueue.AddAfter(key, afterDuration)
}

func (woc *wfOperationCtx) requeue() {
	key, _ := cache.MetaNamespaceKeyFunc(woc.wf)
	woc.controller.wfQueue.AddRateLimited(key)
}

// processNodeRetries updates the retry node state based on the child node state and the retry strategy and returns the node.
func (woc *wfOperationCtx) processNodeRetries(node *wfv1.NodeStatus, retryStrategy wfv1.RetryStrategy, opts *executeTemplateOpts) (*wfv1.NodeStatus, bool, error) {
	if node.Fulfilled() {
		return node, true, nil
	}

	lastChildNode := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)

	if retryStrategy.Expression != "" && len(node.Children) > 0 {
		localScope := buildRetryStrategyLocalScope(node, woc.wf.Status.Nodes)
		scope := env.GetFuncMap(localScope)
		shouldContinue, err := argoexpr.EvalBool(retryStrategy.Expression, scope)
		if err != nil {
			return nil, false, err
		}
		if !shouldContinue {
			return woc.markNodePhase(node.Name, lastChildNode.Phase, "retryStrategy.expression evaluated to false"), true, nil
		}
	}

	if lastChildNode == nil {
		return node, true, nil
	}

	if !lastChildNode.Fulfilled() {
		// last child node is still running.
		return node, true, nil
	}

	if !lastChildNode.FailedOrError() {
		node.Outputs = lastChildNode.Outputs.DeepCopy()
		woc.wf.Status.Nodes.Set(node.ID, *node)
		return woc.markNodePhase(node.Name, wfv1.NodeSucceeded), true, nil
	}

	if woc.GetShutdownStrategy().Enabled() || (woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline)) {
		var message string
		if woc.GetShutdownStrategy().Enabled() {
			message = fmt.Sprintf("Stopped with strategy '%s'", woc.GetShutdownStrategy())
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
		if time.Now().Before(waitingDeadline) && retryStrategy.Limit != nil && int32(len(node.Children)) <= int32(retryStrategy.Limit.IntValue()) {
			woc.requeueAfter(timeToWait)
			retryMessage := fmt.Sprintf("Backoff for %s", humanize.Duration(timeToWait))
			return woc.markNodePhase(node.Name, node.Phase, retryMessage), false, nil
		}

		woc.log.WithField("node", node.Name).Infof("node has maxDuration set, setting executionDeadline to: %s", humanize.Timestamp(maxDurationDeadline))
		opts.executionDeadline = maxDurationDeadline

		node = woc.markNodePhase(node.Name, node.Phase, "")
	}

	var retryOnFailed bool
	var retryOnError bool
	switch retryStrategy.RetryPolicyActual() {
	case wfv1.RetryPolicyAlways:
		retryOnFailed = true
		retryOnError = true
	case wfv1.RetryPolicyOnError:
		retryOnFailed = false
		retryOnError = true
	case wfv1.RetryPolicyOnTransientError:
		if (lastChildNode.Phase == wfv1.NodeFailed || lastChildNode.Phase == wfv1.NodeError) && errorsutil.IsTransientErr(errors.InternalError(lastChildNode.Message)) {
			retryOnFailed = true
			retryOnError = true
		}
	case wfv1.RetryPolicyOnFailure:
		retryOnFailed = true
		retryOnError = false
	default:
		return nil, false, fmt.Errorf("%s is not a valid RetryPolicy", retryStrategy.RetryPolicyActual())
	}
	woc.log.Infof("Retry Policy: %s (onFailed: %v, onError %v)", retryStrategy.RetryPolicyActual(), retryOnFailed, retryOnError)

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
func (woc *wfOperationCtx) podReconciliation(ctx context.Context) error {
	podList, err := woc.getAllWorkflowPods()
	if err != nil {
		return err
	}
	seenPods := make(map[string]*apiv1.Pod)
	seenPodLock := &sync.Mutex{}
	wfNodesLock := &sync.RWMutex{}
	podRunningCondition := wfv1.Condition{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionFalse}
	performAssessment := func(pod *apiv1.Pod) {
		if pod == nil {
			return
		}
		if woc.isAgentPod(pod) {
			woc.updateAgentPodStatus(ctx, pod)
			return
		}
		nodeID := woc.nodeID(pod)
		seenPodLock.Lock()
		seenPods[nodeID] = pod
		seenPodLock.Unlock()

		wfNodesLock.Lock()
		defer wfNodesLock.Unlock()
		node, err := woc.wf.Status.Nodes.Get(nodeID)
		if err == nil {
			if newState := woc.assessNodeStatus(pod, node); newState != nil {
				woc.addOutputsToGlobalScope(newState.Outputs)
				if newState.MemoizationStatus != nil {
					if newState.Succeeded() {
						c := woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, newState.MemoizationStatus.CacheName)
						err := c.Save(ctx, newState.MemoizationStatus.Key, newState.ID, newState.Outputs)
						if err != nil {
							woc.log.WithFields(log.Fields{"nodeID": newState.ID}).WithError(err).Error("Failed to save node outputs to cache")
							newState.Phase = wfv1.NodeError
							newState.Message = err.Error()
						}
					}
				}
				if newState.Phase == wfv1.NodeRunning {
					podRunningCondition.Status = metav1.ConditionTrue
				}
				woc.wf.Status.Nodes.Set(nodeID, *newState)
				woc.updated = true
				// warning!  when the node completes, the daemoned flag will be unset, so we must check the old node
				if !node.IsDaemoned() && !node.Completed() && newState.Completed() {
					if woc.shouldPrintPodSpec(newState) {
						printPodSpecLog(pod, woc.wf.Name)
					}
				}
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
			woc.applyExecutionControl(pod, wfNodesLock)
			<-parallelPodNum
		}(pod)
	}

	wg.Wait()

	woc.wf.Status.Conditions.UpsertCondition(podRunningCondition)

	// Now check for deleted pods. Iterate our nodes. If any one of our nodes does not show up in
	// the seen list it implies that the pod was deleted without the controller seeing the event.
	// It is now impossible to infer pod status. We can do at this point is to mark the node with Error, or
	// we can re-submit it.
	for nodeID, node := range woc.wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod || node.Phase.Fulfilled() || node.StartedAt.IsZero() {
			// node is not a pod, it is already complete, or it can be re-run.
			continue
		}
		if _, ok := seenPods[nodeID]; !ok {

			// grace-period to allow informer sync
			recentlyStarted := recentlyStarted(node)
			woc.log.WithFields(log.Fields{"nodeName": node.Name, "nodePhase": node.Phase, "recentlyStarted": recentlyStarted}).Info("Workflow pod is missing")
			metrics.PodMissingMetric.WithLabelValues(strconv.FormatBool(recentlyStarted), string(node.Phase)).Inc()

			// If the node is pending and the pod does not exist, it could be the case that we want to try to submit it
			// again instead of marking it as an error. Check if that's the case.
			if node.Pending() {
				continue
			}

			if recentlyStarted {
				// If the pod was deleted, then we it is possible that the controller never get another informer message about it.
				// In this case, the workflow will only be requeued after the resync period (20m). This means
				// workflow will not update for 20m. Requeuing here prevents that happening.
				woc.requeue()
				continue
			}

			if node.Daemoned != nil && *node.Daemoned {
				node.Daemoned = nil
				woc.updated = true
			}
			woc.markNodePhase(node.Name, wfv1.NodeError, "pod deleted")
		}
	}
	return nil
}

func (woc *wfOperationCtx) nodeID(pod *apiv1.Pod) string {
	nodeID, ok := pod.Annotations[common.AnnotationKeyNodeID]
	if !ok {
		nodeID = woc.wf.NodeID(pod.Annotations[common.AnnotationKeyNodeName])
	}
	return nodeID
}

func recentlyStarted(node wfv1.NodeStatus) bool {
	return time.Since(node.StartedAt.Time) <= envutil.LookupEnvDurationOr("RECENTLY_STARTED_POD_DURATION", 10*time.Second)
}

// shouldPrintPodSpec return eligible to print to the pod spec
func (woc *wfOperationCtx) shouldPrintPodSpec(node *wfv1.NodeStatus) bool {
	return woc.controller.Config.PodSpecLogStrategy.AllPods ||
		(woc.controller.Config.PodSpecLogStrategy.FailedPod && node.FailedOrError())
}

func (woc *wfOperationCtx) failSuspendedAndPendingNodesAfterDeadlineOrShutdown() {
	for _, node := range woc.wf.Status.Nodes {
		// fail suspended nodes when shuting down
		if woc.GetShutdownStrategy().Enabled() && node.IsActiveSuspendNode() {
			message := fmt.Sprintf("Stopped with strategy '%s'", woc.GetShutdownStrategy())
			woc.markNodePhase(node.Name, wfv1.NodeFailed, message)
			continue
		}

		// fail all pending and suspended nodes when exceeding deadline
		deadlineExceeded := woc.workflowDeadline != nil && time.Now().UTC().After(*woc.workflowDeadline)
		if deadlineExceeded && (node.Phase == wfv1.NodePending || node.IsActiveSuspendNode()) {
			message := "Step exceeded its deadline"
			woc.markNodePhase(node.Name, wfv1.NodeFailed, message)
			continue
		}
	}
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
	log := log.WithField("workflow", wfName).
		WithField("podName", pod.Name).
		WithField("nodeID", pod.Annotations[common.AnnotationKeyNodeID]).
		WithField("namespace", pod.Namespace)
	if err != nil {
		log.
			WithError(err).
			Warn("Unable to marshal pod spec.")
	} else {
		log.
			WithField("spec", string(podSpecByte)).
			Info("Pod Spec")
	}
}

// assessNodeStatus compares the current state of a pod with its corresponding node
// and returns the new node status if something changed
func (woc *wfOperationCtx) assessNodeStatus(pod *apiv1.Pod, old *wfv1.NodeStatus) *wfv1.NodeStatus {
	new := old.DeepCopy()
	tmpl, err := woc.GetNodeTemplate(old)
	if err != nil {
		woc.log.Error(err)
		return nil
	}
	switch pod.Status.Phase {
	case apiv1.PodPending:
		new.Phase = wfv1.NodePending
		new.Message = getPendingReason(pod)
		new.Daemoned = nil
	case apiv1.PodSucceeded:
		new.Phase = wfv1.NodeSucceeded
		new.Daemoned = nil
	case apiv1.PodFailed:
		// ignore pod failure for daemoned steps
		if tmpl != nil && tmpl.IsDaemon() {
			new.Phase = wfv1.NodeSucceeded
		} else {
			new.Phase, new.Message = woc.inferFailedReason(pod, tmpl)
			woc.log.WithField("displayName", old.DisplayName).WithField("templateName", old.TemplateName).
				WithField("pod", pod.Name).Infof("Pod failed: %s", new.Message)
		}
		new.Daemoned = nil
	case apiv1.PodRunning:
		// Daemons are a special case we need to understand the rules:
		// A node transitions into "daemoned" only if it's a daemon template and it becomes running AND ready.
		// A node will be unmarked "daemoned" when its boundary node completes, anywhere killDaemonedChildren is called.
		if tmpl != nil && tmpl.IsDaemon() {
			if !old.Fulfilled() {
				// pod is running and template is marked daemon. check if everything is ready
				for _, ctrStatus := range pod.Status.ContainerStatuses {
					if !ctrStatus.Ready {
						return nil
					}
				}
				// proceed to mark node as running and daemoned
				new.Phase = wfv1.NodeRunning
				new.Daemoned = pointer.BoolPtr(true)
				if !old.IsDaemoned() {
					woc.log.WithField("nodeId", old.ID).Info("Node became daemoned")
				}
			}
		} else {
			new.Phase = wfv1.NodeRunning
		}
		if tmpl != nil {
			woc.cleanUpPod(pod, *tmpl)
		}
	default:
		new.Phase = wfv1.NodeError
		new.Message = fmt.Sprintf("Unexpected pod phase for %s: %s", pod.ObjectMeta.Name, pod.Status.Phase)
	}

	// if it's ContainerSetTemplate pod then the inner container names should match to some node names,
	// in this case need to update nodes according to container status
	for _, c := range pod.Status.ContainerStatuses {
		ctrNodeName := fmt.Sprintf("%s.%s", old.Name, c.Name)
		if _, err := woc.wf.GetNodeByName(ctrNodeName); err != nil {
			continue
		}
		switch {
		case c.State.Waiting != nil:
			woc.markNodePhase(ctrNodeName, wfv1.NodePending)
		case c.State.Running != nil:
			woc.markNodePhase(ctrNodeName, wfv1.NodeRunning)
		case c.State.Terminated != nil:
			exitCode := int(c.State.Terminated.ExitCode)
			message := fmt.Sprintf("%s (exit code %d): %s", c.State.Terminated.Reason, exitCode, c.State.Terminated.Message)
			switch exitCode {
			case 0:
				woc.markNodePhase(ctrNodeName, wfv1.NodeSucceeded)
			case 64:
				// special emissary exit code indicating the emissary errors, rather than the sub-process failure,
				// (unless the sub-process coincidentally exits with code 64 of course)
				woc.markNodePhase(ctrNodeName, wfv1.NodeError, message)
			default:
				woc.markNodePhase(ctrNodeName, wfv1.NodeFailed, message)
			}
		}
	}

	// only update Pod IP for daemoned nodes to reduce number of updates
	if !new.Completed() && new.IsDaemoned() {
		new.PodIP = pod.Status.PodIP
	}

	if x, ok := pod.Annotations[common.AnnotationKeyOutputs]; ok {
		woc.log.Warn("workflow uses legacy/insecure pod patch, see https://argoproj.github.io/argo-workflows/workflow-rbac/")
		if new.Outputs == nil {
			new.Outputs = &wfv1.Outputs{}
		}
		if err := json.Unmarshal([]byte(x), new.Outputs); err != nil {
			new.Phase = wfv1.NodeError
			new.Message = err.Error()
		}
	}

	new.HostNodeName = pod.Spec.NodeName

	if !new.Progress.IsValid() {
		new.Progress = wfv1.ProgressDefault
	}

	if x, ok := pod.Annotations[common.AnnotationKeyProgress]; ok {
		woc.log.Warn("workflow uses legacy/insecure pod patch, see https://argoproj.github.io/argo-workflows/workflow-rbac/")
		if p, ok := wfv1.ParseProgress(x); ok {
			new.Progress = p
		}
	}

	// We capture the exit-code after we look for the task-result.
	// All other outputs are set by the executor, only the exit-code is set by the controller.
	// By waiting, we avoid breaking the race-condition check.
	if exitCode := getExitCode(pod); exitCode != nil {
		if new.Outputs == nil {
			new.Outputs = &wfv1.Outputs{}
		}
		new.Outputs.ExitCode = pointer.StringPtr(fmt.Sprint(*exitCode))
	}

	// If the init container failed, we should mark the node as failed.
	var initContainerFailed bool
	for _, c := range pod.Status.InitContainerStatuses {
		if c.State.Terminated != nil && int(c.State.Terminated.ExitCode) != 0 {
			new.Phase = wfv1.NodeFailed
			initContainerFailed = true
			woc.log.WithField("new.phase", new.Phase).Info("marking node as failed since init container has non-zero exit code")
			break
		}
	}

	// We cannot fail the node until the wait container is finished (unless any init container has failed) because it may be busy saving outputs, and these
	// would not get captured successfully.
	for _, c := range pod.Status.ContainerStatuses {
		if (c.Name == common.WaitContainerName && c.State.Terminated == nil && new.Phase.Completed()) && !initContainerFailed {
			woc.log.WithField("new.phase", new.Phase).Info("leaving phase un-changed: wait container is not yet terminated ")
			new.Phase = old.Phase
		}
	}

	// if we are transitioning from Pending to a different state, clear out unchanged message
	if old.Phase == wfv1.NodePending && new.Phase != wfv1.NodePending && old.Message == new.Message {
		new.Message = ""
	}

	if new.Fulfilled() && new.FinishedAt.IsZero() {
		new.FinishedAt = getLatestFinishedAt(pod)
		new.ResourcesDuration = resource.DurationForPod(pod)
	}

	if !reflect.DeepEqual(old, new) {
		woc.log.WithField("nodeID", old.ID).
			WithField("old.phase", old.Phase).
			WithField("new.phase", new.Phase).
			WithField("old.message", old.Message).
			WithField("new.message", new.Message).
			WithField("old.progress", old.Progress).
			WithField("new.progress", new.Progress).
			Info("node changed")
		return new
	}
	woc.log.WithField("nodeID", old.ID).
		Info("node unchanged")
	return nil
}

func getExitCode(pod *apiv1.Pod) *int32 {
	for _, c := range pod.Status.ContainerStatuses {
		if c.Name == common.MainContainerName && c.State.Terminated != nil {
			return pointer.Int32Ptr(c.State.Terminated.ExitCode)
		}
	}
	return nil
}

func podHasContainerNeedingTermination(pod *apiv1.Pod, tmpl wfv1.Template) bool {
	// pod needs to be terminated if any of the following are true:
	// 1. any main container has exited with non-zero exit code
	// 2. all main containers have exited
	// pod termination will cause the wait container to finish
	for _, c := range pod.Status.ContainerStatuses {
		if tmpl.IsMainContainerName(c.Name) && c.State.Terminated != nil && c.State.Terminated.ExitCode != 0 {
			return true
		}
	}
	for _, c := range pod.Status.ContainerStatuses {
		if tmpl.IsMainContainerName(c.Name) && c.State.Terminated == nil {
			return false
		}
	}
	return true
}

func (woc *wfOperationCtx) cleanUpPod(pod *apiv1.Pod, tmpl wfv1.Template) {
	if podHasContainerNeedingTermination(pod, tmpl) {
		woc.controller.queuePodForCleanup(woc.wf.Namespace, pod.Name, terminateContainers)
	}
}

func getLatestFinishedAt(pod *apiv1.Pod) metav1.Time {
	var latest metav1.Time
	for _, ctr := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		if r := ctr.State.Running; r != nil { // if we are running, then the finished at time must be now or after
			latest = metav1.Now()
		} else if t := ctr.State.Terminated; t != nil && t.FinishedAt.After(latest.Time) {
			latest = t.FinishedAt
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
func (woc *wfOperationCtx) inferFailedReason(pod *apiv1.Pod, tmpl *wfv1.Template) (wfv1.NodePhase, string) {
	if pod.Status.Message != "" {
		// Pod has a nice error message. Use that.
		return wfv1.NodeFailed, pod.Status.Message
	}

	// We only get one message to set for the overall node status.
	// If multiple containers failed, in order of preference:
	// init containers (will be appended later), main (annotated), main (exit code), wait, sidecars.
	order := func(n string) int {
		switch {
		case tmpl.IsMainContainerName(n):
			return 1
		case n == common.WaitContainerName:
			return 2
		default:
			return 3
		}
	}
	ctrs := pod.Status.ContainerStatuses
	sort.Slice(ctrs, func(i, j int) bool { return order(ctrs[i].Name) < order(ctrs[j].Name) })
	// Init containers have the highest preferences over other containers.
	ctrs = append(pod.Status.InitContainerStatuses, ctrs...)

	for _, ctr := range ctrs {

		// Virtual Kubelet environment will not set the terminate on waiting container
		// https://github.com/argoproj/argo-workflows/issues/3879
		// https://github.com/virtual-kubelet/virtual-kubelet/blob/7f2a02291530d2df14905702e6d51500dd57640a/node/sync.go#L195-L208

		if ctr.State.Waiting != nil {
			return wfv1.NodeError, fmt.Sprintf("Pod failed before %s container starts due to %s: %s", ctr.Name, ctr.State.Waiting.Reason, ctr.State.Waiting.Message)
		}
		t := ctr.State.Terminated
		if t == nil {
			// We should never get here
			log.Warnf("Pod %s phase was Failed but %s did not have terminated state", pod.Name, ctr.Name)
			continue
		}
		if t.ExitCode == 0 {
			continue
		}

		msg := fmt.Sprintf("%s (exit code %d)", t.Reason, t.ExitCode)
		if t.Message != "" {
			msg = fmt.Sprintf("%s: %s", msg, t.Message)
		}

		switch {
		case ctr.Name == common.InitContainerName:
			return wfv1.NodeError, msg
		case tmpl.IsMainContainerName(ctr.Name):
			return wfv1.NodeFailed, msg
		case ctr.Name == common.WaitContainerName:
			return wfv1.NodeError, msg
		default:
			if t.ExitCode == 137 || t.ExitCode == 143 {
				// if the sidecar was SIGKILL'd (exit code 137) assume it was because argoexec
				// forcibly killed the container, which we ignore the error for.
				// Java code 143 is a normal exit 128 + 15 https://github.com/elastic/elasticsearch/issues/31847
				log.Infof("Ignoring %d exit code of container '%s'", t.ExitCode, ctr.Name)
			} else {
				return wfv1.NodeFailed, msg
			}
		}
	}

	// If we get here, we have detected that the main/wait containers succeed but the sidecar(s)
	// were  SIGKILL'd. The executor may have had to forcefully terminate the sidecar (kill -9),
	// resulting in a 137 exit code (which we had ignored earlier). If failMessages is empty, it
	// indicates that this is the case and we return Success instead of Failure.
	return wfv1.NodeSucceeded, ""
}

func (woc *wfOperationCtx) createPVCs(ctx context.Context) error {
	if !(woc.wf.Status.Phase == wfv1.WorkflowPending || woc.wf.Status.Phase == wfv1.WorkflowRunning) {
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
		pvc, err := pvcClient.Create(ctx, &pvcTmpl, metav1.CreateOptions{})
		if err != nil && apierr.IsAlreadyExists(err) {
			woc.log.WithField("pvc", pvcTmpl.Name).Info("pvc already exists. Workflow is re-using it")
			pvc, err = pvcClient.Get(ctx, pvcTmpl.Name, metav1.GetOptions{})
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

		// continue
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

func (woc *wfOperationCtx) deletePVCs(ctx context.Context) error {
	gcStrategy := woc.execWf.Spec.GetVolumeClaimGC().GetStrategy()

	switch gcStrategy {
	case wfv1.VolumeClaimGCOnSuccess:
		if woc.wf.Status.Phase == wfv1.WorkflowError || woc.wf.Status.Phase == wfv1.WorkflowFailed {
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
		err := pvcClient.Delete(ctx, pvc.PersistentVolumeClaim.ClaimName, metav1.DeleteOptions{})
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
	if os.Getenv("ARGO_REMOVE_PVC_PROTECTION_FINALIZER") != "false" {
		for _, pvc := range woc.wf.Status.PersistentVolumeClaims {
			woc.log.WithField("claimName", pvc.PersistentVolumeClaim.ClaimName).
				Info("Removing PVC \"kubernetes.io/pvc-protection\" finalizer")
			x, err := pvcClient.Get(ctx, pvc.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			x.Finalizers = slice.RemoveString(x.Finalizers, "kubernetes.io/pvc-protection")
			_, err = pvcClient.Update(ctx, x, metav1.UpdateOptions{})
			if err != nil {
				return err
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
		panic(fmt.Sprintf("could not find node named %q, index %d in Children of node %+v", lastChildNodeName, nodeIndex, node))
	}

	return &lastChildNode
}

func getRetryNodeChildrenIds(node *wfv1.NodeStatus, nodes wfv1.Nodes) []string {
	// A fulfilled Retry node will always reflect the status of its last child node, so its individual attempts don't interest us.
	// To resume the traversal, we look at the children of the last child node and of any on exit nodes.
	var childrenIds []string
	for i := -1; i >= -len(node.Children); i-- {
		node := getChildNodeIndex(node, nodes, i)
		if node == nil {
			continue
		}
		if strings.HasSuffix(node.Name, ".onExit") {
			childrenIds = append(childrenIds, node.ID)
		} else if len(node.Children) > 0 {
			childrenIds = append(childrenIds, node.Children...)
		}
	}
	return childrenIds
}

func buildRetryStrategyLocalScope(node *wfv1.NodeStatus, nodes wfv1.Nodes) map[string]interface{} {
	localScope := make(map[string]interface{})

	// `retries` variable
	localScope[common.LocalVarRetries] = strconv.Itoa(len(node.Children) - 1)

	lastChildNode := getChildNodeIndex(node, nodes, -1)

	exitCode := "-1"
	if lastChildNode.Outputs != nil && lastChildNode.Outputs.ExitCode != nil {
		exitCode = *lastChildNode.Outputs.ExitCode
	}
	localScope[common.LocalVarRetriesLastExitCode] = exitCode
	localScope[common.LocalVarRetriesLastStatus] = string(lastChildNode.Phase)
	localScope[common.LocalVarRetriesLastDuration] = fmt.Sprint(lastChildNode.GetDuration().Seconds())
	localScope[common.LocalVarRetriesLastMessage] = lastChildNode.Message

	return localScope
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
func (woc *wfOperationCtx) executeTemplate(ctx context.Context, nodeName string, orgTmpl wfv1.TemplateReferenceHolder, tmplCtx *templateresolution.Context, args wfv1.Arguments, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	woc.log.Debugf("Evaluating node %s: template: %s, boundaryID: %s", nodeName, common.GetTemplateHolderString(orgTmpl), opts.boundaryID)

	// Set templateScope from which the template resolution starts.
	templateScope := tmplCtx.GetTemplateScope()

	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		// Will be initialized via woc.initializeNodeOrMarkError
		woc.log.Warnf("Node was nil, will be initialized as type Skipped")
	}

	woc.currentStackDepth++
	defer func() { woc.currentStackDepth-- }()

	if woc.currentStackDepth >= woc.controller.maxStackDepth && os.Getenv("DISABLE_MAX_RECURSION") != "true" {
		return woc.initializeNodeOrMarkError(node, nodeName, tmplCtx.GetTemplateScope(), orgTmpl, opts.boundaryID, ErrMaxDepthExceeded), ErrMaxDepthExceeded
	}

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
		localParams[common.LocalVarPodName] = woc.getPodName(nodeName, resolvedTmpl.Name)
	}
	if orgTmpl.IsDAGTask() {
		localParams["tasks.name"] = orgTmpl.GetName()
	}
	if orgTmpl.IsWorkflowStep() {
		localParams["steps.name"] = orgTmpl.GetName()
	}

	localParams["node.name"] = nodeName

	// Merge Template defaults to template
	err = woc.mergedTemplateDefaultsInto(resolvedTmpl)
	if err != nil {
		return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
	}

	// Inputs has been processed with arguments already, so pass empty arguments.
	processedTmpl, err := common.ProcessArgs(resolvedTmpl, &args, woc.globalParams, localParams, false, woc.wf.Namespace, woc.controller.configMapInformer.GetIndexer())
	if err != nil {
		return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
	}

	// Check if this is a fulfilled node for synchronization.
	// If so, release synchronization and return this node. No more logic will be executed.
	if node != nil {
		fulfilledNode := woc.handleNodeFulfilled(nodeName, node, processedTmpl)
		if fulfilledNode != nil {
			woc.controller.syncManager.Release(woc.wf, node.ID, processedTmpl.Synchronization)
			return fulfilledNode, nil
		}
		woc.log.Debugf("Executing node %s of %s is %s", nodeName, node.Type, node.Phase)
	}

	// Check if we took too long operating on this workflow and immediately return if we did
	if time.Now().UTC().After(woc.deadline) {
		woc.log.Warnf("Deadline exceeded")
		woc.requeue()
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

	unlockedNode := false

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
			woc.log.Infof("Could not acquire lock named: %s", lockName)
			return woc.markNodeWaitingForLock(node.Name, lockName.EncodeName())
		} else {
			woc.log.Infof("Node %s acquired synchronization lock", nodeName)
			if node != nil {
				node, err = woc.markNodeWaitingForLock(node.Name, "")
				if err != nil {
					woc.log.WithField("node.Name", node.Name).WithField("lockName", "").Error("markNodeWaitingForLock returned err")
					return nil, err
				}
			}
			// Set this value to check that this node is using synchronization, and has acquired the lock
			unlockedNode = true
		}

		woc.updated = woc.updated || wfUpdated
	}

	// Check memoization cache if the node is about to be created, or was created in the past but is only now allowed to run due to acquiring a lock
	if processedTmpl.Memoize != nil {
		if node == nil || unlockedNode {
			memoizationCache := woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, processedTmpl.Memoize.Cache.ConfigMap.Name)
			if memoizationCache == nil {
				err := fmt.Errorf("cache could not be found or created")
				woc.log.WithFields(log.Fields{"cacheName": processedTmpl.Memoize.Cache.ConfigMap.Name}).WithError(err)
				return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
			}

			entry, err := memoizationCache.Load(ctx, processedTmpl.Memoize.Key)
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
				if node == nil {
					node = woc.initializeCacheHitNode(nodeName, processedTmpl, templateScope, orgTmpl, opts.boundaryID, outputs, memoizationStatus)
				} else {
					woc.log.Infof("Node %s is using mutex with memoize. Cache is hit.", nodeName)
					woc.updateAsCacheHitNode(node, outputs, memoizationStatus)
				}
			} else {
				if node == nil {
					node = woc.initializeCacheNode(nodeName, processedTmpl, templateScope, orgTmpl, opts.boundaryID, memoizationStatus)
				} else {
					woc.log.Infof("Node %s is using mutex with memoize. Cache is NOT hit", nodeName)
					woc.updateAsCacheNode(node, memoizationStatus)
				}
			}
			woc.wf.Status.Nodes.Set(node.ID, *node)
			woc.updated = true
		}
	}

	// Check if this is a fulfilled node for memoization.
	// If so, just return this node. No more logic will be executed.
	if node != nil {
		fulfilledNode := woc.handleNodeFulfilled(nodeName, node, processedTmpl)
		if fulfilledNode != nil {
			return fulfilledNode, nil
		}
		// Memoized nodes don't have StartedAt.
		if node.StartedAt.IsZero() {
			node.StartedAt = metav1.Time{Time: time.Now().UTC()}
			node.EstimatedDuration = woc.estimateNodeDuration(node.Name)
			woc.wf.Status.Nodes.Set(node.ID, *node)
			woc.updated = true
		}
	}

	// If the user has specified retries, node becomes a special retry node.
	// This node acts as a parent of all retries that will be done for
	// the container. The status of this node should be "Success" if any
	// of the retries succeed. Otherwise, it is "Failed".
	retryNodeName := ""

	// Here it is needed to be updated
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
		lastChildNode := getChildNodeIndex(retryParentNode, woc.wf.Status.Nodes, -1)
		// The retry node might have completed by now.
		if retryParentNode.Fulfilled() {
			// If retry node has completed, set the output of the last child node to its output.
			// Runtime parameters (e.g., `status`, `resourceDuration`) in the output will be used to emit metrics.
			if lastChildNode != nil {
				retryParentNode.Outputs = lastChildNode.Outputs.DeepCopy()
				woc.wf.Status.Nodes.Set(node.ID, *retryParentNode)
			}
			if processedTmpl.Metrics != nil {
				// In this check, a completed node may or may not have existed prior to this execution. If it did exist, ensure that it wasn't
				// completed before this execution. If it did not exist prior, then we can infer that it was completed during this execution.
				// The statement "(!ok || !prevNodeStatus.Fulfilled())" checks for this behavior and represents the material conditional
				// "ok -> !prevNodeStatus.Fulfilled()" (https://en.wikipedia.org/wiki/Material_conditional)
				if prevNodeStatus, ok := woc.preExecutionNodePhases[retryParentNode.ID]; (!ok || !prevNodeStatus.Fulfilled()) && retryParentNode.Fulfilled() {
					localScope, realTimeScope := woc.prepareMetricScope(processedRetryParentNode)
					woc.computeMetrics(processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
				}
			}
			if processedTmpl.Synchronization != nil {
				woc.controller.syncManager.Release(woc.wf, node.ID, processedTmpl.Synchronization)
			}
			lastChildNode := getChildNodeIndex(retryParentNode, woc.wf.Status.Nodes, -1)
			if lastChildNode != nil {
				retryParentNode.Outputs = lastChildNode.Outputs.DeepCopy()
				woc.wf.Status.Nodes.Set(node.ID, *retryParentNode)
			}
			return retryParentNode, nil
		} else if lastChildNode != nil && lastChildNode.Fulfilled() && processedTmpl.Metrics != nil {
			// If retry node has not completed and last child node has completed, emit metrics for the last child node.
			localScope, realTimeScope := woc.prepareMetricScope(lastChildNode)
			woc.computeMetrics(processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
		}
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
				localParams[common.LocalVarPodName] = woc.getPodName(nodeName, processedTmpl.Name)
			}
			// Inject the retryAttempt number
			localParams[common.LocalVarRetries] = strconv.Itoa(len(retryParentNode.Children))

			processedTmpl, err = common.SubstituteParams(processedTmpl, map[string]string{}, localParams)
			if errorsutil.IsTransientErr(err) {
				return node, err
			}
			if err != nil {
				return woc.initializeNodeOrMarkError(node, nodeName, templateScope, orgTmpl, opts.boundaryID, err), err
			}
		}
	}

	switch processedTmpl.GetType() {
	case wfv1.TemplateTypeContainer:
		node, err = woc.executeContainer(ctx, nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeContainerSet:
		node, err = woc.executeContainerSet(ctx, nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeSteps:
		node, err = woc.executeSteps(ctx, nodeName, newTmplCtx, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeScript:
		node, err = woc.executeScript(ctx, nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeResource:
		node, err = woc.executeResource(ctx, nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeDAG:
		node, err = woc.executeDAG(ctx, nodeName, newTmplCtx, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeSuspend:
		node, err = woc.executeSuspend(nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeData:
		node, err = woc.executeData(ctx, nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypeHTTP:
		node = woc.executeHTTPTemplate(nodeName, templateScope, processedTmpl, orgTmpl, opts)
	case wfv1.TemplateTypePlugin:
		node = woc.executePluginTemplate(nodeName, templateScope, processedTmpl, orgTmpl, opts)
	default:
		err = errors.Errorf(errors.CodeBadRequest, "Template '%s' missing specification", processedTmpl.Name)
		return woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, templateScope, orgTmpl, opts.boundaryID, wfv1.NodeError, err.Error()), err
	}

	if err != nil {
		node = woc.markNodeError(nodeName, err)

		// If retry policy is not set, or if it is not set to Always or OnError, we won't attempt to retry an errored container
		// and we return instead.
		retryStrategy := woc.retryStrategy(processedTmpl)
		release := false
		if retryStrategy == nil {
			release = true
		} else {
			retryPolicy := retryStrategy.RetryPolicyActual()
			if retryPolicy != wfv1.RetryPolicyAlways &&
				retryPolicy != wfv1.RetryPolicyOnError &&
				retryPolicy != wfv1.RetryPolicyOnTransientError {
				release = true
			}
		}
		if release {
			woc.controller.syncManager.Release(woc.wf, node.ID, processedTmpl.Synchronization)
			return node, err
		}
	}

	if node.Fulfilled() {
		woc.controller.syncManager.Release(woc.wf, node.ID, processedTmpl.Synchronization)
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

	retrieveNode, err := woc.wf.GetNodeByName(node.Name)
	if err != nil {
		err := fmt.Errorf("no Node found by the name of %s;  wf.Status.Nodes=%+v", node.Name, woc.wf.Status.Nodes)
		woc.log.Error(err)
		woc.markWorkflowError(ctx, err)
		return node, err
	}
	node = retrieveNode

	// Swap the node back to retry node
	if retryNodeName != "" {
		retryNode, err := woc.wf.GetNodeByName(retryNodeName)
		if err != nil {
			err := fmt.Errorf("no Retry Node found by the name of %s;  wf.Status.Nodes=%+v", retryNodeName, woc.wf.Status.Nodes)
			woc.log.Error(err)
			woc.markWorkflowError(ctx, err)
			return node, err
		}

		if !retryNode.Fulfilled() && node.Fulfilled() { // if the retry child has completed we need to update outself

			retryNode, err = woc.executeTemplate(ctx, retryNodeName, orgTmpl, tmplCtx, args, opts)
			if err != nil {
				return woc.markNodeError(node.Name, err), err
			}
		}
		node = retryNode
	}

	return node, nil
}

func (woc *wfOperationCtx) handleNodeFulfilled(nodeName string, node *wfv1.NodeStatus, processedTmpl *wfv1.Template) *wfv1.NodeStatus {
	if node == nil || !node.Fulfilled() {
		return nil
	}

	woc.log.Debugf("Node %s already completed", nodeName)

	if processedTmpl.Metrics != nil {
		// Check if this node completed between executions. If it did, emit metrics.
		// We can infer that this node completed during the current operation, emit metrics
		if prevNodeStatus, ok := woc.preExecutionNodePhases[node.ID]; ok && !prevNodeStatus.Fulfilled() {
			localScope, realTimeScope := woc.prepareMetricScope(node)
			woc.computeMetrics(processedTmpl.Metrics.Prometheus, localScope, realTimeScope, false)
		}
	}
	return node
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
func (woc *wfOperationCtx) markWorkflowPhase(ctx context.Context, phase wfv1.WorkflowPhase, message string) {
	markCompleted := false
	if woc.wf.Status.Phase != phase {
		if woc.wf.Status.Fulfilled() {
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
		case wfv1.WorkflowRunning:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeNormal, "WorkflowRunning", "Workflow Running")
		case wfv1.WorkflowSucceeded:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeNormal, "WorkflowSucceeded", "Workflow completed")
		case wfv1.WorkflowFailed, wfv1.WorkflowError:
			woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "WorkflowFailed", message)
		}
		markCompleted = phase.Completed()
	}
	if woc.wf.Status.StartedAt.IsZero() && phase != wfv1.WorkflowPending {
		woc.updated = true
		woc.wf.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
		woc.wf.Status.EstimatedDuration = woc.estimateWorkflowDuration()
	}
	if woc.wf.Status.Message != message {
		woc.log.Infof("Updated message %s -> %s", woc.wf.Status.Message, message)
		woc.updated = true
		woc.wf.Status.Message = message
	}

	if phase == wfv1.WorkflowError {
		entryNode, err := woc.wf.Status.Nodes.Get(woc.wf.ObjectMeta.Name)
		if err != nil {
			woc.log.Errorf("was unable to obtain node for %s", woc.wf.ObjectMeta.Name)
		}
		if (err == nil) && entryNode.Phase == wfv1.NodeRunning {
			entryNode.Phase = wfv1.NodeError
			entryNode.Message = "Workflow operation error"
			woc.wf.Status.Nodes.Set(woc.wf.ObjectMeta.Name, *entryNode)
			woc.updated = true
		}
	}

	switch phase {
	case wfv1.WorkflowSucceeded, wfv1.WorkflowFailed, wfv1.WorkflowError:
		// wait for all daemon nodes to get terminated before marking workflow completed
		if markCompleted && !woc.hasDaemonNodes() {
			woc.log.Info("Marking workflow completed")
			woc.wf.Status.FinishedAt = metav1.Time{Time: time.Now().UTC()}
			woc.globalParams[common.GlobalVarWorkflowDuration] = fmt.Sprintf("%f", woc.wf.Status.FinishedAt.Sub(woc.wf.Status.StartedAt.Time).Seconds())
			if woc.wf.ObjectMeta.Labels == nil {
				woc.wf.ObjectMeta.Labels = make(map[string]string)
			}
			woc.wf.ObjectMeta.Labels[common.LabelKeyCompleted] = "true"
			woc.wf.Status.Conditions.UpsertCondition(wfv1.Condition{Status: metav1.ConditionTrue, Type: wfv1.ConditionTypeCompleted})
			err := woc.deletePDBResource(ctx)
			if err != nil {
				woc.wf.Status.Phase = wfv1.WorkflowError
				woc.wf.ObjectMeta.Labels[common.LabelKeyPhase] = string(wfv1.NodeError)
				woc.updated = true
				woc.wf.Status.Message = err.Error()
			}
			if woc.controller.wfArchive.IsEnabled() {
				if woc.controller.isArchivable(woc.wf) {
					woc.log.Info("Marking workflow as pending archiving")
					woc.wf.Labels[common.LabelKeyWorkflowArchivingStatus] = "Pending"
				} else {
					woc.log.Info("Doesn't match with archive label selector. Skipping Archive")
				}
			}
			woc.updated = true
		}
		woc.controller.queuePodForCleanup(woc.wf.Namespace, woc.getAgentPodName(), deletePod)
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

func (woc *wfOperationCtx) GetNodeTemplate(node *wfv1.NodeStatus) (*wfv1.Template, error) {
	if node.TemplateRef != nil {
		tmplCtx, err := woc.createTemplateContext(node.GetTemplateScope())
		if err != nil {
			woc.markNodeError(node.Name, err)
			return nil, err
		}
		tmpl, err := tmplCtx.GetTemplateFromRef(node.TemplateRef)
		if err != nil {
			woc.markNodeError(node.Name, err)
			return tmpl, err
		}
		return tmpl, nil
	}
	return woc.wf.GetTemplateByName(node.TemplateName), nil
}

func (woc *wfOperationCtx) markWorkflowRunning(ctx context.Context) {
	woc.markWorkflowPhase(ctx, wfv1.WorkflowRunning, "")
}

func (woc *wfOperationCtx) markWorkflowSuccess(ctx context.Context) {
	woc.markWorkflowPhase(ctx, wfv1.WorkflowSucceeded, "")
}

func (woc *wfOperationCtx) markWorkflowFailed(ctx context.Context, message string) {
	woc.markWorkflowPhase(ctx, wfv1.WorkflowFailed, message)
}

func (woc *wfOperationCtx) markWorkflowError(ctx context.Context, err error) {
	woc.markWorkflowPhase(ctx, wfv1.WorkflowError, err.Error())
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
	woc.wf.Status.Nodes.Set(node.ID, *node)
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

// Creates a node status that is or will be cached
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
	ok := woc.wf.Status.Nodes.Has(nodeID)
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

	if boundaryNode, err := woc.wf.Status.Nodes.Get(boundaryID); err == nil {
		node.DisplayName = strings.TrimPrefix(node.Name, boundaryNode.Name)
		if stepsOrDagSeparator.MatchString(node.DisplayName) {
			node.DisplayName = stepsOrDagSeparator.ReplaceAllString(node.DisplayName, "")
		}
	} else {
		woc.log.Infof("was unable to obtain node for %s, letting display name to be nodeName", boundaryID)
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
	woc.wf.Status.Nodes.Set(nodeID, node)
	woc.log.Infof("%s node %v initialized %s%s", node.Type, node.ID, node.Phase, message)
	woc.updated = true
	return &node
}

// Update a node status with cache status
func (woc *wfOperationCtx) updateAsCacheNode(node *wfv1.NodeStatus, memStat *wfv1.MemoizationStatus) {
	node.MemoizationStatus = memStat

	woc.wf.Status.Nodes.Set(node.ID, *node)
	woc.updated = true
}

// Update a node status that has been cached and marked as finished
func (woc *wfOperationCtx) updateAsCacheHitNode(node *wfv1.NodeStatus, outputs *wfv1.Outputs, memStat *wfv1.MemoizationStatus, message ...string) {
	node.Phase = wfv1.NodeSucceeded
	node.Outputs = outputs
	node.FinishedAt = metav1.Time{Time: time.Now().UTC()}

	woc.updateAsCacheNode(node, memStat)
	woc.log.Infof("%s node %v updated %s%s", node.Type, node.ID, node.Phase, message)
}

// markNodePhase marks a node with the given phase, creating the node if necessary and handles timestamps
func (woc *wfOperationCtx) markNodePhase(nodeName string, phase wfv1.NodePhase, message ...string) *wfv1.NodeStatus {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		woc.log.Warningf("workflow '%s' node '%s' uninitialized when marking as %v: %s", woc.wf.Name, nodeName, phase, message)
		node = &wfv1.NodeStatus{}
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
	woc.wf.Status.Nodes.Set(node.ID, *node)
	return node
}

func (woc *wfOperationCtx) getPodByNode(node *wfv1.NodeStatus) (*apiv1.Pod, error) {
	if node.Type != wfv1.NodeTypePod {
		return nil, fmt.Errorf("Expected node type %s, got %s", wfv1.NodeTypePod, node.Type)
	}

	podName := woc.getPodName(node.Name, node.TemplateName)
	return woc.controller.getPod(woc.wf.GetNamespace(), podName)
}

func (woc *wfOperationCtx) recordNodePhaseEvent(node *wfv1.NodeStatus) {
	message := fmt.Sprintf("%v node %s", node.Phase, node.Name)
	if node.Message != "" {
		message = message + ": " + node.Message
	}
	eventType := apiv1.EventTypeWarning
	switch node.Phase {
	case wfv1.NodeSucceeded, wfv1.NodeRunning:
		eventType = apiv1.EventTypeNormal
	}
	eventConfig := woc.controller.Config.NodeEvents
	annotations := map[string]string{
		common.AnnotationKeyNodeType: string(node.Type),
		common.AnnotationKeyNodeName: node.Name,
		common.AnnotationKeyNodeID:   node.ID,
		// For retried/resubmitted workflows, the only main differentiation is the start time of nodes.
		// We include this annotation here so that we could avoid combining events for those nodes.
		common.AnnotationKeyNodeStartTime: strconv.FormatInt(node.StartedAt.UnixNano(), 10),
	}
	var involvedObject runtime.Object = woc.wf
	if eventConfig.SendAsPod {
		pod, err := woc.getPodByNode(node)
		if err != nil {
			woc.log.WithError(err).Info("Error getting pod from workflow node")
		}
		if pod != nil {
			involvedObject = pod
			annotations[common.AnnotationKeyWorkflowName] = woc.wf.Name
			annotations[common.AnnotationKeyWorkflowUID] = string(woc.wf.GetUID())
		}
	}
	woc.eventRecorder.AnnotatedEventf(
		involvedObject,
		annotations,
		eventType,
		fmt.Sprintf("WorkflowNode%s", node.Phase),
		message,
	)
}

// recordNodePhaseChangeEvents creates WorkflowNode Kubernetes events for each node
// that has changes logged during this execution of the operator loop.
func (woc *wfOperationCtx) recordNodePhaseChangeEvents(old wfv1.Nodes, new wfv1.Nodes) {
	if !woc.controller.Config.NodeEvents.IsEnabled() {
		return
	}

	// Check for newly added nodes; send an event for new nodes
	for nodeName, newNode := range new {
		oldNode, exists := old[nodeName]
		if exists {
			if oldNode.Phase == newNode.Phase {
				continue
			}
			if oldNode.Phase == wfv1.NodePending && newNode.Completed() {
				ephemeralNode := newNode.DeepCopy()
				ephemeralNode.Phase = wfv1.NodeRunning
				woc.recordNodePhaseEvent(ephemeralNode)
			}
			woc.recordNodePhaseEvent(&newNode)
		} else {
			if newNode.Phase == wfv1.NodeRunning {
				woc.recordNodePhaseEvent(&newNode)
			} else if newNode.Completed() {
				ephemeralNode := newNode.DeepCopy()
				ephemeralNode.Phase = wfv1.NodeRunning
				woc.recordNodePhaseEvent(ephemeralNode)
				woc.recordNodePhaseEvent(&newNode)
			}
		}
	}
}

// markNodeError is a convenience method to mark a node with an error and set the message from the error
func (woc *wfOperationCtx) markNodeError(nodeName string, err error) *wfv1.NodeStatus {
	woc.log.WithError(err).WithField("nodeName", nodeName).Error("Mark error node")
	return woc.markNodePhase(nodeName, wfv1.NodeError, err.Error())
}

// markNodePending is a convenience method to mark a node and set the message from the error
func (woc *wfOperationCtx) markNodePending(nodeName string, err error) *wfv1.NodeStatus {
	woc.log.Infof("Mark node %s as Pending, due to: %v", nodeName, err)
	return woc.markNodePhase(nodeName, wfv1.NodePending, err.Error()) // this error message will not change often
}

// markNodeWaitingForLock is a convenience method to mark that a node is waiting for a lock
func (woc *wfOperationCtx) markNodeWaitingForLock(nodeName string, lockName string) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		return node, err
	}

	if node.SynchronizationStatus == nil {
		node.SynchronizationStatus = &wfv1.NodeSynchronizationStatus{}
	}

	if lockName == "" {
		// If we are no longer waiting for a lock, nil out the sync status
		node.SynchronizationStatus = nil
		node.Message = ""
	} else {
		node.SynchronizationStatus.Waiting = lockName
	}

	woc.wf.Status.Nodes.Set(node.ID, *node)
	woc.updated = true
	return node, nil
}

// checkParallelism checks if the given template is able to be executed, considering the current active pods and workflow/template parallelism
func (woc *wfOperationCtx) checkParallelism(tmpl *wfv1.Template, node *wfv1.NodeStatus, boundaryID string) error {
	if woc.execWf.Spec.Parallelism != nil && woc.activePods >= *woc.execWf.Spec.Parallelism {
		woc.log.Infof("workflow active pod spec parallelism reached %d/%d", woc.activePods, *woc.execWf.Spec.Parallelism)
		return ErrParallelismReached
	}

	// If we are a DAG or Steps template, check if we have active pods or unsuccessful children
	if node != nil && (tmpl.GetType() == wfv1.TemplateTypeDAG || tmpl.GetType() == wfv1.TemplateTypeSteps) {
		// Check failFast
		if tmpl.IsFailFast() && woc.getUnsuccessfulChildren(node.ID) > 0 {
			woc.markNodePhase(node.Name, wfv1.NodeFailed, "template has failed or errored children and failFast enabled")
			return ErrParallelismReached
		}

		// Check parallelism
		if tmpl.HasParallelism() && woc.getActivePods(node.ID) >= *tmpl.Parallelism {
			woc.log.Infof("template (node %s) active children parallelism exceeded %d", node.ID, *tmpl.Parallelism)
			return ErrParallelismReached
		}
	}

	// if we are about to execute a pod, make sure our parent hasn't reached it's limit
	if boundaryID != "" && (node == nil || (node.Phase != wfv1.NodePending && node.Phase != wfv1.NodeRunning)) {
		boundaryNode, err := woc.wf.Status.Nodes.Get(boundaryID)
		if err != nil {
			woc.log.Errorf("was unable to obtain node for %s", boundaryID)
			return errors.InternalError("boundaryNode not found")
		}
		tmplCtx, err := woc.createTemplateContext(boundaryNode.GetTemplateScope())
		if err != nil {
			return err
		}
		_, boundaryTemplate, templateStored, err := tmplCtx.ResolveTemplate(boundaryNode)
		if err != nil {
			return err
		}
		// A new template was stored during resolution, persist it
		if templateStored {
			woc.updated = true
		}

		// Check failFast
		if boundaryTemplate.IsFailFast() && woc.getUnsuccessfulChildren(boundaryID) > 0 {
			woc.markNodePhase(boundaryNode.Name, wfv1.NodeFailed, "template has failed or errored children and failFast enabled")
			return ErrParallelismReached
		}

		// Check parallelism
		if boundaryTemplate.HasParallelism() && woc.getActiveChildren(boundaryID) >= *boundaryTemplate.Parallelism {
			woc.log.Infof("template (node %s) active children parallelism exceeded %d", boundaryID, *boundaryTemplate.Parallelism)
			return ErrParallelismReached
		}
	}
	return nil
}

func (woc *wfOperationCtx) executeContainer(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}

	// Check if the output of this container is referenced elsewhere in the Workflow. If so, make sure to include it during
	// execution.
	includeScriptOutput, err := woc.includeScriptOutput(nodeName, opts.boundaryID)
	if err != nil {
		return node, err
	}

	woc.log.Debugf("Executing node %s with container template: %v\n", nodeName, tmpl.Name)
	_, err = woc.createWorkflowPod(ctx, nodeName, []apiv1.Container{*tmpl.Container}, tmpl, &createWorkflowPodOpts{
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
	node, err := woc.wf.Status.Nodes.Get(nodeID)
	if err != nil {
		woc.log.Fatalf("was unable to obtain node for %s", nodeID)
		panic(fmt.Sprintf("Expected node for %s", nodeID))
	}
	switch node.Type {
	case wfv1.NodeTypeSkipped, wfv1.NodeTypeSuspend, wfv1.NodeTypeHTTP, wfv1.NodeTypePlugin:
		return []string{node.ID}
	case wfv1.NodeTypePod:

		// Recover the template that created this pod. If we can't just let the pod be its own outbound node
		tmplCtx, err := woc.createTemplateContext(node.GetTemplateScope())
		if err != nil {
			return []string{node.ID}
		}
		_, parentTemplate, _, err := tmplCtx.ResolveTemplate(node)
		if err != nil {
			return []string{node.ID}
		}

		// If this pod does not come from a container set, its outbound node is itself
		if parentTemplate.GetType() != wfv1.TemplateTypeContainerSet {
			return []string{node.ID}
		}

		// If this pod comes from a container set, it should be treated as a container or task group
		fallthrough
	case wfv1.NodeTypeContainer, wfv1.NodeTypeTaskGroup:
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
		outbound = append(outbound, woc.getOutboundNodes(outboundNodeID)...)
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
			val, err := scope.resolveParameter(param.ValueFrom)
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
			resolvedArt, err := scope.resolveArtifact(&art)
			if err != nil {
				// If the artifact was not found and is optional, don't mark an error
				if strings.Contains(err.Error(), "Unable to resolve") && art.Optional {
					log.Warnf("Optional artifact '%s' was not found; it won't be available as an output", art.Name)
					continue
				}
				return nil, fmt.Errorf("unable to resolve outputs from scope: %s", err)
			}
			if resolvedArt == nil {
				continue
			}
			resolvedArt.Name = art.Name
			outputs.Artifacts = append(outputs.Artifacts, *resolvedArt)
		}
	}
	return &outputs, nil
}

func generateOutputResultRegex(name string, parentTmpl *wfv1.Template) (string, string) {
	referenceRegex := fmt.Sprintf(`\.%s\.outputs\.result`, name)
	expressionRegex := fmt.Sprintf(`\[['\"]%s['\"]\]\.outputs.result`, name)
	if parentTmpl.DAG != nil {
		referenceRegex = "tasks" + referenceRegex
		expressionRegex = "tasks" + expressionRegex
	} else if parentTmpl.Steps != nil {
		referenceRegex = "steps" + referenceRegex
		expressionRegex = "steps" + expressionRegex
	}
	return referenceRegex, expressionRegex
}

// hasOutputResultRef will check given template output has any reference
func hasOutputResultRef(name string, parentTmpl *wfv1.Template) bool {
	jsonValue, err := json.Marshal(parentTmpl)
	if err != nil {
		log.Warnf("Unable to marshal template %q: %v", parentTmpl, err)
	}

	// First consider usual case (e.g.: `value: "{{steps.generate.outputs.result}}"`)
	// This is most common, so should be done first.
	referenceRegex, expressionRegex := generateOutputResultRegex(name, parentTmpl)
	contains, err := regexp.MatchString(referenceRegex, string(jsonValue))
	if err != nil {
		log.Warnf("Error in regex compilation %q: %v", referenceRegex, err)
	}

	if contains {
		return true
	}

	// Next, consider expression case (e.g.: `expression: "steps['generate-random-1'].outputs.result"`)
	contains, err = regexp.MatchString(expressionRegex, string(jsonValue))
	if err != nil {
		log.Warnf("Error in regex compilation %q: %v", expressionRegex, err)
	}
	return contains
}

// getStepOrDAGTaskName will extract the node from NodeStatus Name
func getStepOrDAGTaskName(nodeName string) string {
	// Extract the task or step name by ignoring retry IDs and expanded IDs that are included in parenthesis at the end
	// of a node. Example: ".fanout1(0:1)(0)[0]" -> "fanout"

	// Opener is what opened our current parenthesis. Example: if we see a ")", our opener is a "("
	opener := ""
loop:
	for i := len(nodeName) - 1; i >= 0; i-- {
		char := string(nodeName[i])
		switch {
		case char == opener:
			// If we find the opener, we are no longer inside a parenthesis or bracket
			opener = ""
		case opener != "":
			// If the opener is not empty, then we are inside a parenthesis or bracket
			// Do nothing
		case char == ")":
			// We are going inside a parenthesis
			opener = "("
		case char == "]":
			// We are going inside a bracket
			opener = "["
		default:
			// If the current character is not a parenthesis or bracket, and we are not inside one already, we have found
			// the end of the node name.
			nodeName = nodeName[:i+1]
			break loop
		}
	}

	// If our node contains a dot, we're a child node. We're only interested in the step that called us, so return the
	// name of the node after the last dot.
	if lastDotIndex := strings.LastIndex(nodeName, "."); lastDotIndex >= 0 {
		nodeName = nodeName[lastDotIndex+1:]
	}
	return nodeName
}

func (woc *wfOperationCtx) executeScript(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
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

	mainCtr := tmpl.Script.Container
	if len(tmpl.Script.Source) == 0 {
		woc.log.Warn("'script.source' is empty, suggest change template into 'container'")
	} else {
		mainCtr.Args = append(mainCtr.Args, common.ExecutorScriptSourcePath)
	}
	_, err = woc.createWorkflowPod(ctx, nodeName, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{
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
	if errorsutil.IsTransientErr(err) || err == ErrResourceRateLimitReached {
		// Our error was most likely caused by a lack of resources.
		woc.requeue()
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
	if node.HostNodeName != "" {
		key := fmt.Sprintf("%s.hostNodeName", prefix)
		scope.addParamToScope(key, string(node.HostNodeName))
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
		woc.addArtifactToGlobalScope(art)
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
func (woc *wfOperationCtx) processAggregateNodeOutputs(scope *wfScope, prefix string, childNodes []wfv1.NodeStatus) error {
	if len(childNodes) == 0 {
		return nil
	}
	// need to sort the child node list so that the order of outputs are preserved
	sort.Sort(loopNodes(childNodes))
	paramList := make([]map[string]string, 0)
	outputParamValueLists := make(map[string][]string)
	resultsList := make([]wfv1.Item, 0)
	for _, node := range childNodes {
		if node.Outputs == nil || node.Phase != wfv1.NodeSucceeded || node.Type == wfv1.NodeTypeRetry {
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
	{
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
	paramName := fmt.Sprintf("workflow.outputs.parameters.%s", param.GlobalName)
	if param.HasValue() {
		woc.globalParams[paramName] = param.GetValue()
	}
	wfUpdated := wfutil.AddParamToGlobalScope(woc.wf, woc.log, param)
	if wfUpdated {
		woc.updated = true
	}
}

// addArtifactToGlobalScope exports any desired node outputs to the global scope
// Optionally adds to a local scope if supplied
func (woc *wfOperationCtx) addArtifactToGlobalScope(art wfv1.Artifact) {
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
	woc.updated = true
}

// addChildNode adds a nodeID as a child to a parent
// parent and child are both node names
func (woc *wfOperationCtx) addChildNode(parent string, child string) {
	parentID := woc.wf.NodeID(parent)
	childID := woc.wf.NodeID(child)
	node, err := woc.wf.Status.Nodes.Get(parentID)
	if err != nil {
		woc.log.Fatalf("was unable to obtain node for %s", parentID)
		panic(fmt.Sprintf("parent node %s not initialized", parent))
	}
	for _, nodeID := range node.Children {
		if childID == nodeID {
			// already exists
			return
		}
	}
	node.Children = append(node.Children, childID)
	woc.wf.Status.Nodes.Set(parentID, *node)
	woc.updated = true
}

// executeResource is runs a kubectl command against a manifest
func (woc *wfOperationCtx) executeResource(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)

	if err != nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	} else if !node.Pending() {
		return node, nil
	}

	tmpl = tmpl.DeepCopy()

	if tmpl.Resource.SetOwnerReference {
		obj := unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &obj)
		if err != nil {
			return node, err
		}

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
	_, err = woc.createWorkflowPod(ctx, nodeName, []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{onExitPod: opts.onExitTemplate, executionDeadline: opts.executionDeadline})
	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}

	return node, err
}

func (woc *wfOperationCtx) executeData(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	} else if !node.Pending() {
		return node, nil
	}

	dataTemplate, err := json.Marshal(tmpl.Data)
	if err != nil {
		return node, fmt.Errorf("could not marshal data in transformation: %w", err)
	}

	mainCtr := woc.newExecContainer(common.MainContainerName, tmpl)
	mainCtr.Command = []string{"argoexec", "data", string(dataTemplate)}
	_, err = woc.createWorkflowPod(ctx, nodeName, []apiv1.Container{*mainCtr}, tmpl, &createWorkflowPodOpts{onExitPod: opts.onExitTemplate, executionDeadline: opts.executionDeadline, includeScriptOutput: true})
	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}

	return node, nil
}

func (woc *wfOperationCtx) executeSuspend(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeSuspend, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
		woc.resolveInputFieldsForSuspendNode(node)
	}
	woc.log.Infof("node %s suspended", nodeName)

	// If there is either an active workflow deadline, or if this node is suspended with a duration, then the workflow
	// will need to be requeued after a certain amount of time
	var requeueTime *time.Time

	if tmpl.Suspend.Duration != "" {
		node, err := woc.wf.GetNodeByName(nodeName)
		if err != nil {
			return nil, err
		}
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
		woc.requeueAfter(time.Until(*requeueTime))
	}

	_ = woc.markNodePhase(nodeName, wfv1.NodeRunning)
	return node, nil
}

func (woc *wfOperationCtx) resolveInputFieldsForSuspendNode(node *wfv1.NodeStatus) {
	if node.Inputs == nil {
		return
	}
	parameters := node.Inputs.Parameters
	for i, parameter := range parameters {
		if parameter.Value != nil {

			value := parameter.Value.String()
			tempParameter := wfv1.Parameter{}

			if err := json.Unmarshal([]byte(value), &tempParameter); err != nil {
				woc.log.Debugf("Unable to parse input string %s to Parameter %s, %v", value, parameter.Name, err)
				continue
			}

			enum := tempParameter.Enum
			if len(enum) > 0 {
				parameters[i].Enum = enum
				if parameters[i].Default == nil {
					parameters[i].Default = wfv1.AnyStringPtr(enum[0])
				}
			}
		}
	}
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
		return 0, fmt.Errorf("unable to parse %s as a duration: %w", durationString, err)
	}
	return suspendDuration, nil
}

func processItem(tmpl template.Template, name string, index int, item wfv1.Item, obj interface{}, whenCondition string) (string, error) {
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
	var newStepStr string
	// If when is not parameterised and evaluated to false, we are not executing nor resolving artifact,
	// we allow parameter substitution to be Unresolved
	// The parameterised when will get handle by the task-expansion
	proceed, err := shouldExecute(whenCondition)
	if err == nil && !proceed {
		newStepStr, err = tmpl.Replace(replaceMap, true)
	} else {
		newStepStr, err = tmpl.Replace(replaceMap, false)
	}
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
	// Do not display parentheses in node name. Nodes are still guaranteed to be unique due to the index number
	replacer := strings.NewReplacer("(", "", ")", "")
	cleanName := replacer.Replace(fmt.Sprint(desc))
	newName := fmt.Sprintf("%s(%d:%v)", name, index, cleanName)
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
	newVolumesStr, err := template.Replace(string(volumesBytes), params, true)
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
		replacedValue, err := template.Replace(string(metricTmplBytes), localScope, false)
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
			value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "{{"), "}}"))
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
			metricValueString := metricSpec.GetValueString()

			metricValueStringJson, err := json.Marshal(metricValueString)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("unable to marshal metric to JSON for templating '%s': %s", metricSpec.Name, err))
				continue
			}

			replacedValueJson, err := template.Replace(string(metricValueStringJson), localScope, false)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("unable to substitute parameters for metric '%s': %s", metricSpec.Name, err))
				continue
			}

			var replacedStringJson string
			err = json.Unmarshal([]byte(replacedValueJson), &replacedStringJson)
			if err != nil {
				woc.reportMetricEmissionError(fmt.Sprintf("unable to unmarshal templated metric JSON '%s': %s", metricSpec.Name, err))
				continue
			}

			metricSpec.SetValueString(replacedStringJson)

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

func (woc *wfOperationCtx) createPDBResource(ctx context.Context) error {
	if woc.execWf.Spec.PodDisruptionBudget == nil {
		return nil
	}

	pdb, err := woc.controller.kubeclientset.PolicyV1().PodDisruptionBudgets(woc.wf.Namespace).Get(
		ctx,
		woc.wf.Name,
		metav1.GetOptions{},
	)
	if err != nil && !apierr.IsNotFound(err) && !apierr.IsAlreadyExists(err) {
		return err
	}
	if pdb != nil && pdb.Name != "" {
		woc.log.Info("PDB resource already exists for workflow.")
		return nil
	}

	pdbSpec := *woc.execWf.Spec.PodDisruptionBudget
	if pdbSpec.Selector == nil {
		pdbSpec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{common.LabelKeyWorkflow: woc.wf.Name},
		}
	}

	newPDB := policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:   woc.wf.Name,
			Labels: map[string]string{common.LabelKeyWorkflow: woc.wf.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: pdbSpec,
	}
	_, err = woc.controller.kubeclientset.PolicyV1().PodDisruptionBudgets(woc.wf.Namespace).Create(ctx, &newPDB, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	woc.log.Info("Created PDB resource for workflow.")
	woc.updated = true
	return nil
}

func (woc *wfOperationCtx) deletePDBResource(ctx context.Context) error {
	if woc.execWf.Spec.PodDisruptionBudget == nil {
		return nil
	}
	err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		err := woc.controller.kubeclientset.PolicyV1().PodDisruptionBudgets(woc.wf.Namespace).Delete(ctx, woc.wf.Name, metav1.DeleteOptions{})
		if apierr.IsNotFound(err) {
			return true, nil
		}
		return !errorsutil.IsTransientErr(err), err
	})
	if err != nil {
		woc.log.WithField("err", err).Error("Unable to delete PDB resource for workflow.")
		return err
	}
	woc.log.Info("Deleted PDB resource for workflow.")
	return nil
}

// Check if the output of this node is referenced elsewhere in the Workflow. If so, make sure to include it during
// execution.
func (woc *wfOperationCtx) includeScriptOutput(nodeName, boundaryID string) (bool, error) {
	if boundaryNode, err := woc.wf.Status.Nodes.Get(boundaryID); err == nil {
		tmplCtx, err := woc.createTemplateContext(boundaryNode.GetTemplateScope())
		if err != nil {
			return false, err
		}
		_, parentTemplate, templateStored, err := tmplCtx.ResolveTemplate(boundaryNode)
		if err != nil {
			return false, err
		}
		// A new template was stored during resolution, persist it
		if templateStored {
			woc.updated = true
		}

		name := getStepOrDAGTaskName(nodeName)
		return hasOutputResultRef(name, parentTemplate), nil
	} else {
		woc.log.Errorf("was unable to obtain node for %s", boundaryID)
	}
	return false, nil
}

func (woc *wfOperationCtx) fetchWorkflowSpec() (wfv1.WorkflowSpecHolder, error) {
	if woc.wf.Spec.WorkflowTemplateRef == nil { // not-woc-misuse
		return nil, fmt.Errorf("cannot fetch workflow spec without workflowTemplateRef")
	}

	var specHolder wfv1.WorkflowSpecHolder
	var err error
	// Logic for workflow refers Workflow template
	if woc.wf.Spec.WorkflowTemplateRef.ClusterScope { // not-woc-misuse
		if woc.controller.cwftmplInformer == nil {
			woc.log.WithError(err).Error("clusterWorkflowTemplate RBAC is missing")
			return nil, fmt.Errorf("cannot get resource clusterWorkflowTemplate at cluster scope")
		}
		specHolder, err = woc.controller.cwftmplInformer.Lister().Get(woc.wf.Spec.WorkflowTemplateRef.Name) // not-woc-misuse
	} else {
		specHolder, err = woc.controller.wftmplInformer.Lister().WorkflowTemplates(woc.wf.Namespace).Get(woc.wf.Spec.WorkflowTemplateRef.Name) // not-woc-misuse
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

func (woc *wfOperationCtx) setExecWorkflow(ctx context.Context) error {
	if woc.wf.Spec.WorkflowTemplateRef != nil { // not-woc-misuse
		err := woc.setStoredWfSpec()
		if err != nil {
			woc.markWorkflowError(ctx, err)
			return err
		}
		woc.execWf = &wfv1.Workflow{Spec: *woc.wf.Status.StoredWorkflowSpec.DeepCopy()}
		woc.volumes = woc.execWf.Spec.DeepCopy().Volumes
	} else if woc.controller.Config.WorkflowRestrictions.MustUseReference() {
		err := fmt.Errorf("workflows must use workflowTemplateRef to be executed when the controller is in reference mode")
		woc.markWorkflowError(ctx, err)
		return err
	} else {
		err := woc.controller.setWorkflowDefaults(woc.wf)
		if err != nil {
			woc.markWorkflowError(ctx, err)
			return err
		}
		woc.volumes = woc.wf.Spec.DeepCopy().Volumes // not-woc-misuse
	}

	// Perform one-time workflow validation
	if woc.wf.Status.Phase == wfv1.WorkflowUnknown {
		validateOpts := validate.ValidateOpts{}
		wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates(woc.wf.Namespace))
		cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(woc.controller.wfclientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())

		// Validate the execution wfSpec
		err := waitutil.Backoff(retry.DefaultRetry,
			func() (bool, error) {
				validationErr := validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, woc.wf, validateOpts)
				if validationErr != nil {
					return !errorsutil.IsTransientErr(validationErr), validationErr
				}
				return true, nil
			})
		if err != nil {
			msg := fmt.Sprintf("invalid spec: %s", err.Error())
			woc.markWorkflowFailed(ctx, msg)
			return err
		}
	}
	err := woc.setGlobalParameters(woc.execWf.Spec.Arguments)
	if err != nil {
		woc.markWorkflowFailed(ctx, fmt.Sprintf("failed to set global parameters: %s", err.Error()))
		return err
	}

	err = woc.substituteGlobalVariables(woc.globalParams)
	if err != nil {
		return err
	}
	if woc.wf.Status.Phase == wfv1.WorkflowUnknown {
		if err := woc.updateWorkflowMetadata(); err != nil {
			woc.markWorkflowError(ctx, err)
			return err
		}
	}

	// runtime value will be set after the substitution, otherwise will not be reflected from stored wf spec
	woc.setGlobalRuntimeParameters()

	return nil
}

func (woc *wfOperationCtx) setGlobalRuntimeParameters() {
	woc.globalParams[common.GlobalVarWorkflowStatus] = string(woc.wf.Status.Phase)

	// Update workflow duration variable
	if woc.wf.Status.StartedAt.IsZero() {
		woc.globalParams[common.GlobalVarWorkflowDuration] = fmt.Sprintf("%f", time.Duration(0).Seconds())
	} else {
		woc.globalParams[common.GlobalVarWorkflowDuration] = fmt.Sprintf("%f", time.Since(woc.wf.Status.StartedAt.Time).Seconds())
	}
}

func (woc *wfOperationCtx) GetShutdownStrategy() wfv1.ShutdownStrategy {
	return woc.execWf.Spec.Shutdown
}

func (woc *wfOperationCtx) ShouldSuspend() bool {
	return woc.execWf.Spec.Suspend != nil && *woc.execWf.Spec.Suspend
}

func (woc *wfOperationCtx) needsStoredWfSpecUpdate() bool {
	// woc.wf.Status.StoredWorkflowSpec.Entrypoint == "" check is mainly to support  backward compatible with 2.11.x workflow to 2.12.x
	// Need to recalculate StoredWorkflowSpec in 2.12.x format.
	// This check can be removed once all user migrated from 2.11.x to 2.12.x
	return woc.wf.Status.StoredWorkflowSpec == nil || (woc.wf.Spec.Entrypoint != "" && woc.wf.Status.StoredWorkflowSpec.Entrypoint == "") || // not-woc-misuse
		(woc.wf.Spec.Suspend != woc.wf.Status.StoredWorkflowSpec.Suspend) || // not-woc-misuse
		(woc.wf.Spec.Shutdown != woc.wf.Status.StoredWorkflowSpec.Shutdown) // not-woc-misuse
}

func (woc *wfOperationCtx) setStoredWfSpec() error {
	wfDefault := woc.controller.Config.WorkflowDefaults
	if wfDefault == nil {
		wfDefault = &wfv1.Workflow{}
	}

	workflowTemplateSpec := woc.wf.Status.StoredWorkflowSpec

	// Load the spec from WorkflowTemplate in first time.
	if woc.wf.Status.StoredWorkflowSpec == nil {
		wftHolder, err := woc.fetchWorkflowSpec()
		if err != nil {
			return err
		}
		// Join WFT and WfDefault metadata to Workflow metadata.
		wfutil.JoinWorkflowMetaData(&woc.wf.ObjectMeta, &wfDefault.ObjectMeta)
		workflowTemplateSpec = wftHolder.GetWorkflowSpec()
	}
	// Update the Entrypoint, ShutdownStrategy and Suspend
	if woc.needsStoredWfSpecUpdate() {
		// Join workflow, workflow template, and workflow default metadata to workflow spec.
		mergedWf, err := wfutil.JoinWorkflowSpec(&woc.wf.Spec, workflowTemplateSpec, &wfDefault.Spec) // not-woc-misuse
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
		mergedWf, err := wfutil.JoinWorkflowSpec(&woc.wf.Spec, wftHolder.GetWorkflowSpec(), &wfDefault.Spec) // not-woc-misuse
		if err != nil {
			return err
		}
		if mergedWf.Spec.String() != woc.wf.Status.StoredWorkflowSpec.String() {
			return fmt.Errorf("WorkflowSpec may not change during execution when the controller is set `templateReferencing: Secure`")
		}
	}
	return nil
}

func (woc *wfOperationCtx) mergedTemplateDefaultsInto(originalTmpl *wfv1.Template) error {
	if woc.execWf.Spec.TemplateDefaults != nil {
		originalTmplType := originalTmpl.GetType()

		tmplDefaultsJson, err := json.Marshal(woc.execWf.Spec.TemplateDefaults)
		if err != nil {
			return err
		}

		targetTmplJson, err := json.Marshal(originalTmpl)
		if err != nil {
			return err
		}

		resultTmpl, err := strategicpatch.StrategicMergePatch(tmplDefaultsJson, targetTmplJson, wfv1.Template{})
		if err != nil {
			return err
		}
		err = json.Unmarshal(resultTmpl, originalTmpl)
		if err != nil {
			return err
		}
		originalTmpl.SetType(originalTmplType)
	}
	return nil
}

func (woc *wfOperationCtx) substituteGlobalVariables(params common.Parameters) error {
	execWfSpec := woc.execWf.Spec

	// To Avoid the stale Global parameter value substitution to templates.
	// Updated Global parameter values will be substituted in 'executetemplate' for templates.
	execWfSpec.Templates = nil

	wfSpec, err := json.Marshal(execWfSpec)
	if err != nil {
		return err
	}

	resolveSpec, err := template.Replace(string(wfSpec), params, true)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(resolveSpec), &woc.execWf.Spec)
	if err != nil {
		return err
	}

	return nil
}

// getPodName gets the appropriate pod name for a workflow based on the
// POD_NAMES environment variable
func (woc *wfOperationCtx) getPodName(nodeName, templateName string) string {
	version := wfutil.GetWorkflowPodNameVersion(woc.wf)
	return wfutil.GeneratePodName(woc.wf.Name, nodeName, templateName, woc.wf.NodeID(nodeName), version)
}

func (woc *wfOperationCtx) getServiceAccountTokenName(ctx context.Context, name string) (string, error) {
	if name == "" {
		name = "default"
	}
	account, err := woc.controller.kubeclientset.CoreV1().ServiceAccounts(woc.wf.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return secrets.TokenNameForServiceAccount(account), nil
}

// setWfPodNamesAnnotation sets an annotation on a workflow with the pod naming
// convention version
func setWfPodNamesAnnotation(wf *wfv1.Workflow) {
	podNameVersion := wfutil.GetPodNameVersion()

	if wf.Annotations == nil {
		wf.Annotations = map[string]string{}
	}

	wf.Annotations[common.AnnotationKeyPodNameVersion] = podNameVersion.String()
}
