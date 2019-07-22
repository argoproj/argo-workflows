package common

import (
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Context interface {
	// GetTaskNode returns the node status of a task.
	GetTaskNode(taskName string) *wfv1.NodeStatus
}

// GetTaskAncestry returns a list of taskNames which are ancestors of this task.
// The list is ordered by the tasks finished time.
func GetTaskAncestry(ctx Context, taskName string, tasks []wfv1.DAGTask) []string {
	taskByName := make(map[string]wfv1.DAGTask)
	for _, task := range tasks {
		taskByName[task.Name] = task
	}

	visited := make(map[string]time.Time)
	var getAncestry func(s string)
	getAncestry = func(currTask string) {
		task := taskByName[currTask]
		for _, depTask := range task.Dependencies {
			getAncestry(depTask)
		}
		if currTask != taskName {
			if _, ok := visited[currTask]; !ok {
				visited[currTask] = getTimeFinished(ctx, currTask)
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
