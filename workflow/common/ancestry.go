package common

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type DagContext interface {
	GetTask(ctx context.Context, taskName string) *wfv1.DAGTask
	GetTaskDependencies(ctx context.Context, taskName string) []string
	GetTaskFinishedAtTime(ctx context.Context, taskName string) time.Time
}

type TaskResult string

const (
	TaskResultSucceeded    TaskResult = "Succeeded"
	TaskResultFailed       TaskResult = "Failed"
	TaskResultErrored      TaskResult = "Errored"
	TaskResultSkipped      TaskResult = "Skipped"
	TaskResultOmitted      TaskResult = "Omitted"
	TaskResultDaemoned     TaskResult = "Daemoned"
	TaskResultAnySucceeded TaskResult = "AnySucceeded"
	TaskResultAllFailed    TaskResult = "AllFailed"
)

var (
	// TODO: This should use validate.workflowFieldNameFmt, but we can't import it here because an import cycle would be created
	taskNameRegex = regexp.MustCompile(`([a-zA-Z0-9][-a-zA-Z0-9]*?\.[A-Z][a-zA-Z]+)|([a-zA-Z0-9][-a-zA-Z0-9]*)`)
)

type DependencyType int

const (
	DependencyTypeTask DependencyType = iota
	DependencyTypeItems
)

// DependsRef is one task reference in a depends expression: "task.Result"
// (Result set) or a bare "task" (Result empty). Start/End are the byte
// offsets of the reference within the expression.
type DependsRef struct {
	Task   string
	Result TaskResult
	Start  int
	End    int
}

// ParseDepends tokenizes a depends expression into its task references. It is
// the single grammar for depends: workflow validation and the DAG evaluator
// both use it, so an expression cannot pass validation and be read differently
// at runtime. All references are returned even when an unrecognized result
// qualifier is found; that qualifier is reported as the error.
func ParseDepends(depends string) ([]DependsRef, error) {
	matches := taskNameRegex.FindAllStringSubmatchIndex(depends, -1)
	refs := make([]DependsRef, 0, len(matches))
	var err error
	for _, m := range matches {
		switch {
		case m[2] != -1: // taskName.TaskResult
			match := depends[m[2]:m[3]]
			split := strings.Split(match, ".")
			taskName, result := split[0], TaskResult(split[1])
			switch result {
			case TaskResultSucceeded, TaskResultFailed, TaskResultSkipped, TaskResultOmitted, TaskResultErrored, TaskResultDaemoned, TaskResultAnySucceeded, TaskResultAllFailed:
			default:
				if err == nil {
					err = fmt.Errorf("task result '%s' for task '%s' is invalid", result, taskName)
				}
			}
			refs = append(refs, DependsRef{Task: taskName, Result: result, Start: m[2], End: m[3]})
		case m[4] != -1: // bare taskName
			refs = append(refs, DependsRef{Task: depends[m[4]:m[5]], Start: m[4], End: m[5]})
		}
	}
	return refs, err
}

// RewriteDepends returns the expression with every reference replaced by
// rewrite(ref). References are spliced right to left so offsets stay valid.
func RewriteDepends(depends string, refs []DependsRef, rewrite func(DependsRef) string) string {
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		depends = depends[:ref.Start] + rewrite(ref) + depends[ref.End:]
	}
	return depends
}

func GetTaskDependencies(ctx context.Context, task *wfv1.DAGTask, dctx DagContext) (map[string]DependencyType, string) {
	depends := getTaskDependsLogic(ctx, task, dctx)
	// Invalid result qualifiers are reported by ValidateTaskResults.
	refs, _ := ParseDepends(depends)
	dependencies := make(map[string]DependencyType)
	for _, ref := range refs {
		switch {
		case ref.Result == "":
			dependencies[ref.Task] = DependencyTypeTask
		case ref.Result == TaskResultAnySucceeded || ref.Result == TaskResultAllFailed:
			dependencies[ref.Task] = DependencyTypeItems
		default:
			if _, ok := dependencies[ref.Task]; !ok { // DependencyTypeItems takes precedence
				dependencies[ref.Task] = DependencyTypeTask
			}
		}
	}
	// For backwards compatibility, a bare task reference expands to the task
	// having completed in any non-failing way (plus continueOn allowances).
	expanded := RewriteDepends(depends, refs, func(ref DependsRef) string {
		if ref.Result != "" {
			return depends[ref.Start:ref.End]
		}
		return expandDependency(ref.Task, dctx.GetTask(ctx, ref.Task))
	})
	return dependencies, expanded
}

func ValidateTaskResults(dagTask *wfv1.DAGTask) error {
	// If a user didn't specify a depends expression, there are no task results to validate
	if dagTask.Depends == "" {
		return nil
	}
	_, err := ParseDepends(dagTask.Depends)
	return err
}

func getTaskDependsLogic(ctx context.Context, dagTask *wfv1.DAGTask, dctx DagContext) string {
	if dagTask.Depends != "" {
		return dagTask.Depends
	}

	// For backwards compatibility, "dependencies: [A, B]" is equivalent to "depends: (A.Successful || A.Skipped || A.Daemoned)) && (B.Successful || B.Skipped || B.Daemoned)"
	var dependencies []string
	for _, dependency := range dagTask.Dependencies {
		depTask := dctx.GetTask(ctx, dependency)
		dependencies = append(dependencies, expandDependency(dependency, depTask))
	}
	return strings.Join(dependencies, " && ")
}

func expandDependency(depName string, depTask *wfv1.DAGTask) string {
	var continueOn *wfv1.ContinueOn
	if depTask != nil {
		continueOn = depTask.ContinueOn
	}
	return ExpandDependency(depName, continueOn, func(name string) string { return name })
}

// ExpandDependency expands a bare task reference into its default depends
// expression: the task Succeeded, was Skipped or is Daemoned, plus Errored /
// Failed when the dependency has the matching continueOn set. ident rewrites
// the task name in the output (identity for validation; the DAG evaluator
// encodes names into identifiers that are safe for expression evaluation).
func ExpandDependency(depName string, continueOn *wfv1.ContinueOn, ident func(string) string) string {
	name := ident(depName)
	resultForTask := func(result TaskResult) string { return fmt.Sprintf("%s.%s", name, result) }

	taskDepends := []string{resultForTask(TaskResultSucceeded), resultForTask(TaskResultSkipped), resultForTask(TaskResultDaemoned)}
	if continueOn != nil {
		if continueOn.Error {
			taskDepends = append(taskDepends, resultForTask(TaskResultErrored))
		}
		if continueOn.Failed {
			taskDepends = append(taskDepends, resultForTask(TaskResultFailed))
		}
	}
	return "(" + strings.Join(taskDepends, " || ") + ")"
}

// GetTaskAncestry returns a list of taskNames which are ancestors of this task.
// The list is ordered by the tasks finished time.
func GetTaskAncestry(ctx context.Context, dctx DagContext, taskName string) []string {
	visited := make(map[string]time.Time)

	var getAncestry func(currTask string)
	getAncestry = func(currTask string) {
		if _, seen := visited[currTask]; seen {
			return
		}
		for _, depTask := range dctx.GetTaskDependencies(ctx, currTask) {
			getAncestry(depTask)
		}
		if currTask != taskName {
			if _, ok := visited[currTask]; !ok {
				visited[currTask] = dctx.GetTaskFinishedAtTime(ctx, currTask)
			}
		}
	}

	getAncestry(taskName)

	ancestry := make([]string, len(visited))
	for newTask, newFinishedAt := range visited {
		insertTask(visited, ancestry, newTask, newFinishedAt)
	}

	return ancestry
}

// insertTask inserts the newTaskName at the right position ordered by time into the ancestry list.
func insertTask(visited map[string]time.Time, ancestry []string, newTaskName string, finishedAt time.Time) {
	for i, taskName := range ancestry {
		if taskName == "" {
			ancestry[i] = newTaskName
			return
		}
		if finishedAt.Before(visited[taskName]) {
			// insert at position i and shift others
			copy(ancestry[i+1:], ancestry[i:])
			ancestry[i] = newTaskName
			return
		}
	}
}
