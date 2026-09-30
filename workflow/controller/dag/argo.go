package dag

import (
	"context"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// DAGEvaluator provides a high-level API for evaluating DAG workflows.
//
//nolint:revive // DAGEvaluator reads clearer than Evaluator at its many call sites across packages
type DAGEvaluator struct {
	store    *workflowStore
	tasks    *WorkflowTasks
	workflow *wfv1.Workflow
	tmpl     *wfv1.Template

	// exprCache caches compiled expr-lang programs keyed by expression string.
	// Depends expressions are deterministic per task (from dagTopology.dependsLogic),
	// so the compiled program is reusable across evaluations — only the eval scope changes.
	// This eliminates repeated parsing, type-checking, and compilation which accounts
	// for ~44% of CPU and ~814MB of allocations per 10K-node evaluation cycle.
	exprCache map[string]*vm.Program
}

// NewDAGEvaluatorFromTasks creates a new DAGEvaluator for a workflow and a list of tasks.
func NewDAGEvaluatorFromTasks(wf *wfv1.Workflow, tasks []Task, tmpl *wfv1.Template, boundaryID, boundaryName string) *DAGEvaluator {
	store := newWorkflowStore(wf, boundaryID, boundaryName)
	wTasks := newWorkflowTasks(tasks)

	return &DAGEvaluator{
		store:     store,
		tasks:     wTasks,
		exprCache: make(map[string]*vm.Program),
		workflow:  wf,
		tmpl:      tmpl,
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
			// Dep hasn't started.
			return waiting, nil
		}
		// A dependency whose lifecycle or exit hooks are still running is not
		// ready for its dependants, whatever its own type or phase (#12192).
		if !e.store.areHooksFulfilled(depName) {
			return waiting, nil
		}
		// A dependency counts once its node is fulfilled; for a retry node that
		// is once the operator's retry handling has recorded its outcome. A
		// daemon that is still running counts too (Daemoned: true below) and
		// is NOT pending: explicit qualifiers like A.Succeeded are correctly
		// unsatisfiable for running daemons (A.Succeeded only becomes true when
		// killDaemonedChildren runs, which requires the boundary to complete
		// first — so waiting would deadlock).
		daemonRunning := depNode.IsDaemoned() && !depNode.Phase.Fulfilled(depNode.TaskResultSynced)
		if !daemonRunning && !depNode.Fulfilled() {
			return waiting, nil
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

		evalScope[normalizeTaskName(depName)] = taskResult{
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

// FindLeafTaskNames returns tasks that no other task depends on.
func (e *DAGEvaluator) FindLeafTaskNames(_ context.Context) []Key {
	return e.tasks.LeafTaskNames()
}

// evaluateTaskResult builds an EvaluationResult for a single task.
func (e *DAGEvaluator) evaluateTaskResult(ctx context.Context, taskName string) EvaluationResult {
	result := EvaluationResult{TaskName: taskName}

	// Check for depends expression parsing errors (e.g., invalid qualifiers).
	if err := e.tasks.GetDependsError(taskName); err != nil {
		result.Error = err
		return result
	}

	node := e.store.getNode(taskName)

	// Retry node — delegate to specialized assessment
	if node != nil && node.Type == wfv1.NodeTypeRetry {
		return e.evaluateRetryNode(taskName, node)
	}

	// TaskGroup node — delegate to specialized assessment
	if node != nil && node.Type == wfv1.NodeTypeTaskGroup {
		return e.evaluateTaskGroupNode(taskName, node)
	}

	if node != nil {
		if node.Phase == wfv1.NodeOmitted {
			result.Skipped = true
			result.SkipReason = node.Message
			return result
		}
		// Once its node exists a task is dispatched until it is fulfilled,
		// without re-evaluating its depends expression against dependencies
		// that may since have changed (e.g. a dead daemon), as
		// evaluateDependsLogic did before the Engine.
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
		// Determine what we're waiting on: the deps that have not finished.
		deps, _ := e.tasks.GetDependencies(ctx, taskName)
		for _, dep := range deps {
			if depNode := e.store.getNode(dep); depNode == nil || !depNode.Fulfilled() {
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

// Evaluate evaluates one task against the workflow's nodes as they are now.
// The Engine calls it for each task in dependency order, immediately before
// acting on the task, so the task sees what was just done to its
// dependencies; a dependency that can never run already has its Omitted
// node by then, which is why Evaluate needs no cascading-omission pass.
func (e *DAGEvaluator) Evaluate(ctx context.Context, taskName string) EvaluationResult {
	return e.evaluateTaskResult(ctx, taskName)
}

// evaluateRetryNode reports a Retry node by its own phase, which the
// operator's retry handling (processNodeRetries, reading the processed
// retryStrategy) records. Until the node is fulfilled it is dispatched on
// every reconcile, so that handling alone decides whether to start another
// attempt, wait out a backoff, or finish the node.
func (e *DAGEvaluator) evaluateRetryNode(taskName string, node *wfv1.NodeStatus) EvaluationResult {
	result := EvaluationResult{TaskName: taskName}
	if node.Fulfilled() {
		return result
	}
	result.Action = ActionExecute
	result.ShouldRun = true
	result.ActionReason = "retry node driven by the operator's retry handling"
	return result
}

// evaluateTaskGroupNode assesses a TaskGroup node (from withItems/withParam/withSequence).
// Until the group is fulfilled it is dispatched on every reconcile: the Engine
// expands the task, creates or re-enters each item, and completes the group
// with TaskGroupPhase once every item exists and has finished (after a hook
// error, once the items that exist have), as executeDAGTask and
// executeStepGroup did before the Engine.
func (e *DAGEvaluator) evaluateTaskGroupNode(taskName string, node *wfv1.NodeStatus) EvaluationResult {
	result := EvaluationResult{TaskName: taskName}
	if !node.Fulfilled() {
		result.Action = ActionExecute
		result.ShouldRun = true
		result.ActionReason = "task group items in progress"
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
