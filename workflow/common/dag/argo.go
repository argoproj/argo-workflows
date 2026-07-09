package dag

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

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
}

// NewDAGEvaluator creates a new DAGEvaluator for a workflow and DAG template.
func NewDAGEvaluator(wf *wfv1.Workflow, tmpl *wfv1.Template, boundaryID, boundaryName string) *DAGEvaluator {
	var dagTasks []Task
	if tmpl.DAG != nil {
		dagTasks = make([]Task, len(tmpl.DAG.Tasks))
		for i := range tmpl.DAG.Tasks {
			dagTasks[i] = &DAGTask{DAGTask: &tmpl.DAG.Tasks[i]}
		}
	}
	return NewDAGEvaluatorFromTasks(wf, dagTasks, tmpl, boundaryID, boundaryName)
}

// NewDAGEvaluatorFromTasks creates a new DAGEvaluator for a workflow and a list of tasks.
func NewDAGEvaluatorFromTasks(wf *wfv1.Workflow, tasks []Task, tmpl *wfv1.Template, boundaryID, boundaryName string) *DAGEvaluator {
	store := newWorkflowStore(wf, boundaryID, boundaryName)
	wTasks := newWorkflowTasks(tasks)

	return &DAGEvaluator{
		store:           store,
		tasks:           wTasks,
		exprCache:       make(map[string]*vm.Program),
		retryStrategies: make(map[string]*wfv1.RetryStrategy),
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
		if node.Phase == wfv1.NodeRunning {
			return ready, nil
		}
	}
	// No node or node is Pending — evaluate depends logic
	return e.evaluateDependsReadiness(ctx, key)
}

// evaluateDependsReadiness evaluates the depends expression for a task
// and returns a readinessResult.
func (e *DAGEvaluator) evaluateDependsReadiness(ctx context.Context, taskName string) (readinessResult, error) {
	node := e.store.getNode(taskName)
	if node != nil && node.Fulfilled() {
		return ready, nil
	}

	evalScope := make(map[string]taskResult)
	hasPendingDeps := false
	// Track deps treated as pending so the best-case check can upgrade them.
	pendingDepNames := make(map[string]bool)

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
			// Dep hasn't started — include with all-false fields so we can
			// still evaluate the expression to detect unsatisfiable conditions.
			evalTaskName := normalizeTaskName(depName)
			evalScope[evalTaskName] = taskResult{}
			hasPendingDeps = true
			pendingDepNames[depName] = true
			continue
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
					evalTaskName := normalizeTaskName(depName)
					evalScope[evalTaskName] = taskResult{}
					hasPendingDeps = true
					pendingDepNames[depName] = true
					continue
				}
			} else if !depNode.Fulfilled() {
				// Dep running but not fulfilled — include with all-false fields
				evalTaskName := normalizeTaskName(depName)
				evalScope[evalTaskName] = taskResult{}
				hasPendingDeps = true
				pendingDepNames[depName] = true
				continue
			}
		}

		if !e.store.areHooksFulfilled(depName) {
			// Dep is fulfilled but its hooks are still running.
			// Treat as pending — once hooks complete, dep will be fully ready.
			evalTaskName := normalizeTaskName(depName)
			evalScope[evalTaskName] = taskResult{}
			hasPendingDeps = true
			pendingDepNames[depName] = true
			continue
		}

		evalTaskName := normalizeTaskName(depName)
		if _, ok := evalScope[evalTaskName]; ok {
			continue
		}

		anySucceeded := false
		allFailed := false

		if depNode.Type == wfv1.NodeTypeTaskGroup {
			children := e.store.getTaskChildNodes(depName)
			missingChildren := len(children) < len(depNode.Children)
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
		if hasPendingDeps {
			return waiting, nil
		}
		return ready, nil
	}

	result, err := e.evalBool(logic, evalScope)
	if err != nil {
		return omit, fmt.Errorf("depends expression evaluation failed for task %s: %w", taskName, err)
	}

	if !hasPendingDeps {
		// All deps are in terminal states — the expression cannot change.
		if result {
			return ready, nil
		}
		return omit, nil
	}

	// Some deps are pending. Determine whether the expression could still
	// evaluate true under any realistic future outcome, and whether it
	// could still evaluate false. If both are possible, the task is
	// undecided (waiting). If only true is possible, the task can fire
	// now regardless of how pending deps resolve (ready). If only false
	// is possible, the expression is provably unsatisfiable (omit).
	pendingDeps := make([]string, 0, len(pendingDepNames))
	for depName := range pendingDepNames {
		pendingDeps = append(pendingDeps, depName)
	}
	sort.Strings(pendingDeps)

	canBeTrue, canBeFalse := e.enumerateOutcomes(logic, evalScope, pendingDeps, result)
	if !canBeTrue {
		return omit, nil
	}
	if !canBeFalse {
		return ready, nil
	}
	return waiting, nil
}

// pendingDepOutcomes is the set of realistic taskResult shapes a pending
// dep could eventually resolve to. Listing discrete shapes rather than
// the cartesian product of all 8 boolean fields keeps the enumeration
// tractable while still covering the states that affect real depends
// expressions — including the "did-not-fail" shape that proves negated
// references are satisfiable.
var pendingDepOutcomes = []taskResult{
	{Succeeded: true},
	{Succeeded: true, AnySucceeded: true},
	{Failed: true},
	{Failed: true, AllFailed: true},
	{Errored: true},
	{Skipped: true},
	{Omitted: true},
	{Daemoned: true},
	{Daemoned: true, Succeeded: true},
}

// enumerateOutcomes tries each combination of pendingDepOutcomes for the
// given pending deps and reports whether the expression can evaluate true
// (canBeTrue) and/or false (canBeFalse). Stops as soon as both are known.
//
// currentResult seeds canBeTrue/canBeFalse from the already-computed
// evaluation with the naive (all-false) scope for pending deps — this
// ensures that state is included in the search space (the "running, no
// signal yet" shape) without an extra evalBool call.
//
// To avoid exponential cost on tasks with many pending deps, enumeration
// is capped: beyond the cap, both outcomes are conservatively assumed
// possible so the task waits rather than being prematurely omitted.
func (e *DAGEvaluator) enumerateOutcomes(logic string, scope map[string]taskResult, pendingDeps []string, currentResult bool) (canBeTrue, canBeFalse bool) {
	if currentResult {
		canBeTrue = true
	} else {
		canBeFalse = true
	}

	const maxEnumerationDeps = 5
	if len(pendingDeps) > maxEnumerationDeps {
		return true, true
	}

	var recurse func(idx int)
	recurse = func(idx int) {
		if canBeTrue && canBeFalse {
			return
		}
		if idx == len(pendingDeps) {
			r, err := e.evalBool(logic, scope)
			if err != nil {
				return
			}
			if r {
				canBeTrue = true
			} else {
				canBeFalse = true
			}
			return
		}
		depName := pendingDeps[idx]
		evalName := normalizeTaskName(depName)
		original := scope[evalName]
		for _, outcome := range pendingDepOutcomes {
			scope[evalName] = outcome
			recurse(idx + 1)
			if canBeTrue && canBeFalse {
				break
			}
		}
		scope[evalName] = original
	}
	recurse(0)
	return canBeTrue, canBeFalse
}

// evaluateAllStates evaluates all tasks and handles cascading omission.
// Uses a fixed-point loop: if task A is omitted, downstream tasks whose
// depends conditions can never be met are also omitted.
//
// IMPORTANT: This method clears previously-set Omitted states at the start,
// then re-evaluates from scratch. It must be called before any method that
// reads task phases (EvaluateAll, EvaluateTask) to ensure
// consistent state. Multiple calls within the same evaluation cycle are safe
// but wasteful — prefer calling EvaluateAll once and reusing the results.
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
		if isTerminalPhase(phase) || phase == wfv1.NodeRunning {
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
func (e *DAGEvaluator) FindLeafTaskNames(ctx context.Context) []Key {
	isLeaf := make(map[Key]bool)
	for _, key := range e.tasks.Keys() {
		if _, ok := isLeaf[key]; !ok {
			isLeaf[key] = true
		}
		deps, _ := e.tasks.GetDependencies(ctx, key)
		for _, dep := range deps {
			isLeaf[dep] = false
		}
	}

	var leaves []Key
	for key, leaf := range isLeaf {
		if leaf {
			leaves = append(leaves, key)
		}
	}
	sort.Strings(leaves)
	return leaves
}

// EvaluateTask evaluates a single task and returns its evaluation result.
func (e *DAGEvaluator) EvaluateTask(ctx context.Context, taskName string) EvaluationResult {
	// Run evaluateAllStates to handle cascading omission
	e.evaluateAllStates(ctx)
	return e.evaluateTaskResult(ctx, taskName)
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
		return e.evaluateTaskGroupNode(ctx, taskName, node)
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
		if !node.Fulfilled() {
			readiness, exprErr := e.evaluateDependsReadiness(ctx, taskName)
			if readiness == ready {
				result.ShouldRun = true
			}
			if exprErr != nil {
				result.Error = exprErr
			}
		}
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
			if isTerminalPhase(depPhase) {
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
//
// For TaskGroup parents (withItems/withParam/withSequence) that have already been
// expanded into per-item children, an additional EvaluationResult is emitted for
// each non-hook child (e.g. "client(0:0)", "client(1:1)"). This lets the engine
// dispatch execution per-child — for example, retrying a synchronization
// TryAcquire that was queued because a sibling held the lock.
func (e *DAGEvaluator) EvaluateAll(ctx context.Context) map[string]EvaluationResult {
	// Run evaluateAllStates to handle cascading omission
	e.evaluateAllStates(ctx)

	results := make(map[string]EvaluationResult)
	for _, taskName := range e.tasks.TaskNames() {
		results[taskName] = e.evaluateTaskResult(ctx, taskName)
		if e.isTaskGroupParent(taskName) {
			e.appendTaskGroupChildResults(ctx, taskName, results)
		}
	}
	return results
}

// isTaskGroupParent reports whether the static task uses withItems/withParam/withSequence.
func (e *DAGEvaluator) isTaskGroupParent(taskName string) bool {
	task := e.tasks.GetTask(taskName)
	return task != nil && HasExpansion(task)
}

// appendTaskGroupChildResults emits an EvaluationResult for each schedulable
// expanded child of a TaskGroup parent, keyed by the child's bare task name
// (e.g. "client(0:0)"). ParentTaskName carries the static parent so the engine
// can dispatch without reverse-parsing the child name.
func (e *DAGEvaluator) appendTaskGroupChildResults(ctx context.Context, parentName string, results map[string]EvaluationResult) {
	for _, childNode := range e.store.getTaskGroupChildren(parentName) {
		childTaskName := e.store.taskNameFromNodeName(childNode.Name)
		var r EvaluationResult
		// Retry-typed children carry their own state machine; defer to evaluateRetryNode
		// so retry-limit/policy handling stays centralized.
		if childNode.Type == wfv1.NodeTypeRetry {
			r = e.evaluateRetryNode(ctx, childTaskName, childNode)
		} else {
			r = e.evaluateTaskGroupChild(childTaskName, childNode)
		}
		r.ParentTaskName = parentName
		results[childTaskName] = r
	}
}

// evaluateTaskGroupChild returns ActionExecute for children that need the
// engine to re-dispatch them. Pending children (e.g. a sync-gated task whose
// sibling just released the lock) are always re-dispatched.
//
// Running children are re-dispatched only when they are DAG or Steps boundary
// nodes: those only progress when their engine is re-entered, so the outer
// engine must keep visiting them each operate cycle until they reach a
// terminal phase. Running Pods/scripts are driven externally by the kube
// reconciler and must not be re-dispatched here.
func (e *DAGEvaluator) evaluateTaskGroupChild(taskName string, node *wfv1.NodeStatus) EvaluationResult {
	result := EvaluationResult{
		TaskName:     taskName,
		CurrentPhase: node.Phase,
	}
	if node.Fulfilled() {
		return result
	}
	switch node.Phase {
	case wfv1.NodePending:
		result.Action = ActionExecute
		result.ShouldRun = true
		result.ActionReason = "pending child re-dispatched"
	case wfv1.NodeRunning:
		if (node.Type == wfv1.NodeTypeDAG || node.Type == wfv1.NodeTypeSteps) && !node.IsDaemoned() {
			result.Action = ActionExecute
			result.ShouldRun = true
			result.ActionReason = "running DAG/Steps child re-dispatched"
		}
	}
	return result
}

// SetRetryStrategy registers a retry strategy for a task.
// Called by the engine after template resolution.
func (e *DAGEvaluator) SetRetryStrategy(taskName string, rs *wfv1.RetryStrategy) {
	e.retryStrategies[taskName] = rs
}

// nextRetryBackoff computes the delay until the next retry attempt based on
// the strategy's Backoff config and the number of attempts already made.
// Returns 0 if no backoff is configured, the duration cannot be parsed, or
// enough time has already elapsed since the last child finished.
func nextRetryBackoff(rs *wfv1.RetryStrategy, lastChild *wfv1.NodeStatus, attempts int) time.Duration {
	if rs == nil || rs.Backoff == nil || rs.Backoff.Duration == "" {
		return 0
	}
	base, err := time.ParseDuration(rs.Backoff.Duration)
	if err != nil {
		return 0
	}
	factor := int32(1)
	if rs.Backoff.Factor != nil {
		factor = max(rs.Backoff.Factor.IntVal, 1)
	}
	delay := base
	// Each prior failed attempt multiplies the delay by `factor`.
	// attempts is the number of children so far; the first retry uses the
	// base delay (no prior backoff window), so we start the loop at 1.
	for i := 1; i < attempts; i++ {
		delay *= time.Duration(factor)
	}
	if rs.Backoff.Cap != "" {
		if capD, err := time.ParseDuration(rs.Backoff.Cap); err == nil && delay > capD {
			delay = capD
		}
	}
	if rs.Backoff.MaxDuration != "" {
		if maxD, err := time.ParseDuration(rs.Backoff.MaxDuration); err == nil && delay > maxD {
			delay = maxD
		}
	}
	// Subtract time already elapsed since the last child finished.
	if lastChild != nil && !lastChild.FinishedAt.IsZero() {
		elapsed := time.Since(lastChild.FinishedAt.Time)
		delay -= elapsed
	}
	if delay < 0 {
		return 0
	}
	return delay
}

// evaluateRetryNode inspects a Retry node's children and returns what action
// should be taken. This is the pure-assessment equivalent of processNodeRetries
// in operator.go — it produces no side effects, only a result.
func (e *DAGEvaluator) evaluateRetryNode(_ context.Context, taskName string, node *wfv1.NodeStatus) EvaluationResult {
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
		rs := e.retryStrategies[taskName]
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
		rs := e.retryStrategies[taskName]
		if rs == nil {
			result.Action = ActionFail
			result.ActionReason = "no retry strategy configured"
			result.CurrentPhase = lastChild.Phase
			result.FulfilledForDeps = true
			return result
		}

		if !e.shouldRetry(lastChild, rs) {
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

// evaluateTaskGroupNode assesses a TaskGroup node (from withItems/withParam/withSequence)
// by checking if all children have completed.
func (e *DAGEvaluator) evaluateTaskGroupNode(ctx context.Context, taskName string, node *wfv1.NodeStatus) EvaluationResult {
	result := EvaluationResult{
		TaskName:     taskName,
		CurrentPhase: node.Phase,
	}

	if node.Fulfilled() {
		// Don't blindly trust a stale Succeeded phase — a daemon child may have
		// failed after the TaskGroup was marked Succeeded. For non-Succeeded terminal
		// phases (Failed, Error, etc.) the phase is truly final.
		if node.Phase != wfv1.NodeSucceeded {
			result.FulfilledForDeps = true
			return result
		}
		// For Succeeded nodes, fall through to re-verify children.
	}

	children := e.store.getTaskChildNodes(taskName)
	if len(children) == 0 {
		if node.Phase == wfv1.NodeSucceeded {
			// Children pruned/GC'd — trust the authoritative Succeeded phase.
			result.FulfilledForDeps = true
			return result
		}
		return result // still being expanded
	}

	if len(children) < len(node.Children) {
		if node.Phase == wfv1.NodeSucceeded {
			result.FulfilledForDeps = true
			return result
		}
		return result // still waiting for children to be created
	}

	allFulfilled := true
	anyFailed := false
	worstPhase := wfv1.NodeSucceeded
	for _, child := range children {
		// Daemoned running children haven't actually completed.
		if child.IsDaemoned() && !child.Phase.Fulfilled(child.TaskResultSynced) {
			allFulfilled = false
			continue
		}
		// Retry children: use evaluateRetryNode to detect exhausted retries.
		if child.Type == wfv1.NodeTypeRetry && !child.Fulfilled() {
			// Strip boundary prefix: retryStrategies is keyed by the
			// boundary-stripped task name, but child.Name is the full
			// prefixed node name (e.g. "dag.A-retry"). Matches the
			// sibling site in appendTaskGroupChildResults.
			retryTaskName := e.store.taskNameFromNodeName(child.Name)
			retryResult := e.evaluateRetryNode(ctx, retryTaskName, child)
			if retryResult.Action == ActionFail {
				anyFailed = true
				if retryResult.CurrentPhase == wfv1.NodeError {
					worstPhase = wfv1.NodeError
				} else if worstPhase != wfv1.NodeError {
					worstPhase = wfv1.NodeFailed
				}
			} else if !retryResult.FulfilledForDeps {
				allFulfilled = false
			}
			continue
		}
		if !child.Fulfilled() {
			allFulfilled = false
		}
		if child.Phase == wfv1.NodeFailed || child.Phase == wfv1.NodeError {
			anyFailed = true
			if child.Phase == wfv1.NodeError {
				worstPhase = wfv1.NodeError
			} else if worstPhase != wfv1.NodeError {
				worstPhase = wfv1.NodeFailed
			}
		}
	}

	if !allFulfilled {
		return result // children still running
	}

	if anyFailed {
		result.Action = ActionFail
		result.ActionReason = "child task failed"
		result.CurrentPhase = worstPhase
	} else {
		result.Action = ActionSucceed
		result.ActionReason = "all children completed"
		result.CurrentPhase = wfv1.NodeSucceeded
	}
	result.FulfilledForDeps = true
	return result
}

// shouldRetry determines if the retry policy allows retrying for the given
// child's terminal phase. When no explicit policy is set, the default depends
// on whether an expression is configured (see RetryPolicyActual).
func (e *DAGEvaluator) shouldRetry(lastChild *wfv1.NodeStatus, rs *wfv1.RetryStrategy) bool {
	policy := rs.RetryPolicyActual()
	switch policy {
	case wfv1.RetryPolicyAlways:
		return true
	case wfv1.RetryPolicyOnFailure:
		return lastChild.Phase == wfv1.NodeFailed
	case wfv1.RetryPolicyOnError:
		return lastChild.Phase == wfv1.NodeError
	case wfv1.RetryPolicyOnTransientError:
		// Simplified: treat both Failed and Error as retryable.
		// Full transient error detection requires error message inspection
		// which is outside the evaluator's scope.
		return lastChild.Phase == wfv1.NodeFailed || lastChild.Phase == wfv1.NodeError
	default:
		return false
	}
}
