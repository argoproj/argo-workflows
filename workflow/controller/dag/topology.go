package dag

import (
	"context"
	"encoding/hex"
	"maps"
	"slices"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// dagTopology holds the immutable, pre-computed dependency graph for a set of tasks.
type dagTopology struct {
	// dependencies maps each task name to its dependency task names.
	dependencies map[string][]string
	// dependsLogic maps each task name to its normalized depends expression
	// (with task names hex-encoded for safe expression evaluation). It is
	// empty for a task with a legacy "dependencies" list.
	dependsLogic map[string]string
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

	// Every step of a Steps group depends on all the steps of the group
	// before it, so consecutive tasks with the same dependencies share one
	// resolution instead of each holding its own copy.
	var (
		deps  []string
		logic string
		err   error
	)
	for i, task := range tasks {
		name := task.GetName()
		if i == 0 || !sameDepends(task, tasks[i-1]) {
			deps, logic, err = resolveTaskDepends(task, taskProvider)
		}
		if err != nil {
			dependsErrors[name] = err
		}
		dependencies[name] = deps
		dependsLogic[name] = logic
	}

	return &WorkflowTasks{
		taskMap: taskMap,
		topology: &dagTopology{
			dependencies:  dependencies,
			dependsLogic:  dependsLogic,
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

// LeafTaskNames returns the names of tasks that no other task depends on,
// sorted. Used both as the implicit dag.target (DAGEvaluator.FindLeafTaskNames,
// via assessDAGPhase) and as PullOrder's default targets when the template sets
// no explicit dag.target.
func (w *WorkflowTasks) LeafTaskNames() []string {
	dependedOn := make(map[string]bool, len(w.topology.dependencies))
	for _, deps := range w.topology.dependencies {
		for _, dep := range deps {
			dependedOn[dep] = true
		}
	}
	var leaves []string
	for _, name := range w.TaskNames() {
		if !dependedOn[name] {
			leaves = append(leaves, name)
		}
	}
	return leaves
}

// --- Dependency resolution ---
// Depends expressions are tokenized with the grammar validation uses
// (common.ParseDepends), then task names are rewritten to hex-encoded
// identifiers so they are safe for expression evaluation.

// resolveTaskDepends returns a task's dependency names (sorted, unique) and
// its normalized depends expression.
//
// A legacy "dependencies" list (DAG tasks, and every Steps task, whose
// synthetic dependencies are named "[i].step") is structured data: it gets no
// expression, and the evaluator checks each entry directly (dependencyMet). A
// user-written "depends" (DAG tasks only) goes through common.ParseDepends, so
// an expression cannot pass validation and be read differently here.
func resolveTaskDepends(task Task, taskProvider func(string) Task) ([]string, string, error) {
	continueOnFor := func(name string) *wfv1.ContinueOn {
		if dep := taskProvider(name); dep != nil {
			return dep.GetContinueOn()
		}
		return nil
	}

	if task.GetDepends() == "" {
		// Legacy dependencies are not made into an expression: a step depends
		// on every step of the group before it, and an expression over all of
		// them grows with the group and soon exceeds what expr-lang compiles.
		deps := task.GetDependencies()
		if len(deps) == 0 {
			return nil, "", nil
		}
		return slices.Compact(slices.Sorted(slices.Values(deps))), "", nil
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

// sameDepends reports whether two tasks declare the same dependencies, which
// resolveTaskDepends resolves to the same result.
func sameDepends(a, b Task) bool {
	return a.GetDepends() == b.GetDepends() && slices.Equal(a.GetDependencies(), b.GetDependencies())
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

// PullOrder returns the tasks a DAG's walk visits, in the order it visits
// them: from each target in turn (the given targets, as written on
// dag.target, or else the leaves by name when none is set), every task right
// after its dependencies, depth first. A task outside every target's
// ancestry is left out: nothing needs it, so it is never dispatched, and it
// has no node to omit, assess or drive hooks for.
//
// executeDAG passes this order to Execute as its tasks argument, and the
// Engine's walk visits the tasks in it, so each task is evaluated after
// everything it depends on has been dispatched, omitted and had its hooks
// driven in the same walk. Steps templates already come in
// dependency order, as written, and never call PullOrder.
func PullOrder(tasks []Task, targets []string) []Task {
	w := newWorkflowTasks(tasks)
	if len(targets) == 0 {
		targets = w.LeafTaskNames()
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
	return order
}
