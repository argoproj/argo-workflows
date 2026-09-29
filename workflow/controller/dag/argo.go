package dag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// DAGEvaluator provides a high-level API for evaluating DAG workflows.
//
//nolint:revive // DAGEvaluator reads clearer than Evaluator at its many call sites across packages
type DAGEvaluator struct {
	store    *workflowStore
	tasks    *WorkflowTasks
	workflow *wfv1.Workflow
	tmpl     *wfv1.Template

	// previouslyOmitted tracks keys marked Omitted by evaluateAllStates so they
	// can be cleared at the start of the next call. This prevents stale
	// Omitted states from persisting when conditions change between calls.
	previouslyOmitted []Key

	// exprCache caches compiled expr-lang programs keyed by expression string.
	// Depends expressions are deterministic per task (from dagTopology.dependsLogic),
	// so the compiled program is reusable across evaluations — only the eval scope changes.
	// This eliminates repeated parsing, type-checking, and compilation which accounts
	// for ~44% of CPU and ~814MB of allocations per 10K-node evaluation cycle.
	exprCache map[string]*vm.Program

	// retryStrategies holds the resolved retry strategy for each task, registered
	// by the engine after template resolution.
	retryStrategies map[string]*wfv1.RetryStrategy
	// retryDeciders holds the engine-provided retry decision for each task; see
	// RetryDecider. Falls back to the built-in policy switch when absent.
	retryDeciders map[string]RetryDecider
}

// RetryDecider reports whether the retry node's last child may be retried
// under rs. The engine registers one per task so that the retry decision —
// including transient-error classification and retryStrategy.expression,
// which need controller context — has a single authority: the same logic
// processNodeRetries applies when it actually drives the retry.
type RetryDecider func(ctx context.Context, retryNode, lastChild *wfv1.NodeStatus, rs *wfv1.RetryStrategy) bool

// NewDAGEvaluatorFromTasks creates a new DAGEvaluator for a workflow and a list of tasks.
func NewDAGEvaluatorFromTasks(wf *wfv1.Workflow, tasks []Task, tmpl *wfv1.Template, boundaryID, boundaryName string) *DAGEvaluator {
	store := newWorkflowStore(wf, boundaryID, boundaryName)
	wTasks := newWorkflowTasks(tasks)

	return &DAGEvaluator{
		store:           store,
		tasks:           wTasks,
		exprCache:       make(map[string]*vm.Program),
		retryStrategies: make(map[string]*wfv1.RetryStrategy),
		retryDeciders:   make(map[string]RetryDecider),
		workflow:        wf,
		tmpl:            tmpl,
	}
}

// evalBool compiles and evaluates a boolean expression against a scope.
// Compiled programs are cached by expression string since the same depends
// expression is evaluated repeatedly with different scope values.
func (e *DAGEvaluator) evalBool(input string, env map[string]taskResult) (bool, error) {
	prog, ok := e.exprCache[input]
	if !ok {
		var err error
		prog, err = expr.Compile(input, expr.Env(env))
		if err != nil {
			return false, err
		}
		e.exprCache[input] = prog
	}
	result, err := expr.Run(prog, env)
	if err != nil {
		return false, fmt.Errorf("unable to evaluate expression '%s': %w", input, err)
	}
	resultBool, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("unable to cast expression result '%s' to bool", result)
	}
	return resultBool, nil
}

// isReady determines if a task should run, wait, or be omitted.
// Always checks the actual workflow node (ground truth) rather than internal
// store state, so that tasks are re-evaluated when conditions change.
func (e *DAGEvaluator) isReady(ctx context.Context, key Key) (readinessResult, error) {
	node := e.store.getNode(key)
	if node != nil {
		if node.Fulfilled() {
			return omit, nil
		}
		// Once its node exists a task keeps being reconciled, whatever its
		// dependencies do next (a daemon it depends on may die), as
		// evaluateDependsLogic did before the Engine.
		return ready, nil
	}
	// No node yet — evaluate depends logic
	return e.evaluateDependsReadiness(ctx, key)
}

// evaluateDependsReadiness evaluates the depends expression for a task and
// returns a readinessResult. The task waits while any dependency it references
// is still pending; once every one has finished, the expression decides
// whether it runs or is omitted. This is the pre-Engine rule: an expression is
// never evaluated against a dependency that has not finished.
func (e *DAGEvaluator) evaluateDependsReadiness(ctx context.Context, taskName string) (readinessResult, error) {
	node := e.store.getNode(taskName)
	if node != nil && node.Fulfilled() {
		return ready, nil
	}

	evalScope := make(map[string]taskResult)

	deps, _ := e.tasks.GetDependencies(ctx, taskName)
	for _, depName := range deps {
		depNode := e.store.getNode(depName)

		if depNode == nil {
			depPhase := e.store.getPhase(ctx, depName)
			if depPhase == wfv1.NodeOmitted {
				evalTaskName := normalizeTaskName(depName)
				evalScope[evalTaskName] = taskResult{Omitted: true}
				continue
			}
			// Dep hasn't started.
			return waiting, nil
		}
		// A dependency whose lifecycle or exit hooks are still running is not
		// ready for its dependants, whatever its own type or phase (#12192).
		// Checked before the type-specific handling below so that retry nodes,
		// whose assessment returns early, are gated too.
		if !e.store.areHooksFulfilled(depName) {
			return waiting, nil
		}
		// Daemoned and still running — fulfilled for dependency purposes, so skip
		// the retry/not-fulfilled handling and fall through to evalScope building
		// below (sets Daemoned: true). NOT marked as pending: explicit qualifiers
		// like A.Succeeded are correctly unsatisfiable for running daemons
		// (A.Succeeded only becomes true when killDaemonedChildren runs, which
		// requires the boundary to complete first — so waiting would deadlock).
		daemonRunning := depNode.IsDaemoned() && !depNode.Phase.Fulfilled(depNode.TaskResultSynced)
		if !daemonRunning {
			if depNode.Type == wfv1.NodeTypeRetry {
				// For retry nodes, use the evaluator's assessment to determine dep state.
				retryResult := e.evaluateRetryNode(ctx, depName, depNode)
				// The assessment is derived from the attempt children and can run
				// ahead of the retry node's own phase: the engine marks the node
				// Succeeded/Failed only when it dispatches this result. Until then
				// the dependency is pending. Dispatching a dependant in the same
				// pass would link it under the last attempt before the retry node
				// is finalized, and handleRetries would then see an unfulfilled
				// descendant and start a spurious extra attempt. It would also
				// run before the exit hook the engine creates on finalization
				// (#12192). A running daemon child is the exception below.
				if (retryResult.Action == ActionSucceed || retryResult.Action == ActionFail) && !depNode.Fulfilled() {
					return waiting, nil
				}
				if retryResult.Action == ActionFail {
					// Retry is done — use the actual child phase (Error vs Failed)
					evalTaskName := normalizeTaskName(depName)
					evalScope[evalTaskName] = taskResult{
						Failed:  retryResult.CurrentPhase == wfv1.NodeFailed,
						Errored: retryResult.CurrentPhase == wfv1.NodeError,
						Skipped: retryResult.CurrentPhase == wfv1.NodeSkipped,
						Omitted: retryResult.CurrentPhase == wfv1.NodeOmitted,
					}
					continue
				}
				if retryResult.FulfilledForDeps {
					evalTaskName := normalizeTaskName(depName)
					if retryResult.Action == ActionSucceed {
						evalScope[evalTaskName] = taskResult{Succeeded: true}
					} else {
						// Daemoned child running — fulfilled for dep purposes.
						// Same as direct daemon deps: NOT marked as pending.
						evalScope[evalTaskName] = taskResult{Daemoned: true}
					}
					continue
				}
				if !depNode.Fulfilled() {
					return waiting, nil
				}
			} else if !depNode.Fulfilled() {
				// Dep running but not fulfilled.
				return waiting, nil
			}
		}

		evalTaskName := normalizeTaskName(depName)
		if _, ok := evalScope[evalTaskName]; ok {
			continue
		}

		anySucceeded := false
		allFailed := false

		if depNode.Type == wfv1.NodeTypeTaskGroup {
			// Only the expanded items count: a lifecycle hook hanging off the
			// group is not an item and must not make it AnySucceeded.
			children, missingChildren := e.store.taskGroupChildren(depName)
			allFailed = len(children) > 0 && !missingChildren

			for _, child := range children {
				anySucceeded = anySucceeded || child.Phase == wfv1.NodeSucceeded
				allFailed = allFailed && child.Phase == wfv1.NodeFailed
			}
		}

		evalScope[evalTaskName] = taskResult{
			Succeeded:    depNode.Phase == wfv1.NodeSucceeded,
			Failed:       depNode.Phase == wfv1.NodeFailed,
			Errored:      depNode.Phase == wfv1.NodeError,
			Skipped:      depNode.Phase == wfv1.NodeSkipped,
			Omitted:      depNode.Phase == wfv1.NodeOmitted,
			Daemoned:     depNode.IsDaemoned() && depNode.Phase != wfv1.NodePending,
			AnySucceeded: anySucceeded,
			AllFailed:    allFailed,
		}
	}

	logic := e.tasks.GetDependsLogic(ctx, taskName)
	if logic == "" {
		return ready, nil
	}

	result, err := e.evalBool(logic, evalScope)
	if err != nil {
		return omit, fmt.Errorf("depends expression evaluation failed for task %s: %w", taskName, err)
	}
	if result {
		return ready, nil
	}
	return omit, nil
}

// evaluateAllStates evaluates all tasks and handles cascading omission in a
// single pass over the topological order: a task is evaluated after all of
// its dependencies, so if task A is omitted, downstream tasks whose depends
// conditions can never be met are omitted in the same pass.
//
// IMPORTANT: This method clears previously-set Omitted states at the start,
// then re-evaluates from scratch. EvaluateAll, the production entry point,
// calls it first so that every result it returns is read against a
// consistent state; anything else that reads task phases must run after it.
// Multiple calls within the same evaluation cycle are safe but wasteful —
// prefer calling EvaluateAll once and reusing the results.
func (e *DAGEvaluator) evaluateAllStates(ctx context.Context) {
	// Clear Omitted states set by the previous call.
	// Conditions may have changed (e.g., a dep finished), so we must
	// re-evaluate from scratch rather than trust stale Omitted markers.
	for _, key := range e.previouslyOmitted {
		e.store.setPhase(ctx, key, wfv1.NodePending)
	}
	e.previouslyOmitted = nil

	// Evaluate tasks in topological order (dependencies before dependents).
	// This ensures that by the time we evaluate a task, all its dependencies
	// have already been evaluated and marked Omitted if unreachable.
	// Single pass: O(N) instead of O(N²) fixed-point for linear chains.
	for _, key := range e.tasks.TopologicalOrder() {
		phase := e.store.getPhase(ctx, key)
		if phase.Fulfilled(nil) || phase == wfv1.NodeRunning {
			continue
		}
		result, err := e.isReady(ctx, key)
		// Only mark as Omitted when the depends condition is genuinely unsatisfiable.
		// If isReady returned an error (e.g., broken expression syntax), leave the
		// task pending so evaluateTaskResult will re-evaluate and surface the error.
		if result == omit && err == nil {
			e.store.setPhase(ctx, key, wfv1.NodeOmitted)
			e.previouslyOmitted = append(e.previouslyOmitted, key)
		}
	}
}

// FindLeafTaskNames returns tasks that no other task depends on.
func (e *DAGEvaluator) FindLeafTaskNames(_ context.Context) []Key {
	return e.tasks.LeafTaskNames()
}

// evaluateTaskResult builds an EvaluationResult for a single task.
func (e *DAGEvaluator) evaluateTaskResult(ctx context.Context, taskName string) EvaluationResult {
	phase := e.store.getPhase(ctx, taskName)

	result := EvaluationResult{
		TaskName:     taskName,
		CurrentPhase: phase,
	}

	// Check for depends expression parsing errors (e.g., invalid qualifiers).
	if err := e.tasks.GetDependsError(taskName); err != nil {
		result.Error = err
		return result
	}

	node := e.store.getNode(taskName)

	// Retry node — delegate to specialized assessment
	if node != nil && node.Type == wfv1.NodeTypeRetry {
		return e.evaluateRetryNode(ctx, taskName, node)
	}

	// TaskGroup node — delegate to specialized assessment
	if node != nil && node.Type == wfv1.NodeTypeTaskGroup {
		return e.evaluateTaskGroupNode(taskName, node)
	}

	if phase == wfv1.NodeOmitted && node == nil {
		result.Skipped = true
		result.SkipReason = "depends condition not met"
		return result
	}

	if node != nil {
		if node.Phase == wfv1.NodeOmitted {
			result.Skipped = true
			result.SkipReason = node.Message
			return result
		}
		// See isReady: a started task is dispatched until it is fulfilled,
		// without re-evaluating its depends expression against dependencies
		// that may since have changed (e.g. a dead daemon).
		result.ShouldRun = !node.Fulfilled()
		return result
	}

	// No node exists yet — check readiness
	readiness, exprErr := e.evaluateDependsReadiness(ctx, taskName)
	if exprErr != nil {
		result.Error = exprErr
	}
	switch readiness {
	case ready:
		result.ShouldRun = true
	case waiting:
		result.Suspended = true
		// Determine what we're waiting on. Exclude terminal deps (including
		// Omitted from the evaluator's phases map) since they can never complete.
		deps, _ := e.tasks.GetDependencies(ctx, taskName)
		for _, dep := range deps {
			depPhase := e.store.getPhase(ctx, dep)
			if depPhase.Fulfilled(nil) {
				continue
			}
			depNode := e.store.getNode(dep)
			if depNode == nil || !depNode.Fulfilled() {
				result.WaitingOn = append(result.WaitingOn, dep)
			}
		}
	case omit:
		result.Skipped = true
		result.SkipReason = "depends condition not met"
	}

	return result
}

// GetDependencies returns the dependency task names for a given task.
func (e *DAGEvaluator) GetDependencies(ctx context.Context, taskName string) ([]Key, error) {
	return e.tasks.GetDependencies(ctx, taskName)
}

// GetTask returns the Task with the given name, or nil if not found. Used to obtain a task's
// template reference holder (e.g. to resolve a skipped node's declared outputs, which a node
// status alone cannot resolve — it falls back to the boundary template).
func (e *DAGEvaluator) GetTask(name string) Task {
	return e.tasks.GetTask(name)
}

// GetAncestors returns all ancestor task names for a given task (transitive closure
// of dependencies). This is needed because a task may reference outputs from any
// ancestor, not just its direct dependencies (e.g., {{tasks.grandparent.ip}}).
func (e *DAGEvaluator) GetAncestors(ctx context.Context, taskName string) ([]Key, error) {
	visited := make(map[Key]bool)
	var walk func(name Key) error
	walk = func(name Key) error {
		deps, err := e.tasks.GetDependencies(ctx, name)
		if err != nil {
			return err
		}
		for _, dep := range deps {
			if visited[dep] {
				continue
			}
			visited[dep] = true
			if err := walk(dep); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(taskName); err != nil {
		return nil, err
	}
	result := make([]Key, 0, len(visited))
	for k := range visited {
		result = append(result, k)
	}
	return result, nil
}

// GetTargetTasks returns the target tasks for the DAG.
func (e *DAGEvaluator) GetTargetTasks(ctx context.Context) []string {
	if e.tmpl != nil && e.tmpl.DAG != nil && e.tmpl.DAG.Target != "" {
		return strings.Fields(e.tmpl.DAG.Target)
	}
	return e.FindLeafTaskNames(ctx)
}

// EvaluateAll evaluates all tasks in the DAG and returns a map of results.
// An expanded task has one result, for its TaskGroup: the Engine drives its
// items when it dispatches the group (see evaluateTaskGroupNode).
func (e *DAGEvaluator) EvaluateAll(ctx context.Context) map[string]EvaluationResult {
	// Run evaluateAllStates to handle cascading omission
	e.evaluateAllStates(ctx)

	results := make(map[string]EvaluationResult)
	for _, taskName := range e.tasks.TaskNames() {
		results[taskName] = e.evaluateTaskResult(ctx, taskName)
	}
	return results
}

// SetRetryStrategy registers a retry strategy for a task.
// Called by the engine after template resolution.
func (e *DAGEvaluator) SetRetryStrategy(taskName string, rs *wfv1.RetryStrategy) {
	e.retryStrategies[taskName] = rs
}

// SetRetryDecider registers the retry decision for a task; see RetryDecider.
func (e *DAGEvaluator) SetRetryDecider(taskName string, d RetryDecider) {
	e.retryDeciders[taskName] = d
}

// staticTaskName strips the expansion suffix from an expanded
// withItems/withParam/withSequence child name (e.g. "A(0:x)" -> "A").
func staticTaskName(taskName string) string {
	if i := strings.Index(taskName, "("); i > 0 {
		return taskName[:i]
	}
	return taskName
}

// retryStrategyFor returns the retry strategy registered for a task.
// Strategies are registered under static task names, but expanded
// children are looked up under their expanded name; those inherit
// the static task's strategy.
func (e *DAGEvaluator) retryStrategyFor(taskName string) *wfv1.RetryStrategy {
	if rs, ok := e.retryStrategies[taskName]; ok {
		return rs
	}
	return e.retryStrategies[staticTaskName(taskName)]
}

// retryDeciderFor returns the retry decider registered for a task, with the
// same expanded-child fallback as retryStrategyFor.
func (e *DAGEvaluator) retryDeciderFor(taskName string) RetryDecider {
	if d, ok := e.retryDeciders[taskName]; ok {
		return d
	}
	return e.retryDeciders[staticTaskName(taskName)]
}

// nextRetryBackoff returns how much of the backoff window is still left before
// the next retry attempt: common.RetryBackoffWait (the same formula the
// operator enforces) minus the time elapsed since the last child finished.
// Returns 0 if no backoff applies, the strategy is invalid (the operator
// surfaces that error when it drives the retry), or the window has passed.
// backoff.maxDuration is a deadline enforced by processNodeRetries, not a wait.
func nextRetryBackoff(rs *wfv1.RetryStrategy, lastChild *wfv1.NodeStatus, attempts int) time.Duration {
	delay, err := common.RetryBackoffWait(rs, attempts)
	if err != nil || delay <= 0 {
		return 0
	}
	if lastChild != nil && !lastChild.FinishedAt.IsZero() {
		delay = time.Until(lastChild.FinishedAt.Add(delay))
	}
	return max(delay, 0)
}

// evaluateRetryNode inspects a Retry node's children and returns what action
// should be taken. This is the pure-assessment equivalent of processNodeRetries
// in operator.go — it produces no side effects, only a result.
func (e *DAGEvaluator) evaluateRetryNode(ctx context.Context, taskName string, node *wfv1.NodeStatus) EvaluationResult {
	result := EvaluationResult{
		TaskName:     taskName,
		CurrentPhase: node.Phase,
	}

	// Try store lookup first (works for top-level tasks).
	// Fall back to reading children directly from the node (works for TaskGroup
	// children where the task name doesn't match the store's naming convention).
	children := e.store.getRetryChildren(taskName)
	if len(children) == 0 && len(node.Children) > 0 {
		for _, childID := range node.Children {
			child, err := e.store.nodes.Get(childID)
			if err != nil {
				continue
			}
			if child.NodeFlag != nil && child.NodeFlag.Hooked {
				continue
			}
			children = append(children, child)
		}
	}

	// No children yet — first attempt needed.
	if len(children) == 0 {
		result.Action = ActionExecute
		result.ActionReason = "first retry attempt needed"
		result.ShouldRun = true
		return result
	}

	lastChild := children[len(children)-1]

	// Daemoned child that is still running — treat as fulfilled for deps.
	// Guard with phase check: a dead daemon (Daemoned=true + Failed) should
	// fall through to the failure handling, not be treated as running.
	if lastChild.IsDaemoned() && !lastChild.Phase.Fulfilled(lastChild.TaskResultSynced) {
		result.Action = ActionNone
		result.ActionReason = "daemon child is running"
		result.CurrentPhase = wfv1.NodeSucceeded
		result.FulfilledForDeps = true
		return result
	}

	// Last child still running — wait for it to finish.
	if !lastChild.Phase.Fulfilled(lastChild.TaskResultSynced) {
		result.Action = ActionNone
		result.ActionReason = "last attempt still running"
		return result
	}

	// Last child succeeded — propagate success.
	if lastChild.Phase == wfv1.NodeSucceeded {
		result.Action = ActionSucceed
		result.ActionReason = "last attempt succeeded"
		result.FulfilledForDeps = true
		return result
	}

	// Last child skipped or omitted — check retry policy before giving up.
	// RetryPolicyAlways should retry even Skipped children.
	if lastChild.Phase == wfv1.NodeSkipped || lastChild.Phase == wfv1.NodeOmitted {
		rs := e.retryStrategyFor(taskName)
		if rs != nil && rs.RetryPolicyActual() == wfv1.RetryPolicyAlways {
			if rs.Limit != nil && len(children) > rs.Limit.IntValue() {
				result.Action = ActionFail
				result.ActionReason = fmt.Sprintf("retry limit exhausted (%d/%d)", len(children)-1, rs.Limit.IntValue())
				result.CurrentPhase = wfv1.NodeFailed
				result.FulfilledForDeps = true
				return result
			}
			if backoff := nextRetryBackoff(rs, lastChild, len(children)); backoff > 0 {
				result.Action = ActionNone
				result.ActionReason = fmt.Sprintf("waiting %s before retry attempt %d (policy Always)", backoff, len(children))
				result.RequeueAfter = backoff
				return result
			}
			result.Action = ActionExecute
			result.ActionReason = fmt.Sprintf("scheduling retry attempt %d (policy Always)", len(children))
			result.ShouldRun = true
			return result
		}
		result.Action = ActionFail
		result.ActionReason = fmt.Sprintf("last attempt was %s", lastChild.Phase)
		result.CurrentPhase = wfv1.NodeFailed
		result.FulfilledForDeps = true
		return result
	}

	// Last child failed or errored — check retry policy and limits.
	// Propagate the child's actual phase (Error vs Failed) so downstream
	// depends expressions (A.Errored vs A.Failed) work correctly.
	if lastChild.FailedOrError() {
		rs := e.retryStrategyFor(taskName)
		if rs == nil {
			result.Action = ActionFail
			result.ActionReason = "no retry strategy configured"
			result.CurrentPhase = lastChild.Phase
			result.FulfilledForDeps = true
			return result
		}

		if !e.shouldRetry(ctx, taskName, node, lastChild, rs) {
			result.Action = ActionFail
			result.ActionReason = fmt.Sprintf("retry policy %s does not allow retry for phase %s", rs.RetryPolicyActual(), lastChild.Phase)
			result.CurrentPhase = lastChild.Phase
			result.FulfilledForDeps = true
			return result
		}

		if rs.Limit != nil {
			limit := rs.Limit.IntValue()
			if len(children) > limit {
				result.Action = ActionFail
				result.ActionReason = fmt.Sprintf("retry limit exhausted (%d/%d)", len(children)-1, limit)
				result.CurrentPhase = lastChild.Phase
				result.FulfilledForDeps = true
				return result
			}
		}

		if backoff := nextRetryBackoff(rs, lastChild, len(children)); backoff > 0 {
			result.Action = ActionNone
			result.ActionReason = fmt.Sprintf("waiting %s before retry attempt %d", backoff, len(children))
			result.RequeueAfter = backoff
			return result
		}
		result.Action = ActionExecute
		result.ActionReason = fmt.Sprintf("scheduling retry attempt %d", len(children))
		result.ShouldRun = true
		return result
	}

	// Fallback for unexpected phases.
	result.Action = ActionNone
	result.ActionReason = fmt.Sprintf("unexpected child phase: %s", lastChild.Phase)
	return result
}

// evaluateTaskGroupNode assesses a TaskGroup node (from withItems/withParam/withSequence).
// Until the group is fulfilled it is dispatched on every pass: the Engine
// expands the task, creates or re-enters each item, and completes the group
// with TaskGroupPhase once every item exists and has finished, as
// executeDAGTask and executeStepGroup did before the Engine. A completed
// group whose daemoned item has since died reports that failure without
// changing the node.
func (e *DAGEvaluator) evaluateTaskGroupNode(taskName string, node *wfv1.NodeStatus) EvaluationResult {
	result := EvaluationResult{
		TaskName:     taskName,
		CurrentPhase: node.Phase,
	}
	if !node.Fulfilled() {
		result.Action = ActionExecute
		result.ShouldRun = true
		result.ActionReason = "task group items in progress"
		return result
	}
	result.FulfilledForDeps = true
	if node.Phase == wfv1.NodeSucceeded {
		if phase, done := TaskGroupPhase(e.store.getTaskGroupChildren(taskName)); done && phase.FailedOrError() {
			result.CurrentPhase = phase
		}
	}
	return result
}

// TaskGroupPhase is the phase a TaskGroup takes from its item nodes: the
// worst of them, Error outranking Failed outranking Succeeded, whatever their
// order. done is false while an item is missing (nil) or unfinished; a running
// daemon counts as finished, as it does for the group's dependants.
func TaskGroupPhase(items []*wfv1.NodeStatus) (phase wfv1.NodePhase, done bool) {
	phase = wfv1.NodeSucceeded
	for _, item := range items {
		if item == nil || !item.Fulfilled() {
			return "", false
		}
		if item.Phase == wfv1.NodeError || (item.Phase == wfv1.NodeFailed && phase != wfv1.NodeError) {
			phase = item.Phase
		}
	}
	return phase, true
}

// shouldRetry determines if the retry policy allows retrying for the given
// child's terminal phase. When no explicit policy is set, the default depends
// on whether an expression is configured (see RetryPolicyActual).
//
// When the engine registered a RetryDecider for the task, that decision is
// authoritative: it applies the controller's transient-error classification
// and retryStrategy.expression, which this package cannot evaluate. The
// built-in switch below is only the fallback for evaluators used without an
// engine (tests, tooling).
func (e *DAGEvaluator) shouldRetry(ctx context.Context, taskName string, retryNode, lastChild *wfv1.NodeStatus, rs *wfv1.RetryStrategy) bool {
	if decide := e.retryDeciderFor(taskName); decide != nil {
		return decide(ctx, retryNode, lastChild, rs)
	}
	switch rs.RetryPolicyActual() {
	case wfv1.RetryPolicyAlways:
		return true
	case wfv1.RetryPolicyOnFailure:
		return lastChild.Phase == wfv1.NodeFailed
	case wfv1.RetryPolicyOnError:
		return lastChild.Phase == wfv1.NodeError
	case wfv1.RetryPolicyOnTransientError:
		// Fallback only: transient-error detection needs the controller's
		// classifier, provided via RetryDecider.
		return lastChild.Phase == wfv1.NodeFailed || lastChild.Phase == wfv1.NodeError
	default:
		return false
	}
}
