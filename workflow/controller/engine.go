package controller

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Knetic/govaluate"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/template"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
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
	}
}

// Execute orchestrates the execution of a DAG or Steps template.
// It delegates to phase methods that each handle one concern.
// Errors are handled internally by marking the boundary node with the
// appropriate phase (Failed for Steps, Error for DAGs).
func (e *Engine) Execute(ctx context.Context, tasks []dag.Task) {
	e.evaluator = dag.NewDAGEvaluatorFromTasks(e.woc.wf, tasks, e.tmpl, e.boundaryID, e.nodeName)

	// Provide retry strategies to the evaluator
	e.populateRetryStrategies(ctx, tasks)

	e.reconcileDaemonedTasks(ctx, tasks)

	// First hooks pass: process hooks for tasks completed in previous operate cycles.
	// processHooks returns nil error (per-task errors are isolated and logged inside).
	// onExitCompleted is false if any exit handler is still pending or any hook errored.
	onExitCompleted, _ := e.processHooks(ctx, tasks)

	// Fixed-point iteration: evaluate all tasks, dispatch any that need execution,
	// repeat until no new task is executed in a pass. The loop exists because some
	// tasks complete instantly (cache hits, omissions, no-op task groups) and may
	// unblock dependents in the same operate cycle.
	//
	// Termination: every non-terminating iteration must add a task name to
	// executedTasks (that's what anyNew tracks). executedTasks grows monotonically
	// and its membership is bounded by the unique task names that EvaluateAll can
	// emit — finite, since static tasks and their expansions are finite. So the
	// loop converges in at most O(unique-task-names) iterations.
	executedTasks := make(map[string]bool)
	for {
		results := e.evaluateAll(ctx)
		newExecuted, err := e.converge(ctx, tasks, results)
		if err != nil {
			e.markBoundaryError(ctx, err)
			return
		}
		// Assess TaskGroup nodes created during this iteration so that
		// downstream tasks see them as fulfilled in the next iteration.
		e.assessTaskGroups(ctx, tasks)
		anyNew := false
		for k, v := range newExecuted {
			if v && !executedTasks[k] {
				anyNew = true
				executedTasks[k] = true
			}
		}
		if !anyNew {
			break
		}
	}

	// Second hooks pass: process hooks for tasks that became fulfilled during scheduling.
	// This ensures exit handlers trigger in the same operate cycle as task completion.
	// Without this, the Steps/DAG boundary gets marked fulfilled by finalize, and
	// subsequent operate cycles skip the engine entirely (handleNodeFulfilled returns early).
	// A task's exit handler driven by the first pass is only inspected here (not re-run);
	// see hookHandler.exitDriven (#14392 / PR #16088).
	// processHooks returns nil error (per-task errors are isolated and logged inside).
	hooksDone, _ := e.processHooks(ctx, tasks)

	onExitCompleted = onExitCompleted && hooksDone

	e.reconcileExternalCompletions(ctx, tasks, executedTasks)

	e.assessStepGroups(ctx)

	results := e.createOmittedNodes(ctx, tasks)

	if err := e.finalize(ctx, tasks, results, onExitCompleted); err != nil {
		e.markBoundaryError(ctx, err)
	}
}

// markBoundaryError marks the boundary node with an appropriate error phase.
// For Steps templates, uses Failed (not Error) to match legacy behavior.
func (e *Engine) markBoundaryError(ctx context.Context, err error) {
	node, _ := e.woc.wf.GetNodeByName(e.nodeName)
	if node != nil && node.Fulfilled() {
		// Already fulfilled — remap Error→Failed for Steps if needed
		if e.tmpl.GetType() == wfv1.TemplateTypeSteps && node.Phase == wfv1.NodeError {
			e.woc.markNodePhase(ctx, e.nodeName, wfv1.NodeFailed, node.Message)
		}
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
		}
	}
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
				continue // Skip executeTask — converge will pick it up now
			}
		}

		e.log.Info(ctx, fmt.Sprintf("reconciling daemoned task %s", task.GetName()))
		if _, err := e.executeTask(ctx, task, true); err != nil {
			e.log.WithError(err).Error(ctx, "failed to reconcile daemoned task")
		}
	}
}

// processHooks runs lifecycle hooks and exit handlers for all tasks.
// Returns whether all exit handlers have completed. Per-task hook errors
// are isolated to the failing task node (not the boundary), mirroring the
// legacy controller's executeDAGTask behavior — a single bad hook on one
// task must not abort sibling tasks or the DAG/Steps boundary.
func (e *Engine) processHooks(ctx context.Context, tasks []dag.Task) (bool, error) {
	return e.hooks.ProcessAllTaskHooks(ctx, tasks,
		e.getTaskNode,
		e.buildLocalScopeFromTask,
		func(ctx context.Context, taskNode *wfv1.NodeStatus, err error) {
			// Mark the offending task node Errored, not the boundary. Siblings
			// continue to be processed in the same operate cycle.
			e.woc.markNodeError(ctx, taskNode.Name, err)
		},
	)
}

// assessTaskGroups transitions TaskGroup nodes (from withItems/withParam/withSequence)
// to a terminal phase once all their expanded children have completed.
func (e *Engine) assessTaskGroups(ctx context.Context, tasks []dag.Task) {
	for _, task := range tasks {
		if !dag.HasExpansion(task) {
			continue
		}
		taskNodeName := e.taskNodeName(task.GetName())
		tgNode, err := e.woc.wf.GetNodeByName(taskNodeName)
		if err != nil || tgNode.Type != wfv1.NodeTypeTaskGroup || tgNode.Fulfilled() {
			continue
		}
		e.assessTaskGroupPhase(ctx, tgNode)
	}
}

// assessStepGroups transitions StepGroup nodes to a terminal phase once all their
// step tasks have completed. Unlike assessTaskGroupPhase, this handles per-step
// continueOn semantics — each step in a group can have its own continueOn setting.
// Step child nodes are looked up by constructing names from the template definition
// (not from node.Children) because not all steps may have nodes yet if parallelism
// limits prevented scheduling.
func (e *Engine) assessStepGroups(ctx context.Context) {
	if e.tmpl.GetType() != wfv1.TemplateTypeSteps || e.tmpl.Steps == nil {
		return
	}
	for i, stepGroup := range e.tmpl.Steps {
		sgNodeName := fmt.Sprintf("%s[%d]", e.nodeName, i)
		sgNode, err := e.woc.wf.GetNodeByName(sgNodeName)
		if err != nil || sgNode.Fulfilled() {
			continue
		}

		isPending := false
		isRunning := false
		isFailed := false
		isSucceeded := true
		// Track first failing child ID to surface in the StepGroup's failure
		// message, matching pre-refactor executeStepGroup semantics. The message
		// `child '<id>' failed` bubbles up through the Steps node to the workflow
		// status and is what callers / tests (e.g. TestNodeSuspendResume) inspect
		// to identify which leaf failed.
		failingChildID := ""

		for _, step := range stepGroup.Steps {
			childNodeName := fmt.Sprintf("%s[%d].%s", e.nodeName, i, step.Name)
			childNode, err := e.woc.wf.GetNodeByName(childNodeName)
			if err != nil {
				isPending = true
				isSucceeded = false
				continue
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

		// Default to Running; only a clean (non-pending, non-running) success or
		// an outright failure moves the StepGroup off Running.
		newPhase := wfv1.NodeRunning
		var newMessage string
		if isFailed {
			// Always use Failed for FailedOrError children, matching old executeStepGroup behavior.
			newPhase = wfv1.NodeFailed
			if failingChildID != "" {
				newMessage = fmt.Sprintf("child '%s' failed", failingChildID)
			}
		} else if isSucceeded && !isPending && !isRunning {
			newPhase = wfv1.NodeSucceeded
		}

		if sgNode.Phase != newPhase {
			e.woc.markNodePhase(ctx, sgNodeName, newPhase, newMessage)
		}
	}
}

// converge applies evaluation results by performing side effects.
// Static DAG tasks route through executeTask; expanded TaskGroup children
// (e.g. "client(0:0)") route through dispatchTaskGroupChild so each per-item
// instance can be re-reconciled independently — needed when, say, a sync
// lock is released and a queued sibling needs another TryAcquire.
// The evaluator decides WHAT should happen; this layer just dispatches.

// isThrottleErr reports whether err is a deliberate throttling signal from
// the reconciler. These are not real failures — the caller should stop
// dispatching new work this pass but must not treat the situation as fatal.
func isThrottleErr(err error) bool {
	return stderrors.Is(err, ErrParallelismReached) ||
		stderrors.Is(err, ErrResourceRateLimitReached) ||
		stderrors.Is(err, ErrDeadlineExceeded) ||
		stderrors.Is(err, ErrTimeout)
}

// evaluateAll evaluates every task and, when the DAG specifies an explicit target,
// prunes the results to the target tasks plus their transitive ancestors. Without
// this, the engine schedules every dependency-ready task — including roots outside
// the target's ancestry — because EvaluateAll/converge operate on the full task set
// (a push-all-ready model). The legacy executeDAG instead pulled execution
// recursively from the targets, so unrelated roots were never visited. Pruning here
// keeps every downstream consumer (converge, createOmittedNodes, assessDAGPhase)
// consistent: an unscheduled-but-ready root would otherwise also keep the DAG
// Running forever via assessDAGPhase's pending check.
func (e *Engine) evaluateAll(ctx context.Context) map[string]dag.EvaluationResult {
	results := e.evaluator.EvaluateAll(ctx)
	set := e.executableTaskSet(ctx)
	if set == nil {
		return results
	}
	for k, r := range results {
		// Expanded TaskGroup children carry the static parent in ParentTaskName;
		// gate them by the parent's membership.
		name := r.TaskName
		if r.ParentTaskName != "" {
			name = r.ParentTaskName
		}
		if !set[name] {
			delete(results, k)
		}
	}
	return results
}

// executableTaskSet returns the set of task names eligible for execution given the
// DAG's target: the union of the target tasks and their transitive ancestors.
// Returns nil when no explicit target is set (a DAG without target, or a Steps
// template), meaning "no filtering — every task is eligible".
func (e *Engine) executableTaskSet(ctx context.Context) map[string]bool {
	if e.tmpl.DAG == nil || e.tmpl.DAG.Target == "" {
		return nil
	}
	set := make(map[string]bool)
	for _, target := range e.evaluator.GetTargetTasks(ctx) {
		set[target] = true
		ancestors, err := e.evaluator.GetAncestors(ctx, target)
		if err != nil {
			continue
		}
		for _, ancestor := range ancestors {
			set[ancestor] = true
		}
	}
	return set
}

func (e *Engine) converge(ctx context.Context, tasks []dag.Task, results map[string]dag.EvaluationResult) (map[string]bool, error) {
	executedTasks := make(map[string]bool)
	var firstErr error
	// Sort result keys for deterministic dispatch order. Map iteration would
	// otherwise vary per cycle and, under parallelism limits, the winner of a
	// limited slot becomes random — breaking reproducibility.
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		result := results[k]
		needsExecution := result.Action == dag.ActionExecute || result.ShouldRun ||
			result.Action == dag.ActionSucceed || result.Action == dag.ActionFail

		// Retry nodes with a daemoned child (ActionNone + FulfilledForDeps):
		// still need executeTask so processNodeRetries propagates the Daemoned
		// flag from child to parent.
		if !needsExecution && result.Action == dag.ActionNone {
			node := e.getTaskNode(ctx, result.TaskName)
			if node != nil && node.Type == wfv1.NodeTypeRetry && node.Phase == wfv1.NodeRunning {
				needsExecution = true
			}
		}

		if needsExecution {
			executedTasks[result.TaskName] = true
			if result.ParentTaskName != "" {
				if err := e.dispatchTaskGroupChild(ctx, tasks, result.ParentTaskName, result.TaskName); err != nil {
					if isThrottleErr(err) {
						// Deliberate throttling — don't fail, just stop dispatching this pass.
						return executedTasks, nil
					}
					// Per-task errors are already marked on the failing task node
					// by postExecutionHandling / markNodeError. Remember the first
					// error so the caller can mark the boundary AFTER every sibling
					// has had its chance to dispatch in this converge pass.
					if firstErr == nil {
						firstErr = err
					}
					e.log.WithError(err).WithField("task", result.TaskName).Warn(ctx, "task group child dispatch failed; continuing to allow sibling tasks")
					continue
				}
			} else if task := e.getTaskByName(tasks, result.TaskName); task != nil {
				if _, err := e.executeTask(ctx, task, true); err != nil {
					if isThrottleErr(err) {
						return executedTasks, nil
					}
					if firstErr == nil {
						firstErr = err
					}
					e.log.WithError(err).WithField("task", result.TaskName).Warn(ctx, "task execution failed; continuing to allow sibling tasks")
					continue
				}
			}
		}

		if result.RequeueAfter > 0 {
			e.woc.requeueAfter(result.RequeueAfter)
		}
	}
	return executedTasks, firstErr
}

// dispatchTaskGroupChild reconciles a single expanded TaskGroup child by
// re-expanding the static parent against the current scope and forwarding the
// matching expanded entry to the reconciler. The TaskGroup parent node already
// exists from initial expansion, so no parent linkage is needed.
func (e *Engine) dispatchTaskGroupChild(ctx context.Context, tasks []dag.Task, parentTaskName, childTaskName string) error {
	parentTask := e.getTaskByName(tasks, parentTaskName)
	if parentTask == nil {
		return nil
	}
	expanded, err := e.expandTask(ctx, parentTask)
	if err != nil {
		return err
	}
	return e.reconcileExpanded(ctx, expanded, "", func(t dag.Task) bool {
		return t.GetName() == childTaskName
	})
}

// gateExpansionAbsentOptional reproduces main's resolveReferences nil semantics for the
// withParam/withSequence expansion fields. The dag.Substitutor path flattens scope to a string map
// (getParameters), which DROPS the nil markers for skipped/omitted outputs with no default — so a
// withParam referencing such an absent optional would survive substitution as a literal "{{...}}" and
// fail later with a misleading "could not be parsed as a JSON list" error instead of the terminal
// "absent optional" error main raises. Here we substitute those fields against the nil-preserving
// scope with steps/tasks strict (exactly as main's ReplaceStrictAny over the step body): a reference
// to an absent optional (present-but-nil) is a terminal error. A genuinely missing variable
// (IsMissingVariableErr) is left to the existing expansion path — dependency ordering means a real
// producer output is already present by expansion time, and this preserves current branch behavior.
//
// ponytail: validation gate only — the resolved text is discarded; for non-nil refs the existing
// Expand re-resolves identically (those keys are present in both scope views). withItems values are
// not covered (no test, unusual shape); add the field here if that ever needs main parity.
func (e *Engine) gateExpansionAbsentOptional(ctx context.Context, task dag.Task, scope *wfScope) error {
	if task.GetWithParam() == "" && task.GetWithSequence() == nil {
		return nil
	}
	fields := struct {
		WithParam    string         `json:"withParam,omitempty"`
		WithSequence *wfv1.Sequence `json:"withSequence,omitempty"`
	}{task.GetWithParam(), task.GetWithSequence()}
	b, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	if _, err := template.ReplaceStrictAny(ctx, string(b), scope.getParametersAny(e.woc.globalParams()), []string{"steps", "tasks"}); err != nil && !template.IsMissingVariableErr(err) {
		return err
	}
	return nil
}

// expandTask resolves parentTask's per-item expansion against the current scope.
// Pure read: no node creation, no reconciliation.
func (e *Engine) expandTask(ctx context.Context, parentTask dag.Task) ([]dag.Task, error) {
	scope, err := e.buildLocalScopeFromTask(ctx, parentTask)
	if err != nil {
		return nil, err
	}
	return parentTask.Expand(ctx, scope.getParameters(), e.woc)
}

// reconcileExpanded turns each accepted expansion into a DesiredTask and hands
// the batch to the reconciler. parentNodeName, when non-empty, is stamped on
// fresh desired tasks so they're linked to the TaskGroup node during initial
// creation; subsequent dispatches leave it empty (children already linked).
// A nil accept admits every expanded task.
func (e *Engine) reconcileExpanded(ctx context.Context, expanded []dag.Task, parentNodeName string, accept func(dag.Task) bool) error {
	var desired []DesiredTask
	for _, et := range expanded {
		if accept != nil && !accept(et) {
			continue
		}
		dts, err := e.createDesiredTask(ctx, et, false)
		if err != nil {
			return err
		}
		if parentNodeName != "" {
			for i := range dts {
				dts[i].ParentNodeNames = []string{parentNodeName}
			}
		}
		desired = append(desired, dts...)
	}
	if len(desired) == 0 {
		return nil
	}
	return e.reconciler.Reconcile(ctx, desired)
}

// reconcileExpandedChildren reconciles fulfilled children of a TaskGroup
// (withItems/withParam/withSequence). The parent task can't be reconciled directly
// because its arguments contain unresolved {{item.*}} tags. Instead, we walk each
// child and run it through the reconciler so postExecutionHandling fires (which
// releases sync locks and emits metrics). This matches main's behavior where
// executeTemplate is called for every expanded child on every cycle.
func (e *Engine) reconcileExpandedChildren(ctx context.Context, task dag.Task) {
	taskNode := e.getTaskNode(ctx, task.GetName())
	if taskNode == nil || taskNode.Type != wfv1.NodeTypeTaskGroup {
		return
	}
	newTmplCtx, resolvedTmpl, _, err := e.tmplCtx.ResolveTemplate(ctx, task.GetTemplateReferenceHolder())
	if err != nil {
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
	processedTmpl, err := common.ProcessArgs(ctx, resolvedTmpl, &args, e.woc.globalParams(), localParams, false, true, e.woc.wf.Namespace, e.woc.controller.configMapInformer.GetIndexer())
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
func (e *Engine) reconcileExternalCompletions(ctx context.Context, tasks []dag.Task, executedTasks map[string]bool) {
	for _, task := range tasks {
		// For expanded tasks (withItems/withParam/withSequence), we can't reconcile
		// the parent (unresolved {{item.*}} in arguments). Instead, reconcile each
		// fulfilled child individually to release sync locks and emit metrics.
		if dag.HasExpansion(task) {
			e.reconcileExpandedChildren(ctx, task)
			continue
		}
		// Skip tasks already processed by converge to avoid double metric emission
		// and redundant reconciliation.
		if executedTasks[task.GetName()] {
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

// createOmittedNodes creates Omitted workflow nodes for unreachable tasks.
// The scheduler marks them Omitted internally; we create corresponding workflow nodes
// so that downstream tasks and assessDAGPhase can see them.
// Returns the evaluation results for use by finalize.
func (e *Engine) createOmittedNodes(ctx context.Context, tasks []dag.Task) map[string]dag.EvaluationResult {
	results := e.evaluateAll(ctx)
	for _, task := range tasks {
		taskName := task.GetName()
		taskNodeName := e.taskNodeName(taskName)
		if _, err := e.woc.wf.GetNodeByName(taskNodeName); err == nil {
			continue
		}
		if result, ok := results[taskName]; ok && result.Skipped && !result.ShouldRun {
			e.woc.initializeNode(ctx, taskNodeName, wfv1.NodeTypeSkipped, e.tmplCtx.GetTemplateScope(), e.orgTmpl, e.boundaryID, wfv1.NodeOmitted, &wfv1.NodeFlag{}, true, "omitted: depends condition not met")
			e.addChildNode(ctx, task.GetName(), taskNodeName)
		}
	}
	return results
}

// finalize assesses the overall phase and, if terminal, sets outputs,
// saves memoization cache, and marks the node Succeeded/Failed/Error.
func (e *Engine) finalize(ctx context.Context, tasks []dag.Task, results map[string]dag.EvaluationResult, onExitCompleted bool) error {
	targetTasks := e.evaluator.GetTargetTasks(ctx)

	dagPhase := e.assessDAGPhase(ctx, tasks, results, e.woc.GetShutdownStrategy().Enabled() && onExitCompleted)

	switch dagPhase {
	case wfv1.NodeRunning:
		return nil
	case wfv1.NodeError, wfv1.NodeFailed:
		// Wait for any in-flight (non-fulfilled) hook child nodes before
		// transitioning the boundary terminal. Errored hooks ARE fulfilled
		// and don't block — that preserves the bug04 invariant (a single
		// failed hook must not abort siblings). Only Running/Pending hook
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
		_ = e.woc.markNodePhase(ctx, e.nodeName, phase, e.boundaryFailureMessage())
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
// boundary node when the engine transitions it to Failed/Error. Matches old
// executeSteps/executeDAG semantics:
//   - Steps: propagate the first failed StepGroup's message (which
//     assessStepGroups sets to "child '<leaf-id>' failed").
//   - DAG: build "child '<task-id>' failed" naming the first failed,
//     non-Hooked task node within this boundary.
//
// Nodes nested inside the boundary may not be direct children of the
// boundary's node.Children (e.g. Steps groups chain through one another), so
// scan wf.Status.Nodes by BoundaryID instead of walking node.Children.
// Returns "" if no failing in-boundary node is found.
func (e *Engine) boundaryFailureMessage() string {
	boundaryNode, err := e.woc.wf.GetNodeByName(e.nodeName)
	if err != nil {
		return ""
	}
	wantType := wfv1.NodeTypeStepGroup
	if e.tmpl.GetType() == wfv1.TemplateTypeDAG {
		wantType = ""
	}
	for _, n := range e.woc.wf.Status.Nodes {
		if n.BoundaryID != boundaryNode.ID {
			continue
		}
		if n.NodeFlag != nil && n.NodeFlag.Hooked {
			continue
		}
		if !n.FailedOrError() {
			continue
		}
		if wantType != "" {
			if n.Type != wantType {
				continue
			}
			// Propagate the StepGroup's already-formatted message verbatim.
			if n.Message != "" {
				return n.Message
			}
			continue
		}
		// DAG: name the failed task itself.
		return fmt.Sprintf("child '%s' failed", n.ID)
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
		for _, childID := range taskNode.Children {
			childNode, err := e.woc.wf.Status.Nodes.Get(childID)
			if err != nil {
				continue
			}
			if childNode.NodeFlag != nil && childNode.NodeFlag.Hooked && !childNode.Fulfilled() {
				return true
			}
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

func (e *Engine) executeTask(ctx context.Context, task dag.Task, addChild bool) (*wfv1.NodeStatus, error) {
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
		if err != nil {
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

	// build a local scope for the task
	scope, err := e.buildLocalScopeFromTask(ctx, task)
	if err != nil {
		return e.woc.markNodeError(ctx, taskNodeName, err), err
	}

	// Check the task's when clause to decide if it should execute.
	proceed, err := e.evaluateWhenClause(ctx, task, scope)
	if err != nil {
		e.woc.initializeNode(ctx, taskNodeName, wfv1.NodeTypeSkipped, e.tmplCtx.GetTemplateScope(), e.orgTmpl, e.boundaryID, wfv1.NodeError, &wfv1.NodeFlag{}, true, err.Error())
		if addChild {
			e.addChildNode(ctx, taskName, taskNodeName)
		}
		// Mark only the failing task node here; the boundary node is assessed by
		// the dispatch loop after every sibling has had its chance, so it rolls
		// up to Failed rather than being clobbered to a terminal Error.
		return e.woc.markNodeError(ctx, taskNodeName, err), err
	}

	if !proceed {
		if _, err = e.woc.wf.GetNodeByName(taskNodeName); err != nil {
			skipReason := fmt.Sprintf("when '%s' evaluated false", task.GetWhen())
			e.log.WithFields(logging.Fields{"childNodeName": taskNodeName, "skipReason": skipReason}).Info(ctx, "Skipping")
			e.woc.initializeNode(ctx, taskNodeName, wfv1.NodeTypeSkipped, e.tmplCtx.GetTemplateScope(), e.orgTmpl, e.boundaryID, wfv1.NodeSkipped, &wfv1.NodeFlag{}, true, skipReason)
			if addChild {
				e.addChildNode(ctx, taskName, taskNodeName)
			}
		}
		return nil, nil
	}

	// Expand withItems if necessary
	if dag.HasExpansion(task) {
		// A withParam/withSequence reference to an absent optional (skipped/omitted output, no
		// default) is terminal here, matching main; the string-map Substitutor below can't see it.
		if err = e.gateExpansionAbsentOptional(ctx, task, scope); err != nil {
			return e.woc.markNodeError(ctx, taskNodeName, err), err
		}
		expandedTasks, expandErr := task.Expand(ctx, scope.getParameters(), e.woc)
		if expandErr != nil {
			return e.woc.markNodeError(ctx, taskNodeName, expandErr), expandErr
		}

		// Empty expansion (e.g., withParam resolves to []) → skip the task
		if len(expandedTasks) == 0 {
			_, skipNode := e.woc.initializeNode(ctx, taskNodeName, wfv1.NodeTypeSkipped, e.tmplCtx.GetTemplateScope(), e.orgTmpl, e.boundaryID, wfv1.NodeSkipped, &wfv1.NodeFlag{}, true, "Skipped, empty params")
			if addChild {
				e.addChildNode(ctx, taskName, skipNode.Name)
			}
			return skipNode, nil
		}

		var tgNode *wfv1.NodeStatus
		if existingNode, lookupErr := e.woc.wf.GetNodeByName(taskNodeName); lookupErr == nil {
			tgNode = existingNode
		} else {
			_, tgNode = e.woc.initializeNode(ctx, taskNodeName, wfv1.NodeTypeTaskGroup, e.tmplCtx.GetTemplateScope(), e.orgTmpl, e.boundaryID, wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
			if addChild {
				e.addChildNode(ctx, taskName, tgNode.Name)
			}
		}

		if err = e.reconcileExpanded(ctx, expandedTasks, tgNode.Name, nil); err != nil {
			return nil, err
		}
		return tgNode, nil
	}

	// Use reconciler for leaf task
	desired, err := e.createDesiredTask(ctx, task, addChild)
	if err != nil {
		return nil, err
	}
	err = e.reconciler.Reconcile(ctx, desired)
	if err != nil {
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
	if node == nil && len(desired) > 0 {
		return nil, fmt.Errorf("task %s: %w", taskName, ErrReconcilerNoMaterialize)
	}
	return node, nil
}

// initTerminalErrorNode persists a task node in a terminal Error state when task setup fails before
// the node would normally be materialized (argument resolution / ProcessArgs / template resolution).
// The node must exist for markNodeError and converge's boundary handling to take effect; otherwise a
// setup failure — e.g. an unhandled absent optional (#16223) — silently vanishes and the workflow
// requeues forever instead of failing terminally. Mirrors the when-clause error path.
func (e *Engine) initTerminalErrorNode(ctx context.Context, taskNodeName string, parentNodeNames []string, err error) {
	// Only materialize the node when it doesn't already exist. In the omitted-dependency flow the
	// task node is created earlier in the cycle, and initializeNode panics ("already initialized")
	// on a second init — so re-initializing here would turn a terminal arg error into a recurring
	// "Workflow operation error" requeue loop. When it already exists, fall through to markNodeError.
	if node, getErr := e.woc.wf.GetNodeByName(taskNodeName); getErr != nil || node == nil {
		e.woc.initializeNode(ctx, taskNodeName, wfv1.NodeTypeSkipped, e.tmplCtx.GetTemplateScope(), e.orgTmpl, e.boundaryID, wfv1.NodeError, &wfv1.NodeFlag{}, true, err.Error())
		for _, parent := range parentNodeNames {
			e.woc.addChildNode(ctx, parent, taskNodeName)
		}
	}
	e.woc.markNodeError(ctx, taskNodeName, err)
}

// createDesiredTask helper to construct the struct from a dag.Task
func (e *Engine) createDesiredTask(ctx context.Context, task dag.Task, addChild bool) ([]DesiredTask, error) {
	taskName := task.GetName()
	taskNodeName := e.taskNodeName(taskName)

	// Check if already fulfilled
	if taskNode := e.getTaskNode(ctx, taskName); taskNode != nil && taskNode.Fulfilled() {
		return nil, nil
	}

	// Determine Parent Nodes
	var parentNodeNames []string
	if addChild {
		if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
			// For Steps templates, link to the StepGroup node
			if sgName := e.stepGroupNodeName(taskName); sgName != "" {
				parentNodeNames = []string{sgName}
			} else {
				parentNodeNames = []string{e.nodeName}
			}
		} else {
			// DAG templates: use dependency outbound nodes
			deps, err := e.evaluator.GetDependencies(ctx, taskName)
			switch {
			case err != nil:
				e.log.WithFields(logging.Fields{"taskName": taskName, "error": err}).Warn(ctx, "failed to get dependencies")
				parentNodeNames = []string{e.nodeName}
			case len(deps) > 0:
				for _, dep := range deps {
					depNodeName := e.taskNodeName(dep)
					depNodeID := e.woc.wf.NodeID(depNodeName)
					// Dep may not yet be in status: a peer that will be Omitted
					// but whose Omitted node is only created in createOmittedNodes,
					// after the converge loop. Skip linkage here; the next operate
					// cycle reconciles parent linkage once the Omitted node exists.
					if _, getErr := e.woc.wf.Status.Nodes.Get(depNodeID); getErr != nil {
						continue
					}
					outboundIDs := e.woc.getOutboundNodes(ctx, depNodeID)
					for _, outID := range outboundIDs {
						outNode, getErr := e.woc.wf.Status.Nodes.Get(outID)
						if getErr == nil {
							parentNodeNames = append(parentNodeNames, outNode.Name)
						}
					}
				}
			default:
				parentNodeNames = []string{e.nodeName}
			}
		}
	}

	// Build scope
	scope, err := e.buildLocalScopeFromTask(ctx, task)
	if err != nil {
		e.woc.markNodeError(ctx, taskNodeName, err)
		return nil, err
	}

	// Evaluate 'When' clause
	proceed, err := e.evaluateWhenClause(ctx, task, scope)
	if err != nil {
		e.woc.initializeNode(ctx, taskNodeName, wfv1.NodeTypeSkipped, e.tmplCtx.GetTemplateScope(), e.orgTmpl, e.boundaryID, wfv1.NodeError, &wfv1.NodeFlag{}, true, err.Error())
		for _, parent := range parentNodeNames {
			e.woc.addChildNode(ctx, parent, taskNodeName)
		}
		e.woc.markNodeError(ctx, taskNodeName, err)
		e.woc.markNodeError(ctx, e.nodeName, err)
		return nil, err
	}

	if !proceed {
		skipReason := fmt.Sprintf("when '%s' evaluated false", task.GetWhen())
		return []DesiredTask{{
			TaskName:         taskNodeName,
			OriginalTaskName: taskName,
			TemplateScope:    e.tmplCtx.GetTemplateScope(),
			TemplateRef:      task.GetTemplateReferenceHolder(),
			BoundaryID:       e.boundaryID,
			IsOnExit:         e.onExitTemplate,
			Skipped:          true,
			SkipReason:       skipReason,
			ParentNodeNames:  parentNodeNames,
		}}, nil
	}

	// Resolve Template and Arguments
	newTmplCtx, resolvedTmpl, templateStored, err := e.tmplCtx.ResolveTemplate(ctx, task.GetTemplateReferenceHolder())
	if err != nil {
		e.woc.markNodeError(ctx, taskNodeName, err)
		return nil, err
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
		e.woc.markNodeError(ctx, taskNodeName, err)
		return nil, err
	}

	// Process Arguments
	args := task.GetArguments()

	// Resolve argument parameter and artifact references against the scope.
	// This substitutes {{steps.X.outputs.*}} / {{tasks.X.outputs.*}} in argument values
	// and resolves artifact from/fromExpression to concrete storage locations.
	// Arguments are resolved here (not via ProcessArgs localParams) so that scope-level
	// references don't leak into the child template body via SubstituteParams — which
	// would cause bugs like parent step outputs being substituted into recursive
	// template when-clauses.
	args, err = scope.resolveArguments(ctx, args, e.woc.globalParams())
	if err != nil {
		e.initTerminalErrorNode(ctx, taskNodeName, parentNodeNames, err)
		return nil, err
	}

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

	// Allow unresolved tags: templates may contain tags like {{pod.name}} that are
	// resolved later by executeContainer, or task-scope tags for retry strategies.
	processedTmpl, err := common.ProcessArgs(ctx, resolvedTmpl, &args, e.woc.globalParams(), localParams, false, true, e.woc.wf.Namespace, e.woc.controller.configMapInformer.GetIndexer())
	if err != nil {
		e.initTerminalErrorNode(ctx, taskNodeName, parentNodeNames, err)
		return nil, err
	}

	return []DesiredTask{{
		TaskName:         taskNodeName,
		OriginalTaskName: taskName,
		TemplateScope:    e.tmplCtx.GetTemplateScope(),
		TmplCtx:          newTmplCtx,
		Template:         processedTmpl,
		TemplateRef:      task.GetTemplateReferenceHolder(),
		BoundaryID:       e.boundaryID,
		IsOnExit:         e.onExitTemplate,
		ParentNodeNames:  parentNodeNames,
	}}, nil
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
	if strings.HasPrefix(taskName, "[") {
		return fmt.Sprintf("%s%s", e.nodeName, taskName)
	}
	return fmt.Sprintf("%s.%s", e.nodeName, taskName)
}

// taskNodeID formulates the node ID for a dag task
func (e *Engine) taskNodeID(taskName string) string {
	nodeName := e.taskNodeName(taskName)
	return e.woc.wf.NodeID(nodeName)
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

	// Build set of leaf tasks (tasks that no other task depends on).
	leafTasks := e.findLeafTasks(ctx, tasks)

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

// addChildNode adds a child node to the appropriate parent.
// For Steps templates, step tasks are linked to their StepGroup node (restoring pre-refactor behavior).
// For DAG templates, tasks are linked to their dependencies' outbound nodes or the DAG root.
func (e *Engine) addChildNode(ctx context.Context, taskName string, childNodeName string) {
	// For Steps templates, link task nodes to their StepGroup
	if e.tmpl.GetType() == wfv1.TemplateTypeSteps {
		if sgName := e.stepGroupNodeName(taskName); sgName != "" {
			e.woc.addChildNode(ctx, sgName, childNodeName)
			return
		}
	}

	deps, err := e.evaluator.GetDependencies(ctx, taskName)
	if err != nil {
		e.log.WithFields(logging.Fields{"taskName": taskName, "error": err}).Warn(ctx, "failed to get dependencies")
		e.woc.addChildNode(ctx, e.nodeName, childNodeName)
		return
	}
	if len(deps) > 0 {
		for _, dep := range deps {
			depNodeName := e.taskNodeName(dep)
			depNodeID := e.woc.wf.NodeID(depNodeName)
			// Dep may not yet be in status (see createDesiredTask). Skip.
			if _, err := e.woc.wf.Status.Nodes.Get(depNodeID); err != nil {
				continue
			}
			outboundIDs := e.woc.getOutboundNodes(ctx, depNodeID)
			for _, outID := range outboundIDs {
				outNode, err := e.woc.wf.Status.Nodes.Get(outID)
				if err == nil {
					e.woc.addChildNode(ctx, outNode.Name, childNodeName)
				}
			}
		}
	} else {
		e.woc.addChildNode(ctx, e.nodeName, childNodeName)
	}
}

// stepGroupNodeName extracts the StepGroup node name from a task name.
// Task names for Steps are formatted as "[N].stepName" by StepAdapter.GetName().
func (e *Engine) stepGroupNodeName(taskName string) string {
	var groupIdx int
	if n, _ := fmt.Sscanf(taskName, "[%d].", &groupIdx); n == 1 {
		return fmt.Sprintf("%s[%d]", e.nodeName, groupIdx)
	}
	return ""
}

// assessTaskGroupPhase marks a TaskGroup node as terminal once all its non-hook children
// have reached a terminal phase. This is necessary because TaskGroup nodes are created in
// Running state and must be explicitly transitioned; the k8s reconciler only manages
// individual pod nodes, not their TaskGroup parent.
func (e *Engine) assessTaskGroupPhase(ctx context.Context, tgNode *wfv1.NodeStatus) {
	groupPhase := wfv1.NodeSucceeded
	for _, childID := range tgNode.Children {
		childNode, err := e.woc.wf.Status.Nodes.Get(childID)
		if err != nil {
			return // cannot assess phase if a child is missing
		}
		if childNode.NodeFlag != nil && (childNode.NodeFlag.Hooked || childNode.NodeFlag.Retried) {
			continue // hooks and retry placeholders don't affect group phase
		}
		if !childNode.Fulfilled() {
			return // still waiting
		}
		if childNode.FailedOrError() {
			groupPhase = childNode.Phase
		}
	}
	e.woc.markNodePhase(ctx, tgNode.Name, groupPhase)
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
func (e *Engine) addTaskNodeToScope(ctx context.Context, scope *wfScope, ref varkeys.NodeRefKeys, agg varkeys.AggregateKeys, refName, taskName string, node *wfv1.NodeStatus, includeArtifacts bool) error {
	if node.Type == wfv1.NodeTypeTaskGroup {
		if err := e.woc.processAggregateNodeOutputs(scope, agg, refName, e.getChildNodes(node)); err != nil {
			return fmt.Errorf("failed to aggregate outputs for %s: %w", taskName, err)
		}
	}
	e.woc.buildLocalScope(scope, ref, refName, node)
	holder := wfv1.TemplateReferenceHolder(node)
	if t := e.evaluator.GetTask(taskName); t != nil {
		holder = t.GetTemplateReferenceHolder()
	}
	e.woc.addSkippedNodeOutputsToScope(ctx, e.tmplCtx, scope, ref, refName, node, holder, includeArtifacts)
	return nil
}

// buildLocalScopeFromTask builds a local scope for a task.
func (e *Engine) buildLocalScopeFromTask(ctx context.Context, task dag.Task) (*wfScope, error) {
	scope := createScope(e.tmpl)
	// Add all ancestor tasks' outputs to scope (transitive closure of dependencies).
	// A task may reference outputs from any ancestor, not just direct dependencies
	// (e.g., {{tasks.grandparent.ip}} in a DAG). This matches main's behavior which
	// uses GetTaskAncestry to walk the full dependency graph.
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
		// leaves them to resolveArguments' optional-drop / required-error handling.
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
		var currentGroupIdx int
		if n, _ := fmt.Sscanf(task.GetName(), "[%d].", &currentGroupIdx); n != 1 {
			return nil, fmt.Errorf("failed to parse group index from step task name %q", task.GetName())
		}
		for i, stepGroup := range e.tmpl.Steps {
			if i >= currentGroupIdx {
				break
			}
			for _, step := range stepGroup.Steps {
				stepTaskName := fmt.Sprintf("[%d].%s", i, step.Name)
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

	includeArtifacts := e.tmpl.GetType() == wfv1.TemplateTypeSteps
	addNodeToScope := func(taskNode *wfv1.NodeStatus, ref varkeys.NodeRefKeys, agg varkeys.AggregateKeys, name string, tmplHolder wfv1.TemplateReferenceHolder) error {
		if taskNode.Type == wfv1.NodeTypeTaskGroup {
			childNodes := e.getChildNodes(taskNode)
			if aggErr := e.woc.processAggregateNodeOutputs(scope, agg, name, childNodes); aggErr != nil {
				return aggErr
			}
		}
		e.woc.buildLocalScope(scope, ref, name, taskNode)
		// A skipped/omitted task's declared outputs (producer default, else nil) must be in the
		// aggregation scope too, so the template's own output params — including
		// ValueFrom.Expression `??` fallbacks — can resolve them instead of failing to traverse nil.
		e.woc.addSkippedNodeOutputsToScope(ctx, e.tmplCtx, scope, ref, name, taskNode, tmplHolder, includeArtifacts)
		e.woc.addOutputsToGlobalScope(ctx, taskNode.Outputs)
		return nil
	}

	if e.tmpl.DAG != nil {
		for _, task := range e.tmpl.DAG.Tasks {
			taskNode := e.getTaskNode(ctx, task.Name)
			if taskNode == nil {
				continue
			}
			if err = addNodeToScope(taskNode, varkeys.TasksNodeRef, varkeys.TasksAggregate, task.Name, &task); err != nil {
				return err
			}
		}
	} else if e.tmpl.Steps != nil {
		for i, stepGroup := range e.tmpl.Steps {
			for _, step := range stepGroup.Steps {
				// Step nodes use the [i].name format for lookup, but steps.name for scope prefix.
				taskNode := e.getTaskNode(ctx, fmt.Sprintf("[%d].%s", i, step.Name))
				if taskNode == nil {
					continue
				}
				if err = addNodeToScope(taskNode, varkeys.StepsNodeRef, varkeys.StepsAggregate, step.Name, &step); err != nil {
					return err
				}
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

// shouldExecute evaluates a already substituted when expression to decide whether or not a step should execute
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	expression, err := govaluate.NewEvaluableExpression(when)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid token") {
			return false, errors.Errorf(errors.CodeBadRequest, "Invalid 'when' expression '%s': %v (hint: try wrapping the affected expression in quotes)", when, err)
		}
		return false, errors.Errorf(errors.CodeBadRequest, "Invalid 'when' expression '%s': %v", when, err)
	}
	// The following loop converts govaluate variables (which we don't use), into strings. This
	// allows us to have expressions like: "foo != bar" without requiring foo and bar to be quoted.
	tokens := expression.Tokens()
	for i, tok := range tokens {
		switch tok.Kind {
		case govaluate.VARIABLE:
			tok.Kind = govaluate.STRING
		default:
			continue
		}
		tokens[i] = tok
	}
	expression, err = govaluate.NewEvaluableExpressionFromTokens(tokens)
	if err != nil {
		return false, errors.InternalWrapErrorf(err, "Failed to parse 'when' expression '%s': %v", when, err)
	}
	result, err := expression.Evaluate(nil)
	if err != nil {
		return false, errors.InternalWrapErrorf(err, "Failed to evaluate 'when' expresion '%s': %v", when, err)
	}
	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf(errors.CodeBadRequest, "Expected boolean evaluation for '%s'. Got %v", when, result)
	}
	return boolRes, nil
}

// evaluateWhenClause evaluates a task's when clause against the given scope.
// Returns (true, nil) if the task should proceed, (false, nil) if it should be skipped,
// or (false, err) if evaluation failed. Tasks with withItems/withParam/withSequence
// always proceed (their when clause references {{item.*}} resolved during expansion).
//
// We substitute template variables first, then use shouldExecute which converts
// govaluate VARIABLE tokens to STRING tokens. This is critical because after
// substitution, bare words like "odd" and "even" in "odd == even" must be treated
// as string literals, not as nil-valued variables (which would make nil == nil → true).
func (e *Engine) evaluateWhenClause(ctx context.Context, task dag.Task, scope *wfScope) (bool, error) {
	when := task.GetWhen()
	if when == "" || dag.HasExpansion(task) {
		return true, nil
	}

	// nil-preserving view so `??` expression fallbacks can resolve skipped/omitted outputs; a simple
	// tag resolving to an absent optional (nil) is a terminal error, matching argument substitution.
	merged := scope.getParametersAny(e.woc.globalParams())
	tmpl, err := template.NewTemplate(when)
	if err != nil {
		return false, err
	}
	substituted, err := tmpl.Replace(ctx, merged, true)
	if err != nil {
		return false, err
	}
	return shouldExecute(substituted)
}
