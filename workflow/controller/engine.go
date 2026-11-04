package controller

import (
	"cmp"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"maps"
	"slices"
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
	onExitTemplate bool
	log            logging.Logger
	reconciler     TaskReconciler
	hooks          *hookHandler
	// finished holds, per finished task, what the task adds to its
	// dependants' scopes, built on first use: a finished node does not
	// change within a reconcile, and rebuilding its part for every dependant
	// makes a chain of n tasks cost n² template resolutions.
	finished map[string]*variables.Scope
	// hookErr is the hook error that ends the boundary Error: that of a hook
	// node found Error when the reconcile starts (one that could not be
	// created, timed out, errored while it ran, or had its error recorded on
	// it; a hook that ran and Failed is not an error), or one raised in this
	// reconcile (markHookError). While it is set no new task node is created,
	// and finalize ends the boundary Error once nothing in it is running.
	hookErr error
}

// NewEngine creates a new Engine.
func NewEngine(woc *wfOperationCtx, nodeName string, tmplCtx *templateresolution.TemplateContext, tmpl *wfv1.Template, boundaryID string, onExitTemplate bool) *Engine {
	return &Engine{
		woc:            woc,
		nodeName:       nodeName,
		tmplCtx:        tmplCtx,
		tmpl:           tmpl,
		boundaryID:     boundaryID,
		onExitTemplate: onExitTemplate,
		log:            woc.log,
		reconciler:     NewK8sTaskReconciler(woc, tmplCtx, nodeName),
		hooks:          newHookHandler(woc, tmplCtx, boundaryID, tmpl, woc.log),
	}
}

// Execute reconciles a DAG or Steps template in one walk over its tasks in
// dependency order: the tasks come ordered (DAG: dag.PullOrder from the
// targets, which leaves out tasks no target needs; Steps: as written). Each
// task is evaluated immediately before the walk acts on it, so it sees what
// the walk has just done to its dependencies: an instant completion (a
// when-false skip, a memoize hit, a nested template that finished), an
// Omitted node, a StepGroup closed at the group boundary, or an exit hook it
// must wait for (#12192). Each task is visited once per reconcile, so it is
// dispatched at most once and its exit handler is driven at most once
// (#14392).
//
// Before the walk, a Retry node whose daemon has died is made unfulfilled
// again (clearStaleDaemonedRetries), and every task node that has just
// finished, or still holds a lock, is finished (reconcileFulfilledTasks);
// after it, the boundary is assessed (a Steps template group by group) and
// finalized. An error of the template itself ends the boundary Error
// (markBoundaryError).
func (e *Engine) Execute(ctx context.Context, tasks []dag.Task) {
	e.evaluator = dag.NewDAGEvaluatorFromTasks(e.woc.wf, tasks, e.tmpl, e.boundaryID, e.nodeName)

	e.clearStaleDaemonedRetries(ctx, tasks)
	e.reconcileFulfilledTasks(ctx, tasks)

	if hook := e.findTaskHook(ctx, tasks, func(n *wfv1.NodeStatus) bool { return n.Phase == wfv1.NodeError }); hook != nil {
		e.hookErr = stderrors.New(cmp.Or(hook.Message, "hook "+hook.Name+" errored"))
	}
	exitHooksDone, dispatching, group, groupFailed := true, true, 0, false
	for _, task := range tasks {
		name := task.GetName()
		// A step group is closed before the next one starts, so a step sees
		// the recorded phase of the group before it ({{steps.X.status}} of an
		// expanded step). Once a group has failed no later step starts: one
		// that would start is Omitted instead, since its dependencies can
		// read a recorded Succeeded (a TaskGroup whose daemon item died
		// later).
		for i, ok := stepGroupIndexOf(name); ok && group < i; group++ {
			phase, _, _ := e.assessStepGroup(ctx, group)
			groupFailed = groupFailed || phase.FailedOrError()
		}
		result := e.evaluator.Evaluate(ctx, name)
		if groupFailed && result.ShouldRun && e.getTaskNode(ctx, name) == nil {
			result = dag.EvaluationResult{TaskName: name, Skipped: true, SkipReason: "an earlier step group failed"}
		}
		stop := e.visit(ctx, task, result, dispatching)
		dispatching = dispatching && !stop
		// The task's hooks are driven before any dependant is evaluated, so a
		// dependant waits for a pending exit hook in this same walk.
		exitHooksDone = e.processHooks(ctx, task) && exitHooksDone
	}

	if err := e.finalize(ctx, tasks, exitHooksDone); err != nil {
		e.markBoundaryError(ctx, err)
	}
}

// visit acts on one task's evaluation: it records an evaluation error on the
// task's node, creates the Omitted node of a task that can never run, or
// dispatches a task the evaluator found runnable, unless dispatching has
// stopped at the operation deadline, or the task has no node yet and a hook
// error is ending the boundary (what already runs is still reconciled). stop
// is dispatchOutcome's.
func (e *Engine) visit(ctx context.Context, task dag.Task, result dag.EvaluationResult, dispatching bool) (stop bool) {
	name := task.GetName()
	e.logEvaluation(ctx, result)
	node := e.getTaskNode(ctx, name)
	switch {
	case result.Error != nil:
		// The evaluator could not assess this task (e.g. its depends
		// expression failed to evaluate). Record that as a terminal Error
		// node so the boundary can assess it; left unrecorded, the task
		// would stay Pending and the boundary would never complete.
		if node == nil || !node.Fulfilled() {
			e.initTerminalErrorNode(ctx, task, e.parentsFor(ctx, name), result.Error)
		}
		return false
	case result.Skipped && !result.ShouldRun:
		// It can never run: record its Omitted node, so its dependants (later
		// in the walk) and the boundary's assessment see it.
		if node == nil {
			reason := result.SkipReason
			if reason == "" {
				reason = "depends condition not met"
			}
			e.initTaskNode(ctx, task, e.parentsFor(ctx, name), wfv1.NodeTypeSkipped, wfv1.NodeOmitted, "omitted: "+reason)
		}
		return false
	case !dispatching || (node == nil && e.hookErr != nil) || !result.ShouldRun:
		return false
	}
	_, err := e.executeTask(ctx, task)
	return e.dispatchOutcome(ctx, name, err)
}

// markBoundaryError records an error of the template itself (its outputs,
// its memoization, the aggregation of a step's items; see templateError) as
// an Error on the boundary node, for DAG and Steps alike. A boundary that is
// already fulfilled is left alone: terminal phases have no valid transitions.
func (e *Engine) markBoundaryError(ctx context.Context, err error) {
	if node, _ := e.woc.wf.GetNodeByName(e.nodeName); node != nil && node.Fulfilled() {
		return
	}
	e.woc.markNodeError(ctx, e.nodeName, err)
}

// templateError is an error of the enclosing template itself rather than of
// the task being dispatched: the task gets no node, and the boundary ends
// Error (markBoundaryError).
type templateError struct{ error }

func (e templateError) Unwrap() error { return e.error }

// clearStaleDaemonedRetries clears the Daemoned flag of a task's Retry node
// whose last attempt is no longer a running daemon, so the walk finds the
// node unfulfilled and its retry handling runs. A daemoned task is otherwise
// fulfilled, and has nothing to dispatch.
func (e *Engine) clearStaleDaemonedRetries(ctx context.Context, tasks []dag.Task) {
	for _, task := range tasks {
		taskNode := e.getTaskNode(ctx, task.GetName())
		if taskNode == nil || !taskNode.IsDaemoned() || taskNode.Type != wfv1.NodeTypeRetry {
			continue
		}
		if _, lastChild := getChildNodeIdsAndLastRetriedNode(taskNode, e.woc.wf.Status.Nodes); lastChild != nil && !lastChild.IsDaemoned() {
			taskNode.Daemoned = nil
			e.woc.wf.Status.Nodes.Set(ctx, taskNode.ID, *taskNode)
			e.woc.updated = true
		}
	}
}

// processHooks drives the hooks of a task's node and reports whether they are
// done. An expanded task's hooks are its items', which reconcileTaskGroup
// drives with the items, so this controller gives a TaskGroup node no hook of
// its own. One started by an older controller can have some (that controller
// ran a DAG task's lifecycle hooks on its TaskGroup): they are re-entered
// until they finish, and none is created. Likewise an item's hook node that
// has not finished under a TaskGroup recorded before this reconcile
// (`argo retry` reset it) is re-entered, as a task's own existing hook node
// always is.
func (e *Engine) processHooks(ctx context.Context, task dag.Task) bool {
	node := e.getTaskNode(ctx, task.GetName())
	if node == nil || node.Type != wfv1.NodeTypeTaskGroup {
		return e.driveHooks(ctx, task, []dag.Task{task}, false)
	}
	nodeTasks := []dag.Task{task}
	if e.hooks.hasHooks(task) && e.itemHooksToReenter(node) {
		items, err := e.resolveItems(ctx, task, true)
		if err != nil {
			e.log.WithField("task", task.GetName()).WithError(err).Warn(ctx, "cannot re-enter the hooks of a completed task's items")
		}
		nodeTasks = append(nodeTasks, items...)
	}
	return e.driveHooks(ctx, task, nodeTasks, true)
}

// itemHooksToReenter reports whether tg, a TaskGroup, was recorded before
// this reconcile (so reconcileTaskGroup did not drive its items in it) and
// one of its items has a hook node to re-enter.
func (e *Engine) itemHooksToReenter(tg *wfv1.NodeStatus) bool {
	if prev, ok := e.woc.preExecutionNodeStatuses[tg.ID]; !ok || !prev.Fulfilled() {
		return false
	}
	for _, item := range e.itemNodes(tg) {
		if len(e.hooks.hookNodesToReenter(item)) > 0 {
			return true
		}
	}
	return false
}

// driveHooks drives, through hookHandler.DriveTaskHooks, the hooks of each
// node of task that ran: task's own node, or, for an expanded task, each item
// node, with that item's hooks ({{item}} substituted). With existingOnly, or
// when the hooks' scope cannot be built, it only re-enters the hook nodes
// that already exist (hookHandler.reenterHooks). The hooks refer to the task
// by its own name and see its hookScope, built only when there is a hook to
// drive, and copied for each node: driving a node's hooks writes its status
// and outputs into the scope under the task's name, and one item's hooks must
// not see another's. It reports whether every hook is done; a node whose
// hooks errored is done once none of its hook nodes is still running, its
// error recorded (markHookError), so that an error that recurs does not hold
// its task back for good.
func (e *Engine) driveHooks(ctx context.Context, task dag.Task, nodeTasks []dag.Task, existingOnly bool) bool {
	if !e.hooks.hasHooks(task) {
		return true
	}
	var taskScope *wfScope
	var scopeErr error
	done := true
	for _, nodeTask := range nodeTasks {
		node := e.getTaskNode(ctx, nodeTask.GetName())
		if node == nil || !ran(node) || (existingOnly && len(e.hooks.hookNodesToReenter(node)) == 0) {
			continue
		}
		if taskScope == nil && scopeErr == nil {
			taskScope, scopeErr = e.hookScope(ctx, task)
		}
		var scope *wfScope
		if taskScope != nil {
			scope = taskScope.clone()
		}
		var nodeDone bool
		var err error
		if existingOnly || scopeErr != nil {
			nodeDone, err = e.hooks.reenterHooks(ctx, nodeTask, task.GetDisplayName(), node, scope)
			e.markHookError(ctx, node, scopeErr)
		} else {
			nodeDone, err = e.hooks.DriveTaskHooks(ctx, nodeTask, task.GetDisplayName(), node, scope)
		}
		e.markHookError(ctx, node, err)
		done = done && (nodeDone || ((err != nil || scopeErr != nil) && common.CheckAllHooksFullfilled(node, e.woc.wf.Status.Nodes)))
	}
	return done
}

// ran reports whether n's task ran: a task skipped by its when clause or
// omitted by its dependencies never did, and has no hooks to run.
func ran(n *wfv1.NodeStatus) bool {
	return n.Phase != wfv1.NodeSkipped && n.Phase != wfv1.NodeOmitted
}

// hookScope is the scope a task's hooks see: the task's own scope and, for a
// step, the status of the steps in its group.
func (e *Engine) hookScope(ctx context.Context, task dag.Task) (*wfScope, error) {
	scope, err := e.buildLocalScopeFromTask(ctx, task)
	if err != nil || e.tmpl.GetType() != wfv1.TemplateTypeSteps {
		return scope, err
	}
	group, _ := stepGroupIndexOf(task.GetName())
	for _, step := range e.tmpl.Steps[group].Steps {
		if node := e.getTaskNode(ctx, stepTaskNameFor(group, step.Name)); node != nil {
			varkeys.StepsNodeRef.Status.Set(scope.scope, string(node.Phase), step.Name)
		}
	}
	return scope, nil
}

// markHookError records a hook error of node, the node whose hook failed: it
// ends the boundary Error (see hookErr), and a node that has not finished
// also takes it as its own error. The hook's own node is already Error
// (errorHookNode, or it failed to run).
func (e *Engine) markHookError(ctx context.Context, node *wfv1.NodeStatus, err error) {
	if err == nil {
		return
	}
	e.log.WithError(err).WithField("node", node.Name).Error(ctx, "task hook errored")
	if n, getErr := e.woc.wf.GetNodeByName(node.Name); getErr == nil && !n.Fulfilled() {
		e.woc.markNodeError(ctx, node.Name, err)
	}
	e.hookErr = cmp.Or(e.hookErr, err)
}

// assessStepGroups starts each empty StepGroup once the group before it has
// finished, records each group's phase once it is done (assessStepGroup), and
// returns the Steps template's phase with the message that explains a
// failure. Groups run in sequence, so the template is Running at the first
// group that has not finished and Failed at the first group that failed.
// Every group is derived again from its steps on each call: a running daemon
// finishes its group, and if it dies later the template fails, although the
// group, already recorded, keeps its phase.
func (e *Engine) assessStepGroups(ctx context.Context) (wfv1.NodePhase, string) {
	phase, message := wfv1.NodeSucceeded, ""
	for i, stepGroup := range e.tmpl.Steps {
		if len(stepGroup.Steps) == 0 && e.previousStepGroupPhase(i).Fulfilled(nil) {
			// An empty group has no step to start it: it starts, and ends,
			// once the group before it has.
			e.startStepGroup(ctx, i)
		}
		groupPhase, groupMessage, done := e.assessStepGroup(ctx, i)
		switch {
		case phase != wfv1.NodeSucceeded:
			// An earlier group has decided; this one is still recorded.
		case !done:
			phase = wfv1.NodeRunning
		case groupPhase.FailedOrError():
			phase, message = wfv1.NodeFailed, groupMessage
		}
	}
	return phase, message
}

// previousStepGroupPhase is the phase of the group before group i: Succeeded
// for the first group, and "" when that group has not started.
func (e *Engine) previousStepGroupPhase(i int) wfv1.NodePhase {
	if i == 0 {
		return wfv1.NodeSucceeded
	}
	prev, err := e.woc.wf.GetNodeByName(e.stepGroupNodeNameAt(i - 1))
	if err != nil {
		return ""
	}
	return prev.Phase
}

// assessStepGroup derives group i's phase from its steps (stepGroupOutcome)
// and records it once the group is done. A group that has not started is
// not done; one already recorded Failed or Error by something other than its
// steps (a deadline, an error recorded on the group) keeps that.
func (e *Engine) assessStepGroup(ctx context.Context, i int) (phase wfv1.NodePhase, message string, done bool) {
	sgNode, err := e.woc.wf.GetNodeByName(e.stepGroupNodeNameAt(i))
	if err != nil {
		return wfv1.NodeRunning, "", false
	}
	if sgNode.FailedOrError() {
		return sgNode.Phase, sgNode.Message, true
	}
	phase, message, done = e.stepGroupOutcome(ctx, i)
	if done && !sgNode.Fulfilled() {
		e.woc.markNodePhase(ctx, sgNode.Name, phase, message)
	}
	return phase, message, done
}

// stepGroupOutcome derives group i's phase from its steps. The group is done
// once every step has a node that has finished (see outcome: a running daemon
// has, a step whose hooks still run has not); a step that a hook error
// stopped from starting (hookErr) has no node and never will, so it does not
// hold the group open. The first step that failed or errored without
// continueOn then decides it, in the message that bubbles up to the workflow
// status: an errored step makes it Error, "step group deemed errored due to
// child <name> error: <reason>", whether the step errored before it ran (a
// setup error) or while it ran; a failed one makes it Failed, "child '<id>'
// failed". It is Omitted if it never ran (every step omitted because an
// earlier group failed, or an empty group after a group that did not
// succeed), and Succeeded otherwise.
func (e *Engine) stepGroupOutcome(ctx context.Context, i int) (phase wfv1.NodePhase, message string, done bool) {
	steps := e.tmpl.Steps[i].Steps
	phase = wfv1.NodeSucceeded
	allOmitted := len(steps) > 0 || e.previousStepGroupPhase(i) != wfv1.NodeSucceeded
	for _, step := range steps {
		node := e.getTaskNode(ctx, stepTaskNameFor(i, step.Name))
		if node == nil && e.hookErr != nil {
			continue // never starts: a hook error stops new tasks
		}
		if node == nil {
			return wfv1.NodeRunning, "", false
		}
		stepPhase, stepDone := e.outcome(node)
		if !stepDone {
			return wfv1.NodeRunning, "", false
		}
		allOmitted = allOmitted && stepPhase == wfv1.NodeOmitted
		if stepPhase.FailedOrError() && !step.ContinuesOn(stepPhase) && message == "" {
			failed := e.failedNode(node)
			phase, message = wfv1.NodeFailed, fmt.Sprintf("child '%s' failed", failed.ID)
			if failed.Phase == wfv1.NodeError {
				phase, message = wfv1.NodeError, fmt.Sprintf("step group deemed errored due to child %s error: %s", failed.Name, cmp.Or(failed.Message, string(failed.Phase)))
			}
		}
	}
	if allOmitted {
		phase = wfv1.NodeOmitted
	}
	return phase, message, true
}

// failedNode is the node a failure message names for a failed step or task:
// its own node, or, for an expanded one's TaskGroup, its first failed item —
// the node with the pod, or the item's Retry node under a retryStrategy — as
// when items hung directly off the StepGroup.
func (e *Engine) failedNode(node *wfv1.NodeStatus) *wfv1.NodeStatus {
	if node.Type == wfv1.NodeTypeTaskGroup {
		for _, item := range e.itemNodes(node) {
			if item.FailedOrError() {
				return item
			}
		}
	}
	return node
}

// isThrottleErr reports whether err is a deliberate throttling signal from
// the reconciler. These are not real failures — the caller should hold the
// work back for now but must not treat the situation as fatal.
func isThrottleErr(err error) bool {
	return stderrors.Is(err, ErrParallelismReached) ||
		stderrors.Is(err, ErrResourceRateLimitReached) ||
		stderrors.Is(err, ErrDeadlineExceeded)
}

// dispatchOutcome applies the per-task dispatch error policy and
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
// workflow globals under the task's own scope. Globals are needed here
// because an expression tag that mixes {{item}} with
// {{workflow.parameters.x}} can only be evaluated once the item is known
// (#14718); simple global tags were substituted at operate start.
func (e *Engine) expansionScope(scope *wfScope) map[string]string {
	params := make(map[string]string)
	maps.Copy(params, e.woc.globalParams())
	maps.Copy(params, scope.getParameters())
	return params
}

// reconcileTaskGroup drives the items of an expanded task on every reconcile
// until the group finishes: each item is created if it has no node yet (the
// rest of a fan-out held back by parallelism or the operation deadline) or
// re-entered if it has not finished (a deleted pod, a suspend with a
// duration, a nested template, a lock waiter), one item's error staying with
// that item. Each item's hooks are then driven (driveHooks), and the group is
// completed from its items once every one exists and has finished, with its
// hooks, so that it is never recorded before an item's exit hook exists. A
// hook error (hookErr) stops the items with no node: the group completes from
// the others (Omitted if there are none) with
// common.TaskGroupHookStoppedMessage, which `argo retry` resets. items are
// the expansion of task, resolved. Only the operation deadline stops the
// items early; its error is returned.
func (e *Engine) reconcileTaskGroup(ctx context.Context, task dag.Task, tgNode *wfv1.NodeStatus, items []dag.Task) error {
	var msg []string
	if e.hookErr != nil {
		n := len(items)
		items = slices.DeleteFunc(slices.Clone(items), func(item dag.Task) bool { return e.getTaskNode(ctx, item.GetName()) == nil })
		if n > len(items) {
			msg = append(msg, fmt.Sprintf("%s%d", common.TaskGroupHookStoppedMessage, n-len(items)))
		}
	}
	var stopErr error
	for _, item := range items {
		err := e.reconcileTask(ctx, item, []string{tgNode.Name})
		if e.dispatchOutcome(ctx, item.GetName(), err) {
			stopErr = err
			break
		}
	}
	hooksDone := e.driveHooks(ctx, task, items, false)
	itemNodes := make([]*wfv1.NodeStatus, len(items))
	for i, item := range items {
		if n := e.getTaskNode(ctx, item.GetName()); n != nil && common.CheckAllHooksFullfilled(n, e.woc.wf.Status.Nodes) {
			itemNodes[i] = n
		}
	}
	if phase, done := dag.TaskGroupPhase(itemNodes); done && hooksDone {
		if len(items) == 0 {
			phase = wfv1.NodeOmitted
		}
		e.woc.markNodePhase(ctx, tgNode.Name, phase, msg...)
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

// reconcileFulfilledTasks finishes, before anything is dispatched, each
// task node (each item, for an expanded task) that has run to completion
// since the reconcile started, or that still holds a lock (toFinish): the
// reconciler's handleNodeFulfilled releases its lock and, once per
// completion, emits its completion metrics and exports its globalName
// outputs. This is where a node that finished outside a dispatch (a pod, a
// resumed suspend, an HTTP task, a failFast or timeout mark) is finished. A
// node that had already completed, and holds no lock, was finished when it
// completed, so a finished fan-out is not resolved again on every reconcile.
// Each node is reconciled with the template it was dispatched with: the task
// resolved against its scope, items expanded, templateDefaults merged. A task
// whose template has no lock and no metrics has nothing to finish
// (mayNeedFinishing). Legacy items, which have no TaskGroup until dispatch
// creates one, are finished when they are adopted (adoptItems).
func (e *Engine) reconcileFulfilledTasks(ctx context.Context, tasks []dag.Task) {
	holders := lockHolders(e.woc.wf)
	for _, task := range tasks {
		node := e.getTaskNode(ctx, task.GetName())
		if node == nil || !e.hasNodeToFinish(node, holders) || !e.mayNeedFinishing(ctx, task) {
			continue
		}
		log := e.log.WithField("task", task.GetName())
		items, err := e.resolveItems(ctx, task, node.Type == wfv1.NodeTypeTaskGroup)
		if err != nil {
			// A task that could not be resolved to be dispatched ended Error
			// without running, and cannot be resolved now either.
			log.WithError(err).Warn(ctx, "cannot finish a completed task")
			continue
		}
		var desired []DesiredTask
		for _, item := range items {
			if !e.toFinish(e.getTaskNode(ctx, item.GetName()), holders) {
				continue
			}
			dt, err := e.desiredTask(ctx, item)
			if err != nil {
				log.WithError(err).Warn(ctx, "cannot finish a completed task")
				continue
			}
			desired = append(desired, dt)
		}
		if err := e.reconciler.Reconcile(ctx, desired); err != nil {
			log.WithError(err).Warn(ctx, "failed to finish a completed task")
		}
	}
}

// resolveItems resolves task against its scope, as it was dispatched, and,
// with expand, expands it into its items; without, it is its own one item.
func (e *Engine) resolveItems(ctx context.Context, task dag.Task, expand bool) ([]dag.Task, error) {
	scope, err := e.buildLocalScopeFromTask(ctx, task)
	if err != nil {
		return nil, err
	}
	resolved, err := e.resolveTask(ctx, task, scope)
	if err != nil {
		return nil, err
	}
	if !expand {
		return []dag.Task{resolved}, nil
	}
	return resolved.Expand(ctx, e.expansionScope(scope), e.woc)
}

// ranToCompletion reports whether node is a task node that has finished and
// ran: a Skipped or Omitted node holds no lock and owes no metrics.
func ranToCompletion(node *wfv1.NodeStatus) bool {
	return node != nil && node.Phase.Fulfilled(node.TaskResultSynced) && ran(node)
}

// toFinish reports whether node has run to completion and is still to be
// finished: it completed in this reconcile, or it still holds a lock (in
// holders, see lockHolders), as one completed outside the controller, such
// as a resumed suspend, does. A node that had completed when the reconcile
// started, and holds no lock, was finished then.
func (e *Engine) toFinish(node *wfv1.NodeStatus, holders map[string]bool) bool {
	if !ranToCompletion(node) {
		return false
	}
	prev, ok := e.woc.preExecutionNodeStatuses[node.ID]
	return !ok || !ranToCompletion(&prev) || holders[node.ID]
}

// hasNodeToFinish reports whether node, a task's node, or one of its items
// for a TaskGroup, is to be finished (toFinish).
func (e *Engine) hasNodeToFinish(node *wfv1.NodeStatus, holders map[string]bool) bool {
	if node.Type != wfv1.NodeTypeTaskGroup {
		return e.toFinish(node, holders)
	}
	for _, item := range e.itemNodes(node) {
		if e.toFinish(item, holders) {
			return true
		}
	}
	return false
}

// lockHolders returns the IDs of the nodes of wf that hold a mutex or a
// semaphore, as its synchronization status records them: a node's holder key
// ends with its ID ("<namespace>/<workflow>/<node ID>", or the node ID alone
// in the oldest form).
func lockHolders(wf *wfv1.Workflow) map[string]bool {
	holders := make(map[string]bool)
	sync := wf.Status.Synchronization
	if sync == nil {
		return holders
	}
	add := func(key string) { holders[key[strings.LastIndex(key, "/")+1:]] = true }
	if sync.Mutex != nil {
		for _, h := range sync.Mutex.Holding {
			add(h.Holder)
		}
	}
	if sync.Semaphore != nil {
		for _, h := range sync.Semaphore.Holding {
			for _, key := range h.Holders {
				add(key)
			}
		}
	}
	return holders
}

// mayNeedFinishing reports whether a fulfilled node of task can have
// anything to finish: its template takes a lock or has metrics. A template
// that resolves only with the task (a templateRef built from outputs) is
// assumed to. Its globalName outputs need no re-entry: every path that
// fulfils a node exports them (exportCompletedNodes, handleNodeFulfilled).
func (e *Engine) mayNeedFinishing(ctx context.Context, task dag.Task) bool {
	_, tmpl, stored, err := e.tmplCtx.ResolveTemplate(ctx, task.GetTemplateReferenceHolder())
	if err != nil || e.woc.mergedTemplateDefaultsInto(tmpl) != nil {
		return true
	}
	e.woc.updated = e.woc.updated || stored
	return tmpl.Synchronization != nil || tmpl.Metrics != nil
}

// finalize assesses the overall phase and, if terminal, sets outputs,
// saves memoization cache, and marks the node Succeeded/Failed/Error.
func (e *Engine) finalize(ctx context.Context, tasks []dag.Task, onExitCompleted bool) error {
	// Under a Stop shutdown the boundary is failed once its exit handlers are
	// done — unless this boundary IS an onExit handler, which must be allowed
	// to complete (#16488).
	phase, message := e.assessDAGPhase(ctx, tasks, e.woc.GetShutdownStrategy().Enabled() && onExitCompleted && !e.onExitTemplate)
	if e.hookErr != nil && !phase.FailedOrError() {
		// A hook error ends the boundary Error, without outputs or
		// memoization, once nothing it has started is still running.
		if e.running(ctx, tasks) {
			return nil
		}
		phase, message = wfv1.NodeError, e.hookErr.Error()
	}

	switch phase {
	case wfv1.NodeRunning:
		return nil
	case wfv1.NodeError, wfv1.NodeFailed:
		// Wait for any in-flight (non-fulfilled) hook child nodes before
		// transitioning the boundary terminal. Errored hooks are fulfilled
		// and don't block; only Running/Pending hook nodes gate the boundary.
		// This is required because markWorkflowFailed sets the
		// `completed=true` label, after which the controller's
		// reconciliationNeeded filter (controller.go) skips future workqueue
		// events for the workflow — including the pod-completion events that
		// would otherwise advance the hook nodes.
		if e.findTaskHook(ctx, tasks, func(n *wfv1.NodeStatus) bool { return !n.Fulfilled() }) != nil {
			return nil
		}
	default:
		if !onExitCompleted {
			return nil
		}
	}

	// Outbound nodes before outputs: a boundary that ends Error because its
	// own outputs cannot resolve still links its dependants.
	if err := e.updateOutboundNodesForTargetTasks(ctx, e.evaluator.GetTargetTasks(ctx)); err != nil {
		return err
	}
	if phase.FailedOrError() {
		// Surface the failure message on the boundary. This message bubbles
		// up to the workflow status (operator.go uses entry node.Message for
		// workflow.status.Message), and callers / tests rely on it to identify
		// which child triggered the failure (e.g. TestNodeSuspendResume).
		_ = e.woc.markNodePhase(ctx, e.nodeName, phase, message)
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
	_ = e.woc.markNodePhase(ctx, e.nodeName, wfv1.NodeSucceeded)
	return nil
}

// boundaryFailureMessage names the first failed task of a DAG in declaration
// order, "child '<task-id>' failed": the message bubbles up to the workflow
// status. Task nodes are named, never their retry attempts (failedNode is a
// no-op for those); an expanded task's TaskGroup is resolved to its first
// failed item instead, as for Steps (see failedNode). Walking the template
// rather than wf.Status.Nodes keeps the message stable between cycles.
// Returns "" if no task failed.
func (e *Engine) boundaryFailureMessage(ctx context.Context) string {
	for _, task := range e.tmpl.DAG.Tasks {
		if node := e.getTaskNode(ctx, task.Name); node != nil && node.FailedOrError() {
			return fmt.Sprintf("child '%s' failed", e.failedNode(node).ID)
		}
	}
	return ""
}

// findTaskHook returns the first hook node of tasks that matches: a hook of
// a task's node or, for an expanded task, of one of its items.
func (e *Engine) findTaskHook(ctx context.Context, tasks []dag.Task, match func(*wfv1.NodeStatus) bool) *wfv1.NodeStatus {
	for _, task := range tasks {
		taskNode := e.getTaskNode(ctx, task.GetName())
		if taskNode == nil {
			continue
		}
		owners := []*wfv1.NodeStatus{taskNode}
		if taskNode.Type == wfv1.NodeTypeTaskGroup {
			owners = append(owners, e.itemNodes(taskNode)...)
		}
		for _, owner := range owners {
			for _, hook := range common.HookNodes(owner, e.woc.wf.Status.Nodes) {
				if match(hook) {
					return hook
				}
			}
		}
	}
	return nil
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
	if taskNode != nil && taskNode.Fulfilled() {
		e.log.WithFields(logging.Fields{"task": taskName, "node": taskNodeName}).Debug(ctx, "task already fulfilled")
		return taskNode, nil
	}

	// The task's references are resolved once, from one scope, before
	// anything is created for it: a reference not in scope yet leaves the
	// task uncreated, so an expansion never leaves a childless TaskGroup. A
	// scope the template itself cannot build (templateError) ends the
	// boundary before the task's StepGroup is started, so no further group is
	// created.
	scope, err := e.buildLocalScopeFromTask(ctx, task)
	if stderrors.As(err, new(templateError)) {
		e.markBoundaryError(ctx, err)
		return nil, err
	}

	// A scope, resolution or expansion failure is this task's own terminal
	// outcome: it is recorded on an Error node linked under the task's
	// parents, so siblings keep running and the boundary rolls up from its
	// children.
	parents := e.parentsFor(ctx, taskName)
	failTask := func(err error) (*wfv1.NodeStatus, error) {
		e.initTerminalErrorNode(ctx, task, parents, err)
		return e.getTaskNode(ctx, taskName), err
	}
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

	if resolved.ShouldExpand() {
		expandedTasks, expandErr := resolved.Expand(ctx, e.expansionScope(scope), e.woc)
		if expandErr != nil {
			return failTask(expandErr)
		}

		// Empty expansion (e.g., withParam resolves to []) → skip the task. A
		// group that already exists and now expands to no items (after `argo
		// retry --parameter`) is Skipped too where its phase allows it; a
		// Running group cannot become Skipped, so it is completed Succeeded,
		// with the same message.
		if len(expandedTasks) == 0 {
			if taskNode == nil {
				return e.initTaskNode(ctx, resolved, parents, wfv1.NodeTypeSkipped, wfv1.NodeSkipped, "Skipped, empty params"), nil
			}
			phase := wfv1.NodeSkipped
			if !isValidPhaseTransition(taskNode.Phase, phase) {
				phase = wfv1.NodeSucceeded
			}
			return e.woc.markNodePhase(ctx, taskNodeName, phase, "Skipped, empty params"), nil
		}

		tgNode := taskNode
		if tgNode == nil {
			tgNode = e.initTaskNode(ctx, resolved, parents, wfv1.NodeTypeTaskGroup, wfv1.NodeRunning)
			e.adoptItems(ctx, task, tgNode.Name, expandedTasks)
		}
		if err := e.reconcileTaskGroup(ctx, task, tgNode, expandedTasks); err != nil {
			return nil, err
		}
		return tgNode, nil
	}

	// Use reconciler for leaf task
	if err := e.reconcileTask(ctx, resolved, parents); err != nil {
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

// adoptItems moves the items of an expanded task that already have a node
// under its new TaskGroup node tgNodeName. An older controller gave an
// expanded step no TaskGroup: the items hung directly off their StepGroup,
// and a workflow it started still has them there. The TaskGroup is assessed,
// aggregates its outputs and is reconciled over its children, so it must hold
// every item, including those that finished before the upgrade. Each adopted
// item's edge from the task's parents is removed, so it keeps a single
// parent, as retry's graph walk expects. The adopted items are then finished
// as reconcileFulfilledTasks finishes every task's, since it ran before their
// TaskGroup existed: one that completed while the controller was down emits
// its metrics and releases its lock in this reconcile, the one that sees it
// complete, and dispatch skips it.
func (e *Engine) adoptItems(ctx context.Context, task dag.Task, tgNodeName string, expanded []dag.Task) {
	taskName := task.GetName()
	adopted := make(map[string]bool)
	for _, item := range expanded {
		if node := e.getTaskNode(ctx, item.GetName()); node != nil {
			adopted[node.ID] = true
			e.woc.addChildNode(ctx, tgNodeName, node.Name)
		}
	}
	if len(adopted) == 0 {
		return
	}
	for _, parentName := range e.parentsFor(ctx, taskName) {
		if parent, err := e.woc.wf.GetNodeByName(parentName); err == nil {
			parent.Children = slices.DeleteFunc(parent.Children, func(id string) bool { return adopted[id] })
			e.woc.wf.Status.Nodes.Set(ctx, parent.ID, *parent)
		}
	}
	e.reconcileFulfilledTasks(ctx, []dag.Task{task})
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

	dt, err := e.desiredTask(ctx, task)
	if err != nil {
		return failTask(err)
	}
	dt.ParentNodeNames = parents
	return &dt, nil
}

// desiredTask resolves task's template, with the templateDefaults merged in,
// and processes its arguments into the DesiredTask the reconciler executes,
// as reconcileTemplate does (prepareTemplate). task is resolved already (see
// resolveTask), so its arguments are resolved in one place, and a node is
// reconciled with the template it was created from, whether it is being
// dispatched or finished (reconcileFulfilledTasks).
func (e *Engine) desiredTask(ctx context.Context, task dag.Task) (DesiredTask, error) {
	taskNodeName := e.taskNodeName(task.GetName())
	// The name variable follows the boundary's kind and carries the task's
	// display name, where reconcileTemplate follows the template holder's: an
	// expanded step's items are DAG tasks named "[i].<item>", and see
	// {{steps.name}} as the item name alone.
	nameKey := varkeys.TasksName
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		nameKey = varkeys.StepsName
	}
	newTmplCtx, processedTmpl, err := e.woc.prepareTemplate(ctx, taskNodeName, task.GetTemplateReferenceHolder(), e.tmplCtx, task.GetArguments(), nameKey, task.GetDisplayName())
	if err != nil {
		return DesiredTask{}, err
	}

	return DesiredTask{
		TaskName:      taskNodeName,
		TemplateScope: e.tmplCtx.GetTemplateScope(),
		TmplCtx:       newTmplCtx,
		Template:      processedTmpl,
		TemplateRef:   task.GetTemplateReferenceHolder(),
		BoundaryID:    e.boundaryID,
		IsOnExit:      e.onExitTemplate,
	}, nil
}

// taskNodeName is the node name of a task in this boundary (dag.TaskNodeName).
func (e *Engine) taskNodeName(taskName string) string {
	return dag.TaskNodeName(e.nodeName, taskName)
}

// taskNodeID is the node ID of a task in this boundary (dag.TaskNodeID).
func (e *Engine) taskNodeID(taskName string) string {
	return dag.TaskNodeID(e.woc.wf, e.nodeName, taskName)
}

// getTaskNode returns the node status of a task (dag.TaskNode).
func (e *Engine) getTaskNode(ctx context.Context, taskName string) *wfv1.NodeStatus {
	node := dag.TaskNode(e.woc.wf, e.nodeName, taskName)
	if node == nil {
		e.log.WithFields(logging.Fields{"nodeID": e.taskNodeID(taskName), "taskName": taskName}).Debug(ctx, "was unable to obtain the node")
	}
	return node
}

// assessDAGPhase assesses the boundary's phase, and the message that explains
// a failure. A Steps template is assessed group by group (assessStepGroups).
// A DAG is Running while any node it has created has not finished (see
// outcome); then its targets (dag.target in the order written, else the leaf
// tasks) decide: a target whose branch failed without continueOn fails the
// DAG, the first such target when failFast; a target with no node keeps it
// Running unless failFast and another target failed.
func (e *Engine) assessDAGPhase(ctx context.Context, tasks []dag.Task, isShutdown bool) (wfv1.NodePhase, string) {
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		phase, message := e.assessStepGroups(ctx)
		if isShutdown {
			phase = wfv1.NodeFailed
		}
		return phase, message
	}
	if isShutdown {
		return wfv1.NodeFailed, e.boundaryFailureMessage(ctx)
	}

	if e.running(ctx, tasks) {
		return wfv1.NodeRunning, ""
	}
	failFast := e.tmpl.DAG.FailFast == nil || *e.tmpl.DAG.FailFast
	phase := wfv1.NodeSucceeded
	memo := make(map[string]wfv1.NodePhase)
	for _, name := range e.evaluator.GetTargetTasks(ctx) {
		if e.getTaskNode(ctx, name) == nil {
			phase = wfv1.NodeRunning
			if !failFast {
				break
			}
			continue
		}
		if branch := e.branchPhase(ctx, name, memo); branch.FailedOrError() && !e.evaluator.GetTask(name).ContinuesOn(branch) {
			phase = branch
			if failFast {
				break
			}
		}
	}
	if phase.FailedOrError() {
		return phase, e.boundaryFailureMessage(ctx)
	}
	return phase, ""
}

// running reports whether a node of tasks has not finished (see outcome).
func (e *Engine) running(ctx context.Context, tasks []dag.Task) bool {
	for _, task := range tasks {
		if node := e.getTaskNode(ctx, task.GetName()); node != nil {
			if _, done := e.outcome(node); !done {
				return true
			}
		}
	}
	return false
}

// branchPhase is the phase a DAG task passes down its branch: its own phase
// once it has completed (Succeeded, Failed or Error), otherwise (Skipped,
// Omitted, a running daemon) the worst phase of the branches it hangs off.
func (e *Engine) branchPhase(ctx context.Context, name string, memo map[string]wfv1.NodePhase) wfv1.NodePhase {
	if phase, ok := memo[name]; ok {
		return phase
	}
	memo[name] = wfv1.NodeSucceeded // ends a cycle, which validation rejects
	phase := wfv1.NodeSucceeded
	if node := e.getTaskNode(ctx, name); node != nil {
		phase, _ = e.outcome(node)
	}
	if !phase.Completed() {
		phase = wfv1.NodeSucceeded
		deps, _ := e.evaluator.GetDependencies(ctx, name)
		for _, dep := range deps {
			phase = dag.WorsePhase(phase, e.branchPhase(ctx, dep, memo))
		}
	}
	memo[name] = phase
	return phase
}

// outcome reports the phase a task's node has finished with, and whether it
// has finished: the node is fulfilled (a running daemon counts, as it does
// for dependants) and none of its hooks is still running. A TaskGroup has
// finished once reconcileTaskGroup has recorded it, which it does only when
// every item exists and has finished; an item can still fail after that (a
// daemon that dies) and fails the group with it, and an item's hook can
// still be running.
func (e *Engine) outcome(node *wfv1.NodeStatus) (wfv1.NodePhase, bool) {
	nodes := e.woc.wf.Status.Nodes
	if node.Type != wfv1.NodeTypeTaskGroup || !node.Phase.Fulfilled(nil) {
		return node.Phase, node.Fulfilled() && common.CheckAllHooksFullfilled(node, nodes)
	}
	phase := node.Phase
	for _, item := range e.itemNodes(node) {
		if !common.CheckAllHooksFullfilled(item, nodes) {
			return phase, false
		}
		phase = dag.WorsePhase(phase, item.Phase)
	}
	return phase, common.CheckAllHooksFullfilled(node, nodes)
}

// parentsFor returns the nodes a task's node hangs off in the graph. Steps
// tasks are children of their StepGroup node, which is started here when the
// group's first step is given a node (see startStepGroup): every path that
// creates a step's node comes through here. DAG tasks are children of the
// outbound nodes of their dependencies, or of the boundary node when they
// have none. The walk visits a task's dependencies before it, so each has
// its node by then; a dependency with no node is skipped.
func (e *Engine) parentsFor(ctx context.Context, taskName string) []string {
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		if i, ok := stepGroupIndexOf(taskName); ok {
			return []string{e.startStepGroup(ctx, i)}
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

// startStepGroup returns the name of the StepGroup node for group i, creating
// it when the group starts: when a step of it is first given a node
// (dispatched, skipped, errored or Omitted) or, for an empty group, once the
// group before it has finished. Group 0 hangs off the Steps node, and group i
// off the outbound nodes of group i-1's children, or off group i-1 itself
// when it has none. A step of group i only gets a node once every step of
// group i-1 has finished and the walk has closed group i-1; an empty group
// i-1 has no step for the walk to close it at, so it is started and closed
// here first. The link is therefore made once, complete. A later group whose
// steps are recorded Omitted (after a Stop or a deadline) exists and ends
// Omitted; a group that nothing reaches (after failFast ends the Steps node)
// never exists.
func (e *Engine) startStepGroup(ctx context.Context, i int) string {
	name := e.stepGroupNodeNameAt(i)
	if _, err := e.woc.wf.GetNodeByName(name); err == nil {
		return name
	}
	parents := []string{e.nodeName}
	if i > 0 {
		prevName := e.startStepGroup(ctx, i-1)
		if len(e.tmpl.Steps[i-1].Steps) == 0 {
			e.assessStepGroup(ctx, i-1)
		}
		parents = []string{prevName}
		if prev, err := e.woc.wf.GetNodeByName(prevName); err == nil && len(prev.Children) > 0 {
			parents = nil
			for _, childID := range prev.Children {
				for _, outID := range e.woc.getOutboundNodes(ctx, childID) {
					if outNode, err := e.woc.wf.Status.Nodes.Get(outID); err == nil {
						parents = append(parents, outNode.Name)
					}
				}
			}
		}
	}
	e.woc.initializeNode(ctx, name, wfv1.NodeTypeStepGroup, e.tmplCtx.GetTemplateScope(), &wfv1.WorkflowStep{}, e.boundaryID, wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	for _, parent := range parents {
		e.woc.addChildNode(ctx, parent, name)
	}
	return name
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

// itemNodes returns the item nodes of tg, a TaskGroup node
// (dag.TaskGroupItems); a child missing from the node map is left out.
func (e *Engine) itemNodes(tg *wfv1.NodeStatus) []*wfv1.NodeStatus {
	items, _ := dag.TaskGroupItems(e.woc.wf.Status.Nodes, tg)
	return items
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
		if err := e.woc.processAggregateNodeOutputs(scope, agg, refName, e.itemNodes(node)); err != nil {
			if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
				// A finished step's items are aggregated into the Steps
				// template's own scope, so this is the template's error.
				return templateError{err}
			}
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

// scopeNodeForTask returns the node whose fields ({{steps.X.id}}, .status,
// ...) represent taskName in a scope. An expanded Steps step exposes the
// enclosing StepGroup node, as it did when Steps had no TaskGroup node, for
// compatibility: the TaskGroup node only feeds the aggregate outputs. DAG
// tasks expose their own (TaskGroup) node.
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
	// Add all ancestor tasks' outputs to scope (transitive closure of
	// dependencies). A task may reference outputs from any ancestor, not just
	// direct dependencies (e.g., {{tasks.grandparent.ip}} in a DAG). A step's
	// ancestors are every step of every earlier group, because executeSteps
	// (steps.go) makes each group's steps depend on the steps of the last
	// non-empty group before it, so a step can reference any earlier group.
	// Ancestors are added by step name, which relies on validation's rule that
	// step names are unique across the whole template.
	ancestorNames, err := e.evaluator.GetAncestors(ctx, task.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to get ancestors for task %s: %w", task.GetName(), err)
	}
	for _, depName := range ancestorNames {
		if err := e.addTaskToScope(ctx, scope, depName); err != nil {
			return nil, err
		}
	}

	// Add workflow-level global outputs to scope so that references like
	// {{workflow.outputs.artifacts.my-art}} and {{workflow.outputs.parameters.my-param}}
	// can be resolved. These are populated by addOutputsToGlobalScope during execution.
	e.woc.addWorkflowOutputsToLocalScope(e.woc.wf.Status.Outputs, scope)

	return scope, nil
}

// addTaskToScope adds a task's part to scope (addTaskNodeToScope) under the
// name templates refer to it by, {{tasks.<task>}} or {{steps.<step>}}. A task
// with no node yet (e.g. outside dag.target's ancestry) adds nothing.
func (e *Engine) addTaskToScope(ctx context.Context, scope *wfScope, taskName string) error {
	node := e.getTaskNode(ctx, taskName)
	if node == nil {
		return nil
	}
	steps := e.tmpl.GetType() == wfv1.TemplateTypeSteps
	ref, agg, refName := varkeys.TasksNodeRef, varkeys.TasksAggregate, taskName
	if steps {
		ref, agg, refName = varkeys.StepsNodeRef, varkeys.StepsAggregate, stepNameOf(taskName)
	}
	// Steps keeps skipped-node artifact placeholders resolvable (includeArtifacts); DAG
	// leaves them to resolveArtifactArguments' optional-drop / required-error handling.
	return e.addTaskNodeToScope(ctx, scope, ref, agg, refName, taskName, node, steps)
}

// setDAGOutputs sets the outputs of the DAG.
func (e *Engine) setDAGOutputs(ctx context.Context) error {
	node, err := e.woc.wf.GetNodeByName(e.nodeName)
	if err != nil {
		return err
	}
	scope := createScope(e.tmpl)
	// Seed the scope with the workflow's own outputs so that a DAG or Steps
	// template's own output params/artifacts can read {{workflow.outputs.*}}.
	e.woc.addWorkflowOutputsToLocalScope(e.woc.wf.Status.Outputs, scope)

	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		for i, stepGroup := range e.tmpl.Steps {
			for _, step := range stepGroup.Steps {
				if err = e.addTaskToScope(ctx, scope, stepTaskNameFor(i, step.Name)); err != nil {
					return err
				}
			}
		}
	} else if e.tmpl.DAG != nil {
		for _, task := range e.tmpl.DAG.Tasks {
			if err = e.addTaskToScope(ctx, scope, task.Name); err != nil {
				return err
			}
		}
	}

	outputs, err := e.woc.getTemplateOutputsFromScope(ctx, e.tmpl, scope)
	if err != nil {
		return err
	}
	if outputs != nil {
		// Exported when the node is fulfilled (handleNodeFulfilled).
		node.Outputs = outputs
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
// when), and resolves its artifact arguments' from/fromExpression.
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
		if err != nil && !task.ShouldExpand() {
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
