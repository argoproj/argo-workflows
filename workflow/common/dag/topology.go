package dag

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
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
	normalizedToOriginal := make(map[string]string, len(tasks))
	for i := range tasks {
		name := tasks[i].GetName()
		taskMap[name] = tasks[i]
		normalizedToOriginal[normalizeTaskName(name)] = name
	}

	dependencies := make(map[string][]string, len(tasks))
	dependsLogic := make(map[string]string, len(tasks))
	dependsErrors := make(map[string]error)

	taskProvider := func(name string) Task { return taskMap[name] }

	for _, task := range tasks {
		name := task.GetName()
		initialLogic := getTaskDependsLogic(task)
		deps, normalizedLogic, err := resolveDependencies(initialLogic, taskProvider)
		if err != nil {
			dependsErrors[name] = err
		}

		resolvedDeps := make([]string, len(deps))
		for i, dep := range deps {
			if original, ok := normalizedToOriginal[dep]; ok {
				resolvedDeps[i] = original
			} else {
				resolvedDeps[i] = dep
			}
		}

		dependencies[name] = resolvedDeps
		dependsLogic[name] = normalizedLogic
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
	names := make([]string, 0, len(w.taskMap))
	for name := range w.taskMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
// Parses and normalizes "depends" expressions, extracting dependency names and
// converting task names to hex-encoded identifiers for safe expression evaluation.

var (
	// taskNameRegex matches task names in depends expressions.
	// Supports expanded tasks like task(0) or task(0:item) and dotted results like task.Succeeded.
	taskNameRegex = regexp.MustCompile(`[a-zA-Z0-9\[\]\.\-_]+(\([^)]*\))?`)

	// exprKeywords are expression language keywords that should not be treated as task names.
	exprKeywords = map[string]bool{
		"true": true, "false": true,
		"nil": true, "null": true,
		"in": true, "not": true,
		"and": true, "or": true,
	}

	// validResults are the recognized result qualifiers for taskName.Result expressions.
	// Populated from taskResult struct fields via init().
	validResults map[string]bool
)

func init() {
	t := reflect.TypeFor[taskResult]()
	validResults = make(map[string]bool, t.NumField())
	for field := range t.Fields() {
		validResults[field.Name] = true
	}
}

// resolveDependencies parses a depends expression, extracts the unique dependency
// task names, and returns the normalized expression with hex-encoded task names.
// Returns an error if the expression contains an invalid result qualifier (e.g., "A.InvalidStatus").
func resolveDependencies(logic string, taskProvider func(string) Task) ([]string, string, error) {
	dependencySet := make(map[string]struct{})
	var resolveErr error

	newLogic := taskNameRegex.ReplaceAllStringFunc(logic, func(match string) string {
		// Skip expression language keywords
		if exprKeywords[match] {
			return match
		}
		// Check if it's a taskName.Result (e.g., "task.Succeeded")
		// Only the 8 known result qualifiers in validResults trigger the split.
		lastDot := strings.LastIndex(match, ".")
		if lastDot != -1 {
			potentialResult := match[lastDot+1:]
			if validResults[potentialResult] {
				taskName := match[:lastDot]

				dependencySet[taskName] = struct{}{}
				return fmt.Sprintf("%s.%s", normalizeTaskName(taskName), potentialResult)
			}
			// The qualifier is not recognized. If the prefix is a known task name,
			// this is an invalid qualifier reference (e.g., "A.InvalidStatus").
			// If the prefix is not a known task, the whole string is likely a
			// composite task name (e.g., "[0].A" for step tasks) — fall through.
			taskName := match[:lastDot]
			if taskProvider(taskName) != nil {
				resolveErr = fmt.Errorf("invalid depends qualifier %q in expression %q: valid qualifiers are Succeeded, Failed, Errored, Skipped, Omitted, Daemoned, AnySucceeded, AllFailed", potentialResult, match)
				return match
			}
		}

		// Bare taskName (e.g., "task") — expand to default depends expression
		taskName := match
		dependencySet[taskName] = struct{}{}

		task := taskProvider(taskName)
		return expandDependency(taskName, task)
	})

	deps := make([]string, 0, len(dependencySet))
	for dep := range dependencySet {
		deps = append(deps, dep)
	}
	sort.Strings(deps)

	return deps, newLogic, resolveErr
}

// expandDependency expands a bare task name into its default depends expression.
// A bare "taskA" becomes "(taskA.Succeeded || taskA.Skipped || taskA.Daemoned)",
// plus taskA.Errored/taskA.Failed if the task has continueOn set.
func expandDependency(depName string, depTask Task) string {
	normalizedName := normalizeTaskName(depName)
	resultForTask := func(result string) string { return fmt.Sprintf("%s.%s", normalizedName, result) }

	taskDepends := []string{
		resultForTask("Succeeded"),
		resultForTask("Skipped"),
		resultForTask("Daemoned"),
	}

	if depTask != nil {
		continueOn := depTask.GetContinueOn()
		if continueOn != nil {
			if continueOn.Error {
				taskDepends = append(taskDepends, resultForTask("Errored"))
			}
			if continueOn.Failed {
				taskDepends = append(taskDepends, resultForTask("Failed"))
			}
		}
	}

	return "(" + strings.Join(taskDepends, " || ") + ")"
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

// getTaskDependsLogic returns the depends expression for a task.
// If the task has an explicit "depends" field, it is returned directly.
// Otherwise, legacy "dependencies" are converted to a conjunction of expanded expressions.
func getTaskDependsLogic(task Task) string {
	if task.GetDepends() != "" {
		return task.GetDepends()
	}

	// For legacy dependencies, return raw task names joined with &&.
	// resolveDependencies will handle expansion (via expandDependency) and
	// normalization (via normalizeTaskName) in a single pass, avoiding
	// the double-encoding that occurs if we expand here and normalize later.
	deps := task.GetDependencies()
	if len(deps) == 0 {
		return ""
	}
	return strings.Join(deps, " && ")
}
