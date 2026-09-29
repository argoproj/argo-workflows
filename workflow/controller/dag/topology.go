package dag

import (
	"context"
	"encoding/hex"
	"maps"
	"slices"
	"sort"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// dagTopology holds the immutable, pre-computed dependency graph for a set of tasks.
type dagTopology struct {
	// dependencies maps each task name to its dependency task names.
	dependencies map[string][]string
	// dependsLogic maps each task name to its normalized depends expression
	// (with task names hex-encoded for safe expression evaluation).
	dependsLogic map[string]string
	// topoOrder is the topologically sorted task names (dependencies before dependents).
	// Used by evaluateAllStates to evaluate tasks in dependency order (O(N) single pass)
	// instead of a fixed-point loop (O(N²) worst case for linear chains).
	topoOrder []string
	// dependsErrors maps task names to errors encountered while parsing their depends expressions.
	dependsErrors map[string]error
}

// WorkflowTasks holds the task collection and pre-computed topology for a DAG evaluation.
type WorkflowTasks struct {
	taskMap  map[string]Task
	topology *dagTopology
}

// newWorkflowTasks creates a new WorkflowTasks, computing the topology from the task definitions.
func newWorkflowTasks(tasks []Task) *WorkflowTasks {
	taskMap := make(map[string]Task, len(tasks))
	for i := range tasks {
		taskMap[tasks[i].GetName()] = tasks[i]
	}

	dependencies := make(map[string][]string, len(tasks))
	dependsLogic := make(map[string]string, len(tasks))
	dependsErrors := make(map[string]error)

	taskProvider := func(name string) Task { return taskMap[name] }

	for _, task := range tasks {
		name := task.GetName()
		deps, logic, err := resolveTaskDepends(task, taskProvider)
		if err != nil {
			dependsErrors[name] = err
		}
		dependencies[name] = deps
		dependsLogic[name] = logic
	}

	// Compute topological order using Kahn's algorithm so that
	// evaluateAllStates can process tasks in dependency order (O(N))
	// instead of using a fixed-point loop (O(N²) for linear chains).
	topoOrder := topologicalSort(dependencies)

	return &WorkflowTasks{
		taskMap: taskMap,
		topology: &dagTopology{
			dependencies:  dependencies,
			dependsLogic:  dependsLogic,
			topoOrder:     topoOrder,
			dependsErrors: dependsErrors,
		},
	}
}

// GetDependencies returns the dependency task names for a given task.
func (w *WorkflowTasks) GetDependencies(_ context.Context, key Key) ([]Key, error) {
	if deps, ok := w.topology.dependencies[key]; ok {
		return deps, nil
	}
	// Handle dynamic/expanded nodes like "task(0:item)"
	baseName := getBaseTaskName(key)
	if deps, ok := w.topology.dependencies[baseName]; ok {
		return deps, nil
	}
	return nil, nil
}

// GetDependsLogic returns the normalized depends expression for a task.
func (w *WorkflowTasks) GetDependsLogic(_ context.Context, taskName string) string {
	if logic, ok := w.topology.dependsLogic[taskName]; ok {
		return logic
	}
	baseName := getBaseTaskName(taskName)
	return w.topology.dependsLogic[baseName]
}

// GetDependsError returns any error encountered while parsing the depends expression for a task.
func (w *WorkflowTasks) GetDependsError(taskName string) error {
	if err, ok := w.topology.dependsErrors[taskName]; ok {
		return err
	}
	baseName := getBaseTaskName(taskName)
	return w.topology.dependsErrors[baseName]
}

// TaskNames returns all task names (sorted).
func (w *WorkflowTasks) TaskNames() []string {
	return slices.Sorted(maps.Keys(w.taskMap))
}

// GetTask returns the Task with the given name, or nil if not found.
func (w *WorkflowTasks) GetTask(name string) Task {
	return w.taskMap[name]
}

// TopologicalOrder returns task names sorted so that dependencies come before dependents.
func (w *WorkflowTasks) TopologicalOrder() []Key {
	return w.topology.topoOrder
}

// topologicalSort returns task names in dependency order using Kahn's algorithm.
// If the graph has a cycle, falls back to the input order (cycles are caught by validation).
func topologicalSort(dependencies map[string][]string) []string {
	inDegree := make(map[string]int, len(dependencies))
	dependents := make(map[string][]string, len(dependencies))

	for name := range dependencies {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
		for _, dep := range dependencies[name] {
			dependents[dep] = append(dependents[dep], name)
			inDegree[name]++
		}
	}

	// Seed queue with roots (no dependencies)
	queue := make([]string, 0, len(inDegree))
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // deterministic order among roots

	result := make([]string, 0, len(inDegree))
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		result = append(result, name)
		for _, dep := range dependents[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Cycle fallback: return whatever we have (validation catches cycles upstream)
	if len(result) < len(dependencies) {
		for name := range dependencies {
			found := slices.Contains(result, name)
			if !found {
				result = append(result, name)
			}
		}
	}

	return result
}

// --- Dependency resolution ---
// Depends expressions are tokenized with the grammar validation uses
// (common.ParseDepends), then task names are rewritten to hex-encoded
// identifiers so they are safe for expression evaluation.

// resolveTaskDepends returns a task's dependency names (sorted, unique) and
// its normalized depends expression.
//
// A legacy "dependencies" list (DAG tasks, and every Steps task, whose
// synthetic dependencies are named "[i].step") is structured data: each entry
// is expanded directly and never re-parsed as an expression. A user-written
// "depends" (DAG tasks only) goes through common.ParseDepends, so an
// expression cannot pass validation and be read differently here.
func resolveTaskDepends(task Task, taskProvider func(string) Task) ([]string, string, error) {
	continueOnFor := func(name string) *wfv1.ContinueOn {
		if dep := taskProvider(name); dep != nil {
			return dep.GetContinueOn()
		}
		return nil
	}

	if task.GetDepends() == "" {
		deps := task.GetDependencies()
		if len(deps) == 0 {
			return nil, "", nil
		}
		terms := make([]string, len(deps))
		for i, dep := range deps {
			terms[i] = common.ExpandDependency(dep, continueOnFor(dep), normalizeTaskName)
		}
		return slices.Compact(slices.Sorted(slices.Values(deps))), strings.Join(terms, " && "), nil
	}

	depends := task.GetDepends()
	refs, err := common.ParseDepends(depends)
	dependencySet := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		dependencySet[ref.Task] = struct{}{}
	}
	logic := common.RewriteDepends(depends, refs, func(ref common.DependsRef) string {
		if ref.Result != "" {
			return normalizeTaskName(ref.Task) + "." + string(ref.Result)
		}
		return common.ExpandDependency(ref.Task, continueOnFor(ref.Task), normalizeTaskName)
	})
	return slices.Sorted(maps.Keys(dependencySet)), logic, err
}

// normalizeTaskName converts a task name to a safe expression identifier.
// Uses "t" prefix + hex encoding (e.g., "my-task" -> "t6d792d7461736b").
// Hex encoding is preferred over simpler approaches (e.g., dash-to-underscore)
// because it is bijective — task names that differ only in special characters
// (e.g., "my-task" vs "my_task") won't collide after normalization.
func normalizeTaskName(name string) string {
	return "t" + hex.EncodeToString([]byte(name))
}

// getBaseTaskName extracts the base task name from an expanded task name (e.g., "task(0)" -> "task").
func getBaseTaskName(name string) string {
	if before, _, ok := strings.Cut(name, "("); ok {
		return before
	}
	return name
}

// PullOrder returns tasks in the order main's executeDAG visited them: from
// each target in turn (the given targets, as written on dag.target, or else
// the leaves by name when none is set), every task right after its
// dependencies, depth first. Tasks outside a target's ancestry are never
// dispatched, but the Engine still needs to iterate them for omissions and
// hooks, so they follow at the end, in their original order.
//
// executeDAG passes this order to Execute as its tasks argument, which every
// per-reconcile task-list walk (createOmittedNodes, ProcessAllTaskHooks, and,
// once the fixed-point loop is replaced by one ordered walk, dispatch itself)
// iterates directly. That already makes a same-pass chain of omissions
// created dependency-first, so each Omitted node's dependency node exists to
// link it under (fixes a Steps/DAG orphaned-omission bug, C54); dispatch
// order itself still comes from converge's own iteration until the ordered
// walk lands. Steps templates already walk in declaration order and never
// call PullOrder.
func PullOrder(tasks []Task, targets []string) []Task {
	w := newWorkflowTasks(tasks)
	if len(targets) == 0 {
		dependedOn := make(map[string]bool)
		for _, deps := range w.topology.dependencies {
			for _, d := range deps {
				dependedOn[d] = true
			}
		}
		for _, name := range w.TaskNames() {
			if !dependedOn[name] {
				targets = append(targets, name)
			}
		}
	}
	order := make([]Task, 0, len(tasks))
	seen := make(map[string]bool, len(tasks))
	var visit func(name string)
	visit = func(name string) {
		task := w.GetTask(name)
		if task == nil || seen[name] {
			return
		}
		seen[name] = true
		for _, dep := range w.topology.dependencies[name] {
			visit(dep)
		}
		order = append(order, task)
	}
	for _, name := range targets {
		visit(name)
	}
	for _, task := range tasks {
		visit(task.GetName())
	}
	return order
}
