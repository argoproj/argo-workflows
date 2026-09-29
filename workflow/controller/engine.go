package controller

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"maps"
	"sort"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/template"
	"github.com/argoproj/argo-workflows/v4/util/variables"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// Engine is the new generic engine for executing tasks in a DAG or steps.
type Engine struct {
	woc            *wfOperationCtx
	evaluator      *dag.DAGEvaluator
	tmplCtx        *templateresolution.TemplateContext
	boundaryID     string
	nodeName       string
	tmpl           *wfv1.Template
	orgTmpl        wfv1.TemplateReferenceHolder
	onExitTemplate bool
	log            logging.Logger
	reconciler     TaskReconciler
	hooks          *hookHandler
	// expanded holds, per expanded task, the items its latest dispatch in
	// this reconcile expanded it into, so what else reads the items this
	// reconcile sees the list that dispatch drove.
	expanded map[string][]dag.Task
	// finished holds, per finished task, what the task adds to its
	// dependants' scopes, built on first use: a finished node does not
	// change within a reconcile, and rebuilding its part for every dependant
	// makes a chain of n tasks cost n² template resolutions (C67).
	finished map[string]*variables.Scope
}

// NewEngine creates a new Engine.
func NewEngine(woc *wfOperationCtx, nodeName string, tmplCtx *templateresolution.TemplateContext, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, boundaryID string, onExitTemplate bool) *Engine {
	return &Engine{
		woc:            woc,
		nodeName:       nodeName,
		tmplCtx:        tmplCtx,
		tmpl:           tmpl,
		orgTmpl:        orgTmpl,
		boundaryID:     boundaryID,
		onExitTemplate: onExitTemplate,
		log:            woc.log,
		reconciler:     NewK8sTaskReconciler(woc, tmplCtx, nodeName),
		hooks:          newHookHandler(woc, tmplCtx, boundaryID, tmpl, woc.log),
		expanded:       make(map[string][]dag.Task),
	}
}

// Execute reconciles a DAG or Steps template in one walk over its tasks in
// dependency order, as main's executeDAG and executeSteps did: the tasks
// come ordered (DAG: dag.PullOrder from the targets, which leaves out tasks
// no target needs; Steps: as written). Each task is evaluated immediately
// before the walk acts on it, so it sees what the walk has just done to its
// dependencies: an instant completion (a when-false skip, a memoize hit, a
// nested template that finished), an Omitted node, a StepGroup closed at the
// group boundary, or an exit hook it must wait for (#12192). Each task is
// visited once per reconcile, so it is dispatched at most once and its exit
// handler is driven at most once (#14392).
//
// After the walk, externally completed tasks are settled, the StepGroups and
// the boundary are assessed, and the boundary is finalized. Errors are
// handled internally by marking the boundary node with the appropriate
// phase (Failed for Steps, Error for DAGs).
func (e *Engine) Execute(ctx context.Context, tasks []dag.Task) {
	e.evaluator = dag.NewDAGEvaluatorFromTasks(e.woc.wf, tasks, e.tmpl, e.boundaryID, e.nodeName)

	// Provide retry strategies to the evaluator
	e.populateRetryStrategies(ctx, tasks)

	e.reconcileDaemonedTasks(ctx, tasks)

	dispatched := make(map[string]bool)
	exitHooksDone, dispatching, group := true, true, 0
	for _, task := range tasks {
		name := task.GetName()
		// A step group is closed before the next one starts, so a step sees
		// the recorded phase of the group before it ({{steps.X.status}} of an
		// expanded step).
		for i, ok := stepGroupIndexOf(name); ok && group < i; group++ {
			e.assessStepGroup(ctx, group)
		}
		ran, stop := e.visit(ctx, task, e.evaluator.Evaluate(ctx, name), dispatching)
		dispatched[name] = ran
		dispatching = dispatching && !stop
		// The task's hooks are driven before any dependant is evaluated, so a
		// dependant waits for a pending exit hook in this same walk.
		exitHooksDone = e.processHooks(ctx, task) && exitHooksDone
	}

	results := e.settle(ctx, tasks, dispatched)
	e.assessStepGroups(ctx)
	if err := e.finalize(ctx, tasks, results, exitHooksDone); err != nil {
		e.markBoundaryError(ctx, err)
	}
}

// visit acts on one task's evaluation: it records an evaluation error on the
// task's node, creates the Omitted node of a task that can never run, or
// dispatches a task the evaluator found runnable, unless dispatching has
// stopped at the operation deadline. ran reports a dispatch; stop is
// dispatchOutcome's.
func (e *Engine) visit(ctx context.Context, task dag.Task, result dag.EvaluationResult, dispatching bool) (ran, stop bool) {
	name := task.GetName()
	e.logEvaluation(ctx, result)
	if result.RequeueAfter > 0 {
		e.woc.requeueAfter(result.RequeueAfter)
	}
	node := e.getTaskNode(ctx, name)
	switch {
	case result.Error != nil:
		// The evaluator could not assess this task (e.g. its depends
		// expression failed to evaluate). Record that as a terminal Error
		// node so the boundary can assess it; left unrecorded, the task
		// would stay Pending and the boundary would never complete.
		if node == nil || !node.Fulfilled() {
			e.initTerminalErrorNode(ctx, task, e.parentNodeNames(ctx, name), result.Error)
		}
		return false, false
	case result.Skipped && !result.ShouldRun:
		// It can never run: record its Omitted node, so its dependants (later
		// in the walk) and the boundary's assessment see it.
		if node == nil {
			reason := result.SkipReason
			if reason == "" {
				reason = "depends condition not met"
			}
			e.initTaskNode(ctx, task, e.parentNodeNames(ctx, name), wfv1.NodeTypeSkipped, wfv1.NodeOmitted, "omitted: "+reason)
		}
		return false, false
	case !dispatching || !e.needsDispatch(node, result):
		return false, false
	}
	_, err := e.executeTask(ctx, task)
	return true, e.dispatchOutcome(ctx, name, err)
}

// needsDispatch reports whether the evaluator's result asks for the task to
// be dispatched. Execute, Succeed and Fail all do: for Succeed and Fail the
// operator's retry handling records the outcome. So does a Running retry
// node with a daemoned child, so that processNodeRetries propagates the
// Daemoned flag from the child to the retry node.
func (e *Engine) needsDispatch(node *wfv1.NodeStatus, result dag.EvaluationResult) bool {
	switch result.Action {
	case dag.ActionExecute, dag.ActionSucceed, dag.ActionFail:
		return true
	case dag.ActionNone:
		if node != nil && node.Type == wfv1.NodeTypeRetry && node.Phase == wfv1.NodeRunning {
			return true
		}
	}
	return result.ShouldRun
}

// markBoundaryError marks the boundary node with an appropriate error phase.
// For Steps templates, uses Failed (not Error) to match legacy behavior; the
// same convention is applied by finalize when it derives the phase from the
// children. A boundary that is already fulfilled is left alone: terminal
// phases have no valid transitions.
func (e *Engine) markBoundaryError(ctx context.Context, err error) {
	node, _ := e.woc.wf.GetNodeByName(e.nodeName)
	if node != nil && node.Fulfilled() {
		return
	}
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		e.woc.markNodePhase(ctx, e.nodeName, wfv1.NodeFailed, err.Error())
	} else {
		e.woc.markNodeError(ctx, e.nodeName, err)
	}
}

// populateRetryStrategies resolves templates for all tasks and registers
// any retry strategies with the evaluator.
func (e *Engine) populateRetryStrategies(ctx context.Context, tasks []dag.Task) {
	for _, task := range tasks {
		_, resolvedTmpl, _, err := e.tmplCtx.ResolveTemplate(ctx, task.GetTemplateReferenceHolder())
		if err != nil {
			continue
		}
		rs := e.woc.retryStrategy(resolvedTmpl)
		if rs != nil {
			e.evaluator.SetRetryStrategy(task.GetName(), rs)
			e.evaluator.SetRetryDecider(task.GetName(), e.woc.shouldRetryNode)
		}
	}
}

// shouldRetryNode is the Engine's dag.RetryDecider: the retry policy and
// retryStrategy.expression checks are the ones processNodeRetries applies
// when it actually drives the retry, so the evaluator's assessment and the
// operator's decision cannot disagree.
func (woc *wfOperationCtx) shouldRetryNode(ctx context.Context, retryNode, lastChild *wfv1.NodeStatus, rs *wfv1.RetryStrategy) bool {
	retryOnFailed, retryOnError, err := retryPolicyAllows(ctx, lastChild, *rs)
	if err != nil {
		return false
	}
	if (lastChild.Phase == wfv1.NodeFailed && !retryOnFailed) || (lastChild.Phase == wfv1.NodeError && !retryOnError) {
		return false
	}
	if rs.Expression != "" && len(retryNode.Children) > 0 {
		allowed, err := woc.retryExpressionAllows(retryNode, *rs)
		if err != nil {
			return false
		}
		return allowed
	}
	return true
}

// reconcileDaemonedTasks re-executes any tasks whose pods are running as daemons.
func (e *Engine) reconcileDaemonedTasks(ctx context.Context, tasks []dag.Task) {
	for _, task := range tasks {
		taskNode := e.getTaskNode(ctx, task.GetName())
		if taskNode == nil || !taskNode.IsDaemoned() {
			continue
		}

		// If this is a retry node whose daemon child has exited (no longer daemoned),
		// clear the stale Daemoned flag so executeTask doesn't treat it as "fulfilled"
		// and skip the retry logic.
		if taskNode.Type == wfv1.NodeTypeRetry {
			_, lastChild := getChildNodeIdsAndLastRetriedNode(taskNode, e.woc.wf.Status.Nodes)
			if lastChild != nil && !lastChild.IsDaemoned() {
				taskNode.Daemoned = nil
				e.woc.wf.Status.Nodes.Set(ctx, taskNode.ID, *taskNode)
				e.woc.updated = true
				continue // Skip executeTask — the walk will pick it up now
			}
		}

		e.log.Info(ctx, fmt.Sprintf("reconciling daemoned task %s", task.GetName()))
		if _, err := e.executeTask(ctx, task); err != nil {
			e.log.WithError(err).Error(ctx, "failed to reconcile daemoned task")
		}
	}
}

// settle brings the evaluation up to date after the walk: tasks that
// completed outside it (e.g. a pod the pod controller marked Succeeded) are
// re-reconciled for metrics and lock release, then everything is evaluated
// again for StepGroup and boundary assessment.
func (e *Engine) settle(ctx context.Context, tasks []dag.Task, dispatched map[string]bool) map[string]dag.EvaluationResult {
	e.reconcileExternalCompletions(ctx, tasks, dispatched)
	return e.evaluator.EvaluateAll(ctx)
}

// processHooks runs a task's lifecycle hooks and exit handlers and reports
// whether its exit handlers have completed. A hook error is isolated to the
// failing task node (not the boundary), mirroring the legacy controller's
// executeDAGTask behavior — a single bad hook on one task must not abort
// sibling tasks or the DAG/Steps boundary.
func (e *Engine) processHooks(ctx context.Context, task dag.Task) bool {
	// ProcessAllTaskHooks isolates per-task errors on the task (via the
	// callback below) and never returns one.
	done, _ := e.hooks.ProcessAllTaskHooks(ctx, []dag.Task{task},
		e.getTaskNode,
		e.buildLocalScopeFromTask,
		func(ctx context.Context, taskNode *wfv1.NodeStatus, err error) {
			// Mark the offending task node Errored, not the boundary. Siblings
			// continue to be processed in the same operate cycle.
			e.woc.markNodeError(ctx, taskNode.Name, err)
			// If the task node is already fulfilled (e.g. an exit hook errored
			// after the task Succeeded), the phase state machine refuses the
			// Error mark and the failure would be recorded nowhere: finalize
			// gates on onExitCompleted and the boundary stays Running forever
			// (#14031). Surface the error on the boundary instead.
			if n, getErr := e.woc.wf.GetNodeByName(taskNode.Name); getErr == nil && n.Fulfilled() && n.Phase != wfv1.NodeError {
				e.woc.markNodeError(ctx, e.nodeName, err)
			}
		},
	)
	return done
}

// assessStepGroups transitions StepGroup nodes to a terminal phase once all their
// step tasks have completed. Unlike dag.TaskGroupPhase, this handles per-step
// continueOn semantics — each step in a group can have its own continueOn setting.
// Step child nodes are looked up by constructing names from the template definition
// (not from node.Children) because not all steps may have nodes yet if parallelism
// limits prevented scheduling.
func (e *Engine) assessStepGroups(ctx context.Context) {
	if e.tmpl.GetType() != wfv1.TemplateTypeSteps {
		return
	}
	for i := range e.tmpl.Steps {
		e.assessStepGroup(ctx, i)
	}
}

// assessStepGroup records group i's phase once every step in it has finished.
func (e *Engine) assessStepGroup(ctx context.Context, i int) {
	stepGroup := e.tmpl.Steps[i]
	sgNodeName := e.stepGroupNodeNameAt(i)
	sgNode, err := e.woc.wf.GetNodeByName(sgNodeName)
	if err != nil || sgNode.Fulfilled() {
		return
	}

	isPending := false
	isRunning := false
	isFailed := false
	isSucceeded := true
	allOmitted := len(stepGroup.Steps) > 0
	// Track first failing child ID to surface in the StepGroup's failure
	// message, matching pre-refactor executeStepGroup semantics. The message
	// `child '<id>' failed` bubbles up through the Steps node to the workflow
	// status and is what callers / tests (e.g. TestNodeSuspendResume) inspect
	// to identify which leaf failed.
	failingChildID := ""

	for _, step := range stepGroup.Steps {
		childNodeName := e.taskNodeName(stepTaskNameFor(i, step.Name))
		childNode, err := e.woc.wf.GetNodeByName(childNodeName)
		if err != nil {
			isPending = true
			isSucceeded = false
			allOmitted = false
			continue
		}
		if childNode.Phase != wfv1.NodeOmitted {
			allOmitted = false
		}

		switch childNode.Phase {
		case wfv1.NodeFailed:
			if step.ContinueOn == nil || !step.ContinueOn.Failed {
				isFailed = true
				isSucceeded = false
				if failingChildID == "" {
					failingChildID = childNode.ID
				}
			}
		case wfv1.NodeError:
			if step.ContinueOn == nil || !step.ContinueOn.Error {
				isFailed = true
				isSucceeded = false
				if failingChildID == "" {
					failingChildID = childNode.ID
				}
			}
		case wfv1.NodePending:
			isPending = true
			isSucceeded = false
		case wfv1.NodeRunning:
			isRunning = true
			isSucceeded = false
		case wfv1.NodeSucceeded, wfv1.NodeSkipped, wfv1.NodeOmitted:
			// Succeeded or equivalent
		default:
			isSucceeded = false
		}
	}

	// Default to Running; the StepGroup only leaves Running once every step
	// has finished, as executeStepGroup did: a failed step does not fail
	// the group while a sibling is still running. Marking it early would
	// let linkStepGroups hang the next group off a step still in flight.
	newPhase := wfv1.NodeRunning
	var newMessage string
	if isPending || isRunning {
		return
	}
	if isFailed {
		// Always use Failed for FailedOrError children, matching old executeStepGroup behavior.
		newPhase = wfv1.NodeFailed
		if failingChildID != "" {
			newMessage = fmt.Sprintf("child '%s' failed", failingChildID)
		}
	} else if isSucceeded {
		newPhase = wfv1.NodeSucceeded
		if allOmitted {
			// A group whose every step was omitted because an earlier group
			// failed never ran, so it is Omitted rather than Succeeded. Retry
			// and memoized resubmit rely on this: they refuse to reset a
			// failed node with a Succeeded descendant, and this group hangs
			// off the failed group's outbound nodes.
			newPhase = wfv1.NodeOmitted
		}
	}

	if sgNode.Phase != newPhase {
		e.woc.markNodePhase(ctx, sgNodeName, newPhase, newMessage)
	}
}

// isThrottleErr reports whether err is a deliberate throttling signal from
// the reconciler. These are not real failures — the caller should hold the
// work back for now but must not treat the situation as fatal.
func isThrottleErr(err error) bool {
	return stderrors.Is(err, ErrParallelismReached) ||
		stderrors.Is(err, ErrResourceRateLimitReached) ||
		stderrors.Is(err, ErrDeadlineExceeded) ||
		stderrors.Is(err, ErrTimeout)
}

// dispatchOutcome applies the per-task dispatch error policy, main's, and
// reports whether to stop dispatching for the rest of the walk: only the
// operate deadline does. A task held back by parallelism or a rate limit waits for a free
// slot while the others, which may be the ones to free it, are still
// dispatched; a missing reference requeues the task. Any other error is the
// task's own outcome, already recorded as an Error on its node (by
// initTerminalErrorNode or the reconciler's recordTaskError), and rolls up
// through phase assessment while its siblings keep running.
func (e *Engine) dispatchOutcome(ctx context.Context, taskName string, err error) (stop bool) {
	switch {
	case err == nil, stderrors.Is(err, ErrRequeue),
		stderrors.Is(err, ErrParallelismReached), stderrors.Is(err, ErrResourceRateLimitReached):
		return false
	case stderrors.Is(err, ErrDeadlineExceeded):
		return true
	}
	e.log.WithError(err).WithField("task", taskName).Warn(ctx, "task dispatch failed; continuing to allow sibling tasks")
	return false
}

// logEvaluation records the evaluator's diagnostics for a task at debug
// level, so "why has this task not started" can be answered from the logs.
func (e *Engine) logEvaluation(ctx context.Context, result dag.EvaluationResult) {
	if result.ActionReason == "" && !result.Suspended && !result.Skipped {
		return
	}
	e.log.WithFields(logging.Fields{
		"task":       result.TaskName,
		"action":     result.Action,
		"reason":     result.ActionReason,
		"shouldRun":  result.ShouldRun,
		"waiting":    result.Suspended,
		"waitingOn":  result.WaitingOn,
		"skipped":    result.Skipped,
		"skipReason": result.SkipReason,
	}).Debug(ctx, "task evaluation")
}

// expansionScope is the string scope items are substituted against: the
// workflow globals under the task's own scope, as expandTask and expandStep
// passed before the Engine. Globals are needed here because an expression tag
// that mixes {{item}} with {{workflow.parameters.x}} can only be evaluated
// once the item is known (#14718); simple global tags were substituted at
// operate start.
func (e *Engine) expansionScope(scope *wfScope) map[string]string {
	params := make(map[string]string)
	maps.Copy(params, e.woc.globalParams())
	maps.Copy(params, scope.getParameters())
	return params
}

// reconcileTaskGroup drives the items of an expanded task, as executeDAGTask
// and executeStepGroup did on every reconcile until the group finished: each
// item is created if it has no node yet (the rest of a fan-out held back by
// parallelism or the operation deadline) or re-entered if it has not finished
// (a deleted pod, a suspend with a duration, a nested template, a lock
// waiter), one item's error staying with that item. The group is then
// completed from its items once every one exists and has finished, with its
// exit hooks. items are the resolved task's expansion. Only the operation
// deadline stops the items early; its error is returned.
func (e *Engine) reconcileTaskGroup(ctx context.Context, tgNode *wfv1.NodeStatus, items []dag.Task) error {
	var stopErr error
	for _, item := range items {
		err := e.reconcileTask(ctx, item, []string{tgNode.Name})
		if e.dispatchOutcome(ctx, item.GetName(), err) {
			stopErr = err
			break
		}
	}
	itemNodes := make([]*wfv1.NodeStatus, len(items))
	for i, item := range items {
		if n := e.getTaskNode(ctx, item.GetName()); n != nil && !e.hasPendingHooks(n) {
			itemNodes[i] = n
		}
	}
	if phase, done := dag.TaskGroupPhase(itemNodes); done {
		e.woc.markNodePhase(ctx, tgNode.Name, phase)
	}
	return stopErr
}

// reconcileTask hands one resolved task (see resolveTask) to the
// reconciler, linking a node it creates under parents.
func (e *Engine) reconcileTask(ctx context.Context, task dag.Task, parents []string) error {
	desired, err := e.createDesiredTask(ctx, task, parents)
	if err != nil || desired == nil {
		return err
	}
	return e.reconciler.Reconcile(ctx, []DesiredTask{*desired})
}

// reconcileExpandedChildren reconciles fulfilled children of a TaskGroup
// (withItems/withParam/withSequence). The parent task can't be reconciled directly
// because its arguments contain unresolved {{item.*}} tags. Instead, we walk each
// child and run it through the reconciler so postExecutionHandling fires (which
// releases sync locks and emits metrics), as the pre-Engine controller did by
// calling executeTemplate for every expanded child on every cycle.
func (e *Engine) reconcileExpandedChildren(ctx context.Context, task dag.Task) {
	taskNode := e.getTaskNode(ctx, task.GetName())
	if taskNode == nil || taskNode.Type != wfv1.NodeTypeTaskGroup {
		return
	}
	newTmplCtx, resolvedTmpl, _, err := e.tmplCtx.ResolveTemplate(ctx, task.GetTemplateReferenceHolder())
	if err != nil {
		// The items already ran; their outcome stands. Before the Engine this
		// marked the fulfilled node Error (#13548), which was a side effect of
		// avoiding a nil dereference rather than a decision, so only log it:
		// the children's post-execution handling (sync release, metrics) is
		// skipped this cycle.
		e.log.WithFields(logging.Fields{"task": task.GetName()}).WithError(err).Warn(ctx, "failed to resolve template for completed task group; skipping its children")
		return
	}

	for _, childID := range taskNode.Children {
		child, err := e.woc.wf.Status.Nodes.Get(childID)
		if err != nil || !child.Fulfilled() {
			continue
		}
		if err := e.reconcileFulfilledNode(ctx, task, resolvedTmpl, newTmplCtx, child.Name); err != nil {
			e.log.WithFields(logging.Fields{"child": child.Name}).WithError(err).Warn(ctx, "failed to reconcile expanded child")
		}
	}
}

// reconcileFulfilledNode re-runs an already-fulfilled node through the reconciler
// so postExecutionHandling fires (sync lock release, metric emission). Local params
// are built with global params so {{workflow.uid}} etc. in synchronization configs
// resolve.
func (e *Engine) reconcileFulfilledNode(ctx context.Context, task dag.Task, resolvedTmpl *wfv1.Template, newTmplCtx *templateresolution.TemplateContext, nodeName string) error {
	localParams := make(common.Parameters)
	localParams["node.name"] = nodeName
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		localParams["steps.name"] = task.GetDisplayName()
	} else {
		localParams["tasks.name"] = task.GetDisplayName()
	}
	args := task.GetArguments()
	processedTmpl, err := common.ProcessArgs(ctx, resolvedTmpl, &args, e.woc.globalParams(), localParams, false, e.woc.wf.Namespace, e.woc.controller.typedConfigMapInformer.GetIndexer())
	if err != nil {
		return err
	}
	return e.reconciler.Reconcile(ctx, []DesiredTask{{
		TaskName:      nodeName,
		TemplateScope: e.tmplCtx.GetTemplateScope(),
		TmplCtx:       newTmplCtx,
		Template:      processedTmpl,
		TemplateRef:   task.GetTemplateReferenceHolder(),
		BoundaryID:    e.boundaryID,
		IsOnExit:      e.onExitTemplate,
	}})
}

// reconcileExternalCompletions handles tasks that completed between operate cycles
// (e.g. pod controller marked a node Succeeded). We re-reconcile them so that
// handleNodeFulfilled emits metrics and releases synchronization locks.
func (e *Engine) reconcileExternalCompletions(ctx context.Context, tasks []dag.Task, dispatched map[string]bool) {
	for _, task := range tasks {
		// For expanded tasks (withItems/withParam/withSequence), we can't reconcile
		// the parent (unresolved {{item.*}} in arguments). Instead, reconcile each
		// fulfilled child individually to release sync locks and emit metrics.
		if dag.HasExpansion(task) {
			e.reconcileExpandedChildren(ctx, task)
			continue
		}
		// Skip tasks the walk dispatched to avoid double metric emission
		// and redundant reconciliation.
		if dispatched[task.GetName()] {
			continue
		}
		taskNode := e.getTaskNode(ctx, task.GetName())
		if taskNode == nil || !taskNode.Fulfilled() {
			continue
		}
		if prev, ok := e.woc.preExecutionNodeStatuses[taskNode.ID]; ok && prev.Fulfilled() {
			continue
		}
		newTmplCtx, resolvedTmpl, _, err := e.tmplCtx.ResolveTemplate(ctx, task.GetTemplateReferenceHolder())
		if err != nil {
			e.log.WithFields(logging.Fields{"task": task.GetName()}).WithError(err).Warn(ctx, "failed to resolve template for completed task")
			continue
		}
		// See note in createDesiredTask: merge templateDefaults so per-task metric
		// emission in postExecutionHandling fires (otherwise resolvedTmpl.Metrics is nil).
		if err = e.woc.mergedTemplateDefaultsInto(resolvedTmpl); err != nil {
			e.log.WithFields(logging.Fields{"task": task.GetName()}).WithError(err).Warn(ctx, "failed to merge template defaults for completed task")
			continue
		}
		if err := e.reconcileFulfilledNode(ctx, task, resolvedTmpl, newTmplCtx, e.taskNodeName(task.GetName())); err != nil {
			e.log.WithFields(logging.Fields{"task": task.GetName()}).WithError(err).Warn(ctx, "failed to reconcile completed task (metrics/locks may be missed)")
		}
	}
}

// finalize assesses the overall phase and, if terminal, sets outputs,
// saves memoization cache, and marks the node Succeeded/Failed/Error.
func (e *Engine) finalize(ctx context.Context, tasks []dag.Task, results map[string]dag.EvaluationResult, onExitCompleted bool) error {
	targetTasks := e.evaluator.GetTargetTasks(ctx)

	// Under a Stop shutdown the boundary is failed once its exit handlers are
	// done — unless this boundary IS an onExit handler, which must be allowed
	// to complete (#16488).
	dagPhase := e.assessDAGPhase(ctx, tasks, results, e.woc.GetShutdownStrategy().Enabled() && onExitCompleted && !e.onExitTemplate)

	switch dagPhase {
	case wfv1.NodeRunning:
		return nil
	case wfv1.NodeError, wfv1.NodeFailed:
		// Wait for any in-flight (non-fulfilled) hook child nodes before
		// transitioning the boundary terminal. Errored hooks ARE fulfilled
		// and don't block — per-task hook isolation: a single failed hook
		// must not abort siblings. Only Running/Pending hook
		// nodes gate the boundary. This is required because markWorkflowFailed
		// sets the `completed=true` label, after which the controller's
		// reconciliationNeeded filter (controller.go) skips future workqueue
		// events for the workflow — including the pod-completion events that
		// would otherwise advance the hook nodes.
		if e.hasPendingTaskHooks(ctx, tasks) {
			return nil
		}
		if err := e.updateOutboundNodesForTargetTasks(ctx, targetTasks); err != nil {
			return err
		}
		// For Steps templates, always use Failed (matching old executeSteps behavior).
		// DAG templates preserve the exact phase (Error vs Failed).
		phase := dagPhase
		if e.tmpl.GetType() == wfv1.TemplateTypeSteps && dagPhase == wfv1.NodeError {
			phase = wfv1.NodeFailed
		}
		// Surface a "child '<id>' failed" message on the boundary, matching the
		// pre-refactor executeSteps/executeDAG semantics. This message bubbles up
		// to the workflow status (operator.go uses entry node.Message for
		// workflow.status.Message), and callers / tests rely on it to identify
		// which child triggered the failure (e.g. TestNodeSuspendResume).
		_ = e.woc.markNodePhase(ctx, e.nodeName, phase, e.boundaryFailureMessage(ctx))
		return nil
	}

	if !onExitCompleted {
		return nil
	}

	if err := e.setDAGOutputs(ctx); err != nil {
		return err
	}

	// Save memoization cache inline (before marking Succeeded) so that
	// sibling tasks in the parent template can hit the cache in the same
	// reconcile cycle.
	if err := e.saveMemoizationCache(ctx); err != nil {
		return err
	}

	if err := e.updateOutboundNodesForTargetTasks(ctx, targetTasks); err != nil {
		return err
	}
	_ = e.woc.markNodePhase(ctx, e.nodeName, wfv1.NodeSucceeded)
	return nil
}

// boundaryFailureMessage returns the failure message to propagate to the
// boundary node when it is marked Failed/Error:
//   - Steps: the message of the last failed StepGroup in declaration order,
//     which is the group that stopped execution (executeSteps marked the
//     boundary with that group's message).
//   - DAG: "child '<task-id>' failed" naming the first failed task in
//     declaration order. Task nodes are named, never their retry attempts.
//
// Walking the template rather than wf.Status.Nodes keeps the message stable
// between operate cycles; the map order would make it vary. Returns "" if
// no failing task or group is found.
func (e *Engine) boundaryFailureMessage(ctx context.Context) string {
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		var message string
		for i := range e.tmpl.Steps {
			sgNode, err := e.woc.wf.GetNodeByName(e.stepGroupNodeNameAt(i))
			if err != nil || !sgNode.FailedOrError() || sgNode.Message == "" {
				continue
			}
			message = sgNode.Message
		}
		return message
	}
	if e.tmpl.DAG == nil {
		return ""
	}
	for _, task := range e.tmpl.DAG.Tasks {
		if node := e.getTaskNode(ctx, task.Name); node != nil && node.FailedOrError() {
			return fmt.Sprintf("child '%s' failed", node.ID)
		}
	}
	return ""
}

// hasPendingTaskHooks returns true if any task in tasks has a Hooked child
// node that is not yet fulfilled. Used by finalize to gate boundary
// termination on in-flight hook completion (see comment in finalize).
func (e *Engine) hasPendingTaskHooks(ctx context.Context, tasks []dag.Task) bool {
	for _, task := range tasks {
		taskNode := e.getTaskNode(ctx, task.GetName())
		if taskNode == nil {
			continue
		}
		if e.hasPendingHooks(taskNode) {
			return true
		}
		if taskNode.Type != wfv1.NodeTypeTaskGroup {
			continue
		}
		// An expanded task's exit hooks hang off its item nodes.
		for _, childID := range taskNode.Children {
			if childNode, err := e.woc.wf.Status.Nodes.Get(childID); err == nil && e.hasPendingHooks(childNode) {
				return true
			}
		}
	}
	return false
}

// hasPendingHooks reports whether node has a hook child that is not fulfilled.
func (e *Engine) hasPendingHooks(node *wfv1.NodeStatus) bool {
	for _, childID := range node.Children {
		childNode, err := e.woc.wf.Status.Nodes.Get(childID)
		if err != nil {
			continue
		}
		if childNode.NodeFlag != nil && childNode.NodeFlag.Hooked && !childNode.Fulfilled() {
			return true
		}
	}
	return false
}

// saveMemoizationCache persists the node outputs to the memoization cache if configured.
func (e *Engine) saveMemoizationCache(ctx context.Context) error {
	node, err := e.woc.wf.GetNodeByName(e.nodeName)
	if err != nil {
		return nil //nolint:nilerr // node not yet in status → nothing to memoize
	}
	if node.MemoizationStatus == nil {
		return nil
	}
	c := e.woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, node.MemoizationStatus.CacheName)
	if saveErr := c.Save(ctx, node.MemoizationStatus.Key, node.ID, node.Outputs); saveErr != nil {
		e.log.WithError(saveErr).Error(ctx, "Failed to save node outputs to cache")
		_ = e.woc.markNodePhase(ctx, e.nodeName, wfv1.NodeError, saveErr.Error())
		return saveErr
	}
	return nil
}

func (e *Engine) executeTask(ctx context.Context, task dag.Task) (*wfv1.NodeStatus, error) {
	taskName := task.GetName()
	taskNodeName := e.taskNodeName(taskName)

	taskNode := e.getTaskNode(ctx, taskName)
	if taskNode != nil && (taskNode.Fulfilled() || taskNode.Phase == wfv1.NodeRunning) {
		scope, err := e.buildLocalScopeFromTask(ctx, task)
		if err != nil {
			return e.woc.markNodeError(ctx, taskNodeName, err), err
		}
		e.hooks.ref.Status.Set(scope.scope, string(taskNode.Phase), task.GetDisplayName())
		hookCompleted, err := e.hooks.ExecuteLifecycleHooks(ctx, scope, task.GetHooks(), taskNode, task.GetDisplayName())
		if err != nil && !isThrottleErr(err) {
			e.woc.markNodeError(ctx, taskNodeName, err)
		}
		if !hookCompleted {
			return taskNode, nil
		}
	}

	if taskNode != nil && taskNode.Fulfilled() {
		e.log.WithFields(logging.Fields{"task": taskName, "node": taskNodeName}).Debug(ctx, "task already fulfilled")
		return taskNode, nil
	}

	// A scope, resolution or expansion failure is this task's own terminal
	// outcome: it is recorded on an Error node linked under the task's
	// parents (as executeDAGTask did), so siblings keep running and the
	// boundary rolls up from its children.
	parentNodeNames := e.parentNodeNames(ctx, taskName)
	failTask := func(err error) (*wfv1.NodeStatus, error) {
		e.initTerminalErrorNode(ctx, task, parentNodeNames, err)
		return e.getTaskNode(ctx, taskName), err
	}

	// The task's references are resolved once, from one scope, before
	// anything is created for it: a reference not in scope yet leaves the
	// task uncreated, so an expansion never leaves a childless TaskGroup.
	scope, err := e.buildLocalScopeFromTask(ctx, task)
	if err != nil {
		return failTask(err)
	}
	resolved, err := e.resolveTask(ctx, task, scope)
	if err != nil {
		if stderrors.Is(err, ErrRequeue) {
			// Not this task's failure: a dependency output is not in scope yet.
			e.log.WithField("task", taskName).WithError(err).Debug(ctx, "was unable to find variable")
			e.woc.requeue()
			return nil, err
		}
		return failTask(err)
	}

	if dag.HasExpansion(resolved) {
		expandedTasks, expandErr := resolved.Expand(ctx, e.expansionScope(scope), e.woc)
		if expandErr != nil {
			return failTask(expandErr)
		}

		e.expanded[taskName] = expandedTasks

		// Empty expansion (e.g., withParam resolves to []) → skip the task. A
		// group that already exists and now expands to no items (after `argo
		// retry --parameter`) is Skipped too where its phase allows it; a
		// Running group cannot become Skipped, so it is completed Succeeded,
		// with the same message.
		if len(expandedTasks) == 0 {
			if taskNode == nil {
				return e.initTaskNode(ctx, resolved, parentNodeNames, wfv1.NodeTypeSkipped, wfv1.NodeSkipped, "Skipped, empty params"), nil
			}
			phase := wfv1.NodeSkipped
			if !isValidPhaseTransition(taskNode.Phase, phase) {
				phase = wfv1.NodeSucceeded
			}
			return e.woc.markNodePhase(ctx, taskNodeName, phase, "Skipped, empty params"), nil
		}

		tgNode := taskNode
		if tgNode == nil {
			tgNode = e.initTaskNode(ctx, resolved, parentNodeNames, wfv1.NodeTypeTaskGroup, wfv1.NodeRunning)
		}
		if err := e.reconcileTaskGroup(ctx, tgNode, expandedTasks); err != nil {
			return nil, err
		}
		return tgNode, nil
	}

	// Use reconciler for leaf task
	if err := e.reconcileTask(ctx, resolved, parentNodeNames); err != nil {
		// Throttling sentinels mean "didn't materialize, but it's deliberate".
		// Propagate them up so callers can distinguish from real failures, but
		// don't synthesize a fake "no materialization" error here.
		return nil, err
	}

	// Reconciler returned nil claiming success. A leaf task with desired work
	// should have materialized a node. Missing node here is an unexpected
	// silent failure — surface it via ErrReconcilerNoMaterialize so callers
	// don't conflate it with deliberate throttling.
	node := e.getTaskNode(ctx, taskName)
	if node == nil {
		return nil, fmt.Errorf("task %s: %w", taskName, ErrReconcilerNoMaterialize)
	}
	return node, nil
}

// initTaskNode creates a node the Engine records itself for task (Omitted,
// Skipped, TaskGroup or a terminal Error), under the task's own template, and
// links it under parents in the same step, so it is never left unreachable.
func (e *Engine) initTaskNode(ctx context.Context, task dag.Task, parents []string, nodeType wfv1.NodeType, phase wfv1.NodePhase, msg ...string) *wfv1.NodeStatus {
	nodeName := e.taskNodeName(task.GetName())
	_, node := e.woc.initializeNode(ctx, nodeName, nodeType, e.tmplCtx.GetTemplateScope(), task.GetTemplateReferenceHolder(), e.boundaryID, phase, &wfv1.NodeFlag{}, true, msg...)
	for _, parent := range parents {
		e.woc.addChildNode(ctx, parent, nodeName)
	}
	return node
}

// initTerminalErrorNode records err as the task's own terminal outcome: an
// Error node, linked under parents, created if the task has none yet (a
// setup failure such as an unresolvable templateRef or an unhandled absent
// optional, #16223). Its siblings keep running and the boundary rolls up from
// its children.
func (e *Engine) initTerminalErrorNode(ctx context.Context, task dag.Task, parents []string, err error) {
	if e.getTaskNode(ctx, task.GetName()) == nil {
		e.initTaskNode(ctx, task, parents, wfv1.NodeTypeSkipped, wfv1.NodeError, err.Error())
		return
	}
	e.woc.markNodeError(ctx, e.taskNodeName(task.GetName()), err)
}

// createDesiredTask builds the DesiredTask the reconciler executes for task,
// already resolved (see resolveTask), to be linked under parents when its
// node is created. It returns nil for a task that is already fulfilled. A
// setup error is recorded on the task's node.
func (e *Engine) createDesiredTask(ctx context.Context, task dag.Task, parents []string) (*DesiredTask, error) {
	taskName := task.GetName()
	taskNodeName := e.taskNodeName(taskName)

	// Check if already fulfilled
	if taskNode := e.getTaskNode(ctx, taskName); taskNode != nil && taskNode.Fulfilled() {
		return nil, nil
	}

	failTask := func(err error) (*DesiredTask, error) {
		e.initTerminalErrorNode(ctx, task, parents, err)
		return nil, err
	}

	// The when is resolved, so it is evaluated as is (an item's with its
	// {{item}} substituted by the expansion).
	proceed, err := dag.ShouldExecute(task.GetWhen())
	if err != nil {
		return failTask(err)
	}

	if !proceed {
		return &DesiredTask{
			TaskName:        taskNodeName,
			TemplateScope:   e.tmplCtx.GetTemplateScope(),
			TemplateRef:     task.GetTemplateReferenceHolder(),
			BoundaryID:      e.boundaryID,
			IsOnExit:        e.onExitTemplate,
			Skipped:         true,
			SkipReason:      fmt.Sprintf("when '%s' evaluated false", task.GetWhen()),
			ParentNodeNames: parents,
		}, nil
	}

	// Resolve Template and Arguments
	newTmplCtx, resolvedTmpl, templateStored, err := e.tmplCtx.ResolveTemplate(ctx, task.GetTemplateReferenceHolder())
	if err != nil {
		return failTask(err)
	}
	if templateStored {
		e.woc.updated = true
	}

	// Merge templateDefaults (metrics, retryStrategy, etc.) into the resolved template.
	// reconcileTemplate (the entry-template path) does this at operator.go; the Engine
	// dispatch path bypasses reconcileTemplate, so without an explicit merge here, per-task
	// templateDefaults — including the Prometheus metrics that drive
	// argo_workflows_<name>_counter emissions on node completion — are silently dropped.
	if err = e.woc.mergedTemplateDefaultsInto(resolvedTmpl); err != nil {
		return failTask(err)
	}

	args := task.GetArguments()

	// Build minimal local params for ProcessArgs (matching reconcileTemplate behavior).
	localParams := make(common.Parameters)
	localParams["node.name"] = taskNodeName
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		localParams["steps.name"] = task.GetDisplayName()
	} else {
		localParams["tasks.name"] = task.GetDisplayName()
	}
	// Set pod.name for pod-type templates (matching reconcileTemplate behavior).
	if resolvedTmpl.IsPodType() && e.woc.retryStrategy(resolvedTmpl) == nil {
		localParams[varkeys.PodName.Template()] = e.woc.getPodName(taskNodeName, resolvedTmpl.Name)
	}

	processedTmpl, err := common.ProcessArgs(ctx, resolvedTmpl, &args, e.woc.globalParams(), localParams, false, e.woc.wf.Namespace, e.woc.controller.typedConfigMapInformer.GetIndexer())
	if err != nil {
		return failTask(err)
	}

	return &DesiredTask{
		TaskName:        taskNodeName,
		TemplateScope:   e.tmplCtx.GetTemplateScope(),
		TmplCtx:         newTmplCtx,
		Template:        processedTmpl,
		TemplateRef:     task.GetTemplateReferenceHolder(),
		BoundaryID:      e.boundaryID,
		IsOnExit:        e.onExitTemplate,
		ParentNodeNames: parents,
	}, nil
}

func (e *Engine) getTaskByName(tasks []dag.Task, name string) dag.Task {
	for _, task := range tasks {
		if task.GetName() == name {
			return task
		}
	}
	return nil
}

// taskNodeName formulates the nodeName for a dag task
func (e *Engine) taskNodeName(taskName string) string {
	return dag.TaskNodeName(e.nodeName, taskName)
}

// taskNodeID formulates the node ID for a dag task
func (e *Engine) taskNodeID(taskName string) string {
	nodeName := e.taskNodeName(taskName)
	return e.woc.wf.ResolveNodeID(nodeName)
}

// getTaskNode returns the node status of a task.
func (e *Engine) getTaskNode(ctx context.Context, taskName string) *wfv1.NodeStatus {
	nodeID := e.taskNodeID(taskName)
	node, err := e.woc.wf.Status.Nodes.Get(nodeID)
	if err != nil {
		e.log.WithFields(logging.Fields{"nodeID": nodeID, "taskName": taskName}).Debug(ctx, "was unable to obtain the node")
		return nil
	}
	return node
}

// assessDAGPhase assesses the overall DAG status.
// Only leaf tasks (tasks with no dependents) are considered when determining the overall phase.
// This matches the old engine behavior: intermediate task failures are "absorbed" when their
// downstream leaf tasks succeed via enhanced depends logic (e.g. depends: "C.Failed").
func (e *Engine) assessDAGPhase(ctx context.Context, tasks []dag.Task, results map[string]dag.EvaluationResult, isShutdown bool) wfv1.NodePhase {
	if isShutdown {
		return wfv1.NodeFailed
	}

	// First pass: if ANY task is still Running or Pending AND not fulfilled for deps,
	// the DAG is not yet terminal. Tasks that are Running but FulfilledForDeps (e.g.,
	// a retry node whose daemon child is running) don't block the DAG.
	for _, result := range results {
		if (result.CurrentPhase == wfv1.NodeRunning || result.CurrentPhase == wfv1.NodePending) && !result.FulfilledForDeps {
			return wfv1.NodeRunning
		}
	}

	// Steps templates: a step's continueOn is applied when its StepGroup is
	// assessed, and groups run in sequence, so the template fails as soon as
	// any group has failed (matching the pre-Engine executeSteps). The DAG leaf
	// rule below must not be used here: a later step omitted because an earlier
	// group failed would otherwise excuse that failure with its own continueOn.
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		for i := range e.tmpl.Steps {
			if sgNode, err := e.woc.wf.GetNodeByName(e.stepGroupNodeNameAt(i)); err == nil && sgNode.FailedOrError() {
				return wfv1.NodeFailed
			}
		}
		return wfv1.NodeSucceeded
	}

	// Build set of leaf tasks (tasks that no other task depends on).
	leafTasks := e.findLeafTasks(ctx, tasks)

	// When the DAG declares explicit targets, only the target tasks decide the
	// phase: "we only succeed if all the target tasks have been considered"
	// (#693, #3035). The graph leaves may have been pruned out of results
	// entirely when a non-leaf target is declared.
	if e.tmpl.DAG != nil && e.tmpl.DAG.Target != "" {
		leafTasks = make(map[string]bool)
		for _, name := range e.evaluator.GetTargetTasks(ctx) {
			leafTasks[name] = true
		}
	}

	// Build a map for quick result lookup
	resultMap := make(map[string]wfv1.NodePhase, len(results))
	for _, result := range results {
		resultMap[result.TaskName] = result.CurrentPhase
	}

	// All tasks are in terminal states — check for unhandled failures in leaf tasks only.
	// For omitted leaf tasks, inherit the worst phase from their dependencies (matching old engine's
	// branchPhase BFS behavior). This ensures that if a task is omitted because its dependency failed,
	// the failure propagates to the DAG phase.
	// FailFast only applies to DAG templates (Steps templates don't have this concept).
	// For DAGs, failFast defaults to true unless explicitly set to false.
	failFast := e.tmpl.DAG != nil && (e.tmpl.DAG.FailFast == nil || *e.tmpl.DAG.FailFast)
	// Collect and sort leaf task names for deterministic phase assignment
	var leafTaskNames []string
	for _, result := range results {
		if leafTasks[result.TaskName] {
			leafTaskNames = append(leafTaskNames, result.TaskName)
		}
	}
	sort.Strings(leafTaskNames)

	// Build result lookup by task name
	resultByName := make(map[string]dag.EvaluationResult, len(results))
	for _, result := range results {
		resultByName[result.TaskName] = result
	}

	phase := wfv1.NodeSucceeded
	for _, name := range leafTaskNames {
		result := resultByName[name]
		effectiveState := result.CurrentPhase
		if effectiveState == wfv1.NodeOmitted {
			effectiveState = e.inheritedBranchPhase(ctx, name, resultMap)
		}
		if effectiveState == wfv1.NodeFailed || effectiveState == wfv1.NodeError {
			task := e.getTaskByName(tasks, name)
			if !task.ContinuesOn(effectiveState) {
				phase = effectiveState
				if failFast {
					break
				}
			}
		}
	}

	return phase
}

// findLeafTasks returns a set of task names that have no dependents (i.e., no other task depends on them).
// It delegates to the evaluator's FindLeafTaskNames which handles both legacy Dependencies and enhanced Depends fields.
func (e *Engine) findLeafTasks(ctx context.Context, tasks []dag.Task) map[string]bool {
	leaves := e.evaluator.FindLeafTaskNames(ctx)
	leafSet := make(map[string]bool, len(tasks))
	for _, task := range tasks {
		leafSet[task.GetName()] = false
	}
	for _, name := range leaves {
		leafSet[name] = true
	}
	return leafSet
}

// inheritedBranchPhase returns the worst phase from a task's dependencies.
// Used for omitted leaf tasks so they inherit the phase of the branch that
// caused them to be omitted (matching old engine BFS behavior).
func (e *Engine) inheritedBranchPhase(ctx context.Context, taskName string, resultMap map[string]wfv1.NodePhase) wfv1.NodePhase {
	memo := make(map[string]wfv1.NodePhase)
	return e.inheritedBranchPhaseHelper(ctx, taskName, resultMap, memo, make(map[string]bool))
}

func (e *Engine) inheritedBranchPhaseHelper(ctx context.Context, taskName string, resultMap map[string]wfv1.NodePhase, memo map[string]wfv1.NodePhase, onPath map[string]bool) wfv1.NodePhase {
	if phase, ok := memo[taskName]; ok {
		return phase
	}
	if onPath[taskName] {
		// Cycle (shouldn't happen for valid DAGs; defensive) — treat as Succeeded
		// so the recursion terminates without poisoning the worst-phase result.
		return wfv1.NodeSucceeded
	}
	onPath[taskName] = true
	defer delete(onPath, taskName)

	deps, err := e.evaluator.GetDependencies(ctx, taskName)
	if err != nil || len(deps) == 0 {
		memo[taskName] = wfv1.NodeSucceeded
		return wfv1.NodeSucceeded
	}
	worst := wfv1.NodeSucceeded
	for _, dep := range deps {
		depState, ok := resultMap[dep]
		if !ok {
			continue
		}
		if depState == wfv1.NodeOmitted {
			depState = e.inheritedBranchPhaseHelper(ctx, dep, resultMap, memo, onPath)
		}
		if depState == wfv1.NodeError || (depState == wfv1.NodeFailed && worst != wfv1.NodeError) {
			worst = depState
		}
	}
	memo[taskName] = worst
	return worst
}

// parentNodeNames returns the nodes a task's node hangs off in the graph.
// Steps tasks are children of their StepGroup node. DAG tasks are children
// of the outbound nodes of their dependencies, or of the boundary node when
// they have none. The walk visits a task's dependencies before it, so each
// has its node by then; a dependency with no node is skipped.
func (e *Engine) parentNodeNames(ctx context.Context, taskName string) []string {
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		if sgName := e.stepGroupNodeName(taskName); sgName != "" {
			return []string{sgName}
		}
		return []string{e.nodeName}
	}

	deps, err := e.evaluator.GetDependencies(ctx, taskName)
	if err != nil {
		e.log.WithFields(logging.Fields{"taskName": taskName, "error": err}).Warn(ctx, "failed to get dependencies")
		return []string{e.nodeName}
	}
	if len(deps) == 0 {
		return []string{e.nodeName}
	}
	var parents []string
	for _, dep := range deps {
		depNodeID := e.taskNodeID(dep)
		if _, getErr := e.woc.wf.Status.Nodes.Get(depNodeID); getErr != nil {
			continue
		}
		for _, outID := range e.woc.getOutboundNodes(ctx, depNodeID) {
			if outNode, getErr := e.woc.wf.Status.Nodes.Get(outID); getErr == nil {
				parents = append(parents, outNode.Name)
			}
		}
	}
	return parents
}

// stepGroupNodeNameAt is the name of the StepGroup node for group index i.
func (e *Engine) stepGroupNodeNameAt(i int) string {
	return stepGroupNodeName(e.nodeName, i)
}

// stepGroupNodeName extracts the StepGroup node name from a task name.
// Task names for Steps are formatted as "[N].stepName" by StepAdapter.GetName().
func (e *Engine) stepGroupNodeName(taskName string) string {
	if groupIdx, ok := stepGroupIndexOf(taskName); ok {
		return e.stepGroupNodeNameAt(groupIdx)
	}
	return ""
}

// getChildNodes returns all direct child NodeStatus objects of a node.
func (e *Engine) getChildNodes(node *wfv1.NodeStatus) []wfv1.NodeStatus {
	children := make([]wfv1.NodeStatus, 0, len(node.Children))
	for _, childID := range node.Children {
		if child, err := e.woc.wf.Status.Nodes.Get(childID); err == nil {
			children = append(children, *child)
		}
	}
	return children
}

// addTaskNodeToScope adds one dependency/step node's outputs to scope: aggregates
// TaskGroup children so {{tasks.X.outputs.parameters.foo}} resolves to the JSON array
// of all values, adds the live scope entries, and back-fills a skipped/omitted node's
// declared outputs (producer's valueFrom.default where present, else nil) so downstream
// refs resolve instead of requeuing forever. The task (not the node) is the template
// holder: a skipped node alone resolves to the boundary template, not its own.
//
// Once the task has finished, so has everything its part reads (a running
// daemon has not, and neither has the StepGroup an expanded step is read
// through until the walk closes it), and its part is built once per
// reconcile and kept in e.finished.
func (e *Engine) addTaskNodeToScope(ctx context.Context, scope *wfScope, ref varkeys.NodeRefKeys, agg varkeys.AggregateKeys, refName, taskName string, node *wfv1.NodeStatus, includeArtifacts bool) error {
	scopeNode := e.scopeNodeForTask(taskName, node)
	if !node.Fulfilled() || node.IsDaemoned() || !scopeNode.Fulfilled() {
		return e.buildTaskNodeScope(ctx, scope, ref, agg, refName, taskName, node, scopeNode, includeArtifacts)
	}
	own, ok := e.finished[taskName]
	if !ok {
		built := createScope(nil)
		if err := e.buildTaskNodeScope(ctx, built, ref, agg, refName, taskName, node, scopeNode, includeArtifacts); err != nil {
			return err
		}
		own = built.scope
		if e.finished == nil {
			e.finished = make(map[string]*variables.Scope)
		}
		e.finished[taskName] = own
	}
	scope.scope.Merge(own)
	return nil
}

// buildTaskNodeScope adds what addTaskNodeToScope describes to scope, from
// node and scopeNode, the node that represents the task in the scope.
func (e *Engine) buildTaskNodeScope(ctx context.Context, scope *wfScope, ref varkeys.NodeRefKeys, agg varkeys.AggregateKeys, refName, taskName string, node, scopeNode *wfv1.NodeStatus, includeArtifacts bool) error {
	if node.Type == wfv1.NodeTypeTaskGroup {
		if err := e.woc.processAggregateNodeOutputs(scope, agg, refName, e.getChildNodes(node)); err != nil {
			return fmt.Errorf("failed to aggregate outputs for %s: %w", taskName, err)
		}
	}
	e.woc.buildLocalScope(scope, ref, refName, scopeNode)
	holder := wfv1.TemplateReferenceHolder(node)
	if t := e.evaluator.GetTask(taskName); t != nil {
		holder = t.GetTemplateReferenceHolder()
	}
	e.woc.addSkippedNodeOutputsToScope(ctx, e.tmplCtx, scope, ref, refName, node, holder, includeArtifacts)
	return nil
}

// scopeNodeForTask returns the node whose fields ({{steps.X.id}}, .status, ...)
// represent taskName in a scope. For an expanded Steps step the pre-Engine
// controller exposed the enclosing StepGroup node — Steps had no TaskGroup
// node — and that is preserved for compatibility: the TaskGroup node only
// feeds the aggregate outputs. DAG tasks expose their own (TaskGroup) node.
func (e *Engine) scopeNodeForTask(taskName string, node *wfv1.NodeStatus) *wfv1.NodeStatus {
	if e.tmpl.GetType() != wfv1.TemplateTypeSteps || node.Type != wfv1.NodeTypeTaskGroup {
		return node
	}
	if sgNode, err := e.woc.wf.GetNodeByName(e.stepGroupNodeName(taskName)); err == nil {
		return sgNode
	}
	return node
}

// buildLocalScopeFromTask builds a local scope for a task.
func (e *Engine) buildLocalScopeFromTask(ctx context.Context, task dag.Task) (*wfScope, error) {
	scope := createScope(e.tmpl)
	// Add all ancestor tasks' outputs to scope (transitive closure of dependencies).
	// A task may reference outputs from any ancestor, not just direct dependencies
	// (e.g., {{tasks.grandparent.ip}} in a DAG), as the pre-Engine controller did
	// with GetTaskAncestry.
	ancestorNames, err := e.evaluator.GetAncestors(ctx, task.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to get ancestors for task %s: %w", task.GetName(), err)
	}
	for _, depName := range ancestorNames {
		depNode := e.getTaskNode(ctx, depName)
		if depNode == nil {
			continue // ancestor may not have a node yet (e.g., dag.target filtering)
		}
		ref, agg, refName := varkeys.TasksNodeRef, varkeys.TasksAggregate, depName
		if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
			ref, agg = varkeys.StepsNodeRef, varkeys.StepsAggregate
			parts := strings.SplitN(depName, ".", 2)
			if len(parts) == 2 {
				refName = parts[1]
			}
		}

		// Steps keeps skipped-node artifact placeholders resolvable (includeArtifacts); DAG
		// leaves them to resolveArtifactArguments' optional-drop / required-error handling.
		if err := e.addTaskNodeToScope(ctx, scope, ref, agg, refName, depName, depNode, e.tmpl.GetType() == wfv1.TemplateTypeSteps); err != nil {
			return nil, err
		}
	}

	// Add workflow-level global outputs to scope so that references like
	// {{workflow.outputs.artifacts.my-art}} and {{workflow.outputs.parameters.my-param}}
	// can be resolved. These are populated by addOutputsToGlobalScope during execution.
	e.woc.addWorkflowOutputsToLocalScope(e.woc.wf.Status.Outputs, scope)

	// For steps templates, a step can reference outputs from ANY earlier group, not just
	// its direct predecessor. The old executeStepGroup accumulated scope cumulatively across
	// all groups. Replicate that here by adding all preceding groups' outputs.
	// Step task names are formatted as "[N].stepName" by StepAdapter.GetName(), so we
	// parse the group index from the name prefix.
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		currentGroupIdx, ok := stepGroupIndexOf(task.GetName())
		if !ok {
			return nil, fmt.Errorf("failed to parse group index from step task name %q", task.GetName())
		}
		for i, stepGroup := range e.tmpl.Steps {
			if i >= currentGroupIdx {
				break
			}
			for _, step := range stepGroup.Steps {
				stepTaskName := stepTaskNameFor(i, step.Name)
				stepNode := e.getTaskNode(ctx, stepTaskName)
				if stepNode == nil {
					continue
				}
				if err := e.addTaskNodeToScope(ctx, scope, varkeys.StepsNodeRef, varkeys.StepsAggregate, step.Name, stepTaskName, stepNode, true); err != nil {
					return nil, err
				}
			}
		}
	}

	return scope, nil
}

// setDAGOutputs sets the outputs of the DAG.
func (e *Engine) setDAGOutputs(ctx context.Context) error {
	node, err := e.woc.wf.GetNodeByName(e.nodeName)
	if err != nil {
		return err
	}
	scope := createScope(e.tmpl)
	// Seed the scope with the workflow's own outputs so that a template's own
	// output params/artifacts can read {{workflow.outputs.*}} (C43), as base's
	// executeSteps did — and, as an improvement, executeDAG too (P8).
	e.woc.addWorkflowOutputsToLocalScope(e.woc.wf.Status.Outputs, scope)

	includeArtifacts := e.tmpl.GetType() == wfv1.TemplateTypeSteps

	if e.tmpl.DAG != nil {
		for _, task := range e.tmpl.DAG.Tasks {
			taskNode := e.getTaskNode(ctx, task.Name)
			if taskNode == nil {
				continue
			}
			if err = e.addTaskNodeToScope(ctx, scope, varkeys.TasksNodeRef, varkeys.TasksAggregate, task.Name, task.Name, taskNode, includeArtifacts); err != nil {
				return err
			}
			e.woc.addOutputsToGlobalScope(ctx, taskNode.Outputs)
		}
	} else if e.tmpl.Steps != nil {
		for i, stepGroup := range e.tmpl.Steps {
			for _, step := range stepGroup.Steps {
				// Step nodes use the [i].name format for lookup, but steps.name for scope prefix.
				taskName := stepTaskNameFor(i, step.Name)
				taskNode := e.getTaskNode(ctx, taskName)
				if taskNode == nil {
					continue
				}
				if err = e.addTaskNodeToScope(ctx, scope, varkeys.StepsNodeRef, varkeys.StepsAggregate, step.Name, taskName, taskNode, includeArtifacts); err != nil {
					return err
				}
				e.woc.addOutputsToGlobalScope(ctx, taskNode.Outputs)
			}
		}
	}

	outputs, err := e.woc.getTemplateOutputsFromScope(ctx, e.tmpl, scope)
	if err != nil {
		return err
	}
	if outputs != nil {
		node.Outputs = outputs
		e.woc.addOutputsToGlobalScope(ctx, node.Outputs)
		e.woc.wf.Status.Nodes.Set(ctx, node.ID, *node)
	}
	return nil
}

// updateOutboundNodesForTargetTasks sets the outbound nodes for the target tasks.
func (e *Engine) updateOutboundNodesForTargetTasks(ctx context.Context, targetTasks []string) error {
	outbound := make([]string, 0)
	for _, taskName := range targetTasks {
		taskNode := e.getTaskNode(ctx, taskName)
		if taskNode != nil {
			outbound = append(outbound, e.woc.getOutboundNodes(ctx, taskNode.ID)...)
		}
	}
	node, err := e.woc.wf.GetNodeByName(e.nodeName)
	if err != nil {
		return err
	}
	node.OutboundNodes = outbound
	e.woc.wf.Status.Nodes.Set(ctx, node.ID, *node)
	return nil
}

// resolveTask substitutes the task's references to other tasks' and steps'
// outputs across its whole body except its hooks (arguments, including
// artifact locations, templateRef, withItems/withParam/withSequence and
// when), and resolves its artifact arguments' from/fromExpression, as
// resolveDependencyReferences and resolveReferences did before the Engine.
// spec.volumes, which may reference them too (docs/variables.md), are
// substituted from the same scope.
//
// The when is resolved and evaluated first. A task whose when is false is
// returned with only its when resolved, so its other references need not
// resolve: it is skipped, or expanded leniently (a dynamic list that does not
// parse gives no items, a literal list gives Skipped items). An expanded
// task's when that cannot be evaluated yet (it needs {{item}}) is evaluated
// per item instead. A reference that is not in scope yet is ErrRequeue.
func (e *Engine) resolveTask(ctx context.Context, task dag.Task, scope *wfScope) (dag.Task, error) {
	// Globals were substituted into the volumes once at operate start.
	if err := e.woc.substituteParamsInVolumes(ctx, scope.getParametersAny(nil)); err != nil {
		return nil, err
	}
	// nil-preserving view so expression tags can apply `??` fallbacks to skipped/omitted outputs
	params := scope.getParametersAny(e.woc.globalParams())
	if when := task.GetWhen(); when != "" {
		resolvedWhen, err := substituteJSON(ctx, when, params)
		if err != nil {
			return nil, err
		}
		proceed, err := dag.ShouldExecute(resolvedWhen)
		if err != nil && !dag.HasExpansion(task) {
			return nil, err
		}
		if err == nil && !proceed {
			return task.Resolve(func(body wfv1.DAGTask) (wfv1.DAGTask, error) {
				body.When = resolvedWhen
				return body, nil
			})
		}
	}
	return task.Resolve(func(body wfv1.DAGTask) (wfv1.DAGTask, error) {
		// Hooks are resolved when they run: an exit hook may reference this
		// task's own outputs.
		hooks := body.Hooks
		body.Hooks = nil
		// A pure reference to an absent optional becomes a sentinel that
		// ProcessArgs reads as "unsupplied" (see markAbsentOptionalArgs).
		scope.markAbsentOptionalArgs(&body.Arguments)
		body, err := substituteJSON(ctx, body, params)
		if err != nil {
			return body, err
		}
		if body.Arguments.Artifacts, err = scope.resolveArtifactArguments(ctx, body.Arguments.Artifacts); err != nil {
			return body, err
		}
		body.Hooks = hooks
		return body, nil
	})
}

// substituteJSON substitutes the tags params resolves in v's JSON form:
// values are escaped for the JSON context and the unmarshal reverses it, so
// values containing quotes or backslashes arrive intact. References to tasks
// and steps must resolve: a missing one means the producer's output is not
// in scope yet, reported as ErrRequeue so the caller waits rather than run
// with the literal tag (#15513). Other tags stay for later passes.
func substituteJSON[T any](ctx context.Context, v T, params map[string]any) (T, error) {
	var out T
	b, err := json.Marshal(v)
	if err != nil {
		return out, err
	}
	resolved, err := template.ReplaceStrictAny(ctx, string(b), params, []string{"tasks", "steps"})
	if err != nil {
		if template.IsMissingVariableErr(err) {
			return out, fmt.Errorf("%w: %w", ErrRequeue, err)
		}
		return out, err
	}
	err = json.Unmarshal([]byte(resolved), &out)
	return out, err
}
