package common

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Context interface {
	// GetTaskNode returns the node status of a task.
	GetTaskNode(taskName string) *wfv1.NodeStatus
}

type TaskResult string

const (
	TaskResultSucceeded  TaskResult = "Succeeded"
	TaskResultFailed     TaskResult = "Failed"
	TaskResultSkipped    TaskResult = "Skipped"
	TaskResultCompleted  TaskResult = "Completed"
	TaskResultAny        TaskResult = "Any"
	TaskResultSuccessful TaskResult = "Successful"
)

type TaskDependency struct {
	TaskName   string
	TaskResult TaskResult
}

var (
	// TODO: This should use validate.workflowFieldNameFmt, but we can't import it here because an import cycle would be created
	taskNameRegex   = regexp.MustCompile(`([a-zA-Z0-9][-a-zA-Z0-9]*?\.[A-Z][a-z]+)|([a-zA-Z0-9][-a-zA-Z0-9]*)`)
	taskResultRegex = regexp.MustCompile(`([a-zA-Z0-9][-a-zA-Z0-9]*?\.[A-Z][a-z]+)`)
)

func GetTaskDependencies(dagTask *wfv1.DAGTask) []string {
	if dagTask.Depends != "" {
		return GetTaskDependenciesFromDepends(dagTask.Depends)
	}
	return dagTask.Dependencies
}

func GetTaskDependenciesFromDepends(depends string) []string {
	matches := taskNameRegex.FindAllStringSubmatch(depends, -1)
	var out []string
	for _, matchGroup := range matches {
		if matchGroup[1] != "" {
			split := strings.Split(matchGroup[1], ".")
			out = append(out, split[0])
		} else if matchGroup[2] != "" {
			out = append(out, matchGroup[2])
		}
	}
	return out
}

func GetTaskDependsLogic(dagTask *wfv1.DAGTask) string {
	if dagTask.Depends != "" {
		return dagTask.Depends
	}

	// For backwards compatibility, "dependencies: [A, B]" is equivalent to "depends: A.Successful && B.Successful"
	var dependencies []string
	for _, dependency := range dagTask.Dependencies {
		dependencies = append(dependencies, fmt.Sprintf("%s.%s", dependency, TaskResultSuccessful))
	}
	return strings.Join(dependencies, " && ")
}

// GetTaskAncestry returns a list of taskNames which are ancestors of this task.
// The list is ordered by the tasks finished time.
func GetTaskAncestry(ctx Context, taskName string, tasks []wfv1.DAGTask) []string {
	taskByName := make(map[string]wfv1.DAGTask)
	for _, task := range tasks {
		taskByName[task.Name] = task
	}
	visitedFlag := make(map[string]bool)
	visited := make(map[string]time.Time)
	var getAncestry func(s string)
	getAncestry = func(currTask string) {
		if !visitedFlag[currTask] {
			task := taskByName[currTask]
			for _, depTask := range GetTaskDependencies(&task) {
				getAncestry(depTask)
			}
			if currTask != taskName {
				if _, ok := visited[currTask]; !ok {
					visited[currTask] = getTimeFinished(ctx, currTask)
				}
				if currTask != taskName {
					if _, ok := visited[currTask]; !ok {
						visited[currTask] = getTimeFinished(ctx, currTask)
					}
				}
				visitedFlag[currTask] = true
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

func ValidateTaskResults(dagTask *wfv1.DAGTask) error {
	// If a user didn't specify a depends expression, there are no task results to validate
	if dagTask.Depends == "" {
		return nil
	}

	matches := taskResultRegex.FindAllStringSubmatch(dagTask.Depends, -1)
	for _, matchGroup := range matches {
		split := strings.Split(matchGroup[1], ".")
		taskName, taskResult := split[0], TaskResult(split[1])
		switch taskResult {
		case TaskResultSucceeded, TaskResultFailed, TaskResultSkipped, TaskResultCompleted, TaskResultAny,
			TaskResultSuccessful:
			// Do nothing
		default:
			return fmt.Errorf("task result '%s' for task '%s' is invalid", taskResult, taskName)
		}
	}
	return nil
}

// getTimeFinished returns the finishedAt time of the corresponding node.
// If the finished time is not set, the started time is returned.
// If ctx is not defined the current time is returned to ensure consistent order in the validation step.
func getTimeFinished(ctx Context, taskName string) time.Time {
	if ctx != nil {
		node := ctx.GetTaskNode(taskName)
		if !node.FinishedAt.IsZero() {
			return node.FinishedAt.Time
		}
		return node.StartedAt.Time
	} else {
		return time.Now()
	}
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
