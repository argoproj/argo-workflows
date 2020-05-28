package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestGetTaskDependenciesFromDepends(t *testing.T) {
	testTasks := []*wfv1.DAGTask{
		{
			Name: "task-1",
		},
		{
			Name: "task-2",
		},
		{
			Name: "task-3",
		},
	}

	ctx := &testContext{
		testTasks: testTasks,
	}

	task := &wfv1.DAGTask{Depends: "(task-1 || task-2.Succeeded) && !task-3.Succeeded"}
	deps, logic := GetTaskDependencies(task, ctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}
	assert.Equal(t, "((task-1.Succeeded || task-1.Skipped || task-1.Daemoned) || task-2.Succeeded) && !task-3.Succeeded", logic)

	task = &wfv1.DAGTask{Depends: "(task-1||task-2.Completed)&&!task-3.Failed"}
	deps, logic = GetTaskDependencies(task, ctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}
	assert.Equal(t, "((task-1.Succeeded || task-1.Skipped || task-1.Daemoned)||task-2.Completed)&&!task-3.Failed", logic)

	task = &wfv1.DAGTask{Depends: "(task-1 || task-1.Succeeded) && !task-1.Failed"}
	deps, logic = GetTaskDependencies(task, ctx)
	assert.Equal(t, []string{"task-1"}, deps)
	assert.Equal(t, "((task-1.Succeeded || task-1.Skipped || task-1.Daemoned) || task-1.Succeeded) && !task-1.Failed", logic)

	ctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Failed: true}
	task = &wfv1.DAGTask{Depends: "task-1"}
	deps, logic = GetTaskDependencies(task, ctx)
	assert.Equal(t, []string{"task-1"}, deps)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Failed)", logic)

	ctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Error: true}
	task = &wfv1.DAGTask{Depends: "task-1"}
	deps, logic = GetTaskDependencies(task, ctx)
	assert.Equal(t, []string{"task-1"}, deps)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Errored)", logic)

	ctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Failed: true, Error: true}
	task = &wfv1.DAGTask{Depends: "task-1"}
	deps, logic = GetTaskDependencies(task, ctx)
	assert.Equal(t, []string{"task-1"}, deps)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Errored || task-1.Failed)", logic)
}

func TestValidateTaskResults(t *testing.T) {
	task := &wfv1.DAGTask{Depends: "(task-1 || task-2.Succeeded) && !task-3"}
	err := ValidateTaskResults(task)
	assert.NoError(t, err)

	task = &wfv1.DAGTask{Depends: "(task-1.Completed || task-2.Succeeded) && !task-3.Skipped && task-2.Failed || task-6.Succeeded"}
	err = ValidateTaskResults(task)
	assert.NoError(t, err)

	task = &wfv1.DAGTask{Depends: "(task-1.DoeNotExist || task-2.Succeeded)"}
	err = ValidateTaskResults(task)
	assert.Error(t, err, "task result 'DoeNotExist' for task 'task-1' is invalid")
}

func TestGetTaskDependsLogic(t *testing.T) {
	testTasks := []*wfv1.DAGTask{
		{
			Name: "task-1",
		},
		{
			Name: "task-2",
		},
		{
			Name: "task-3",
		},
	}

	ctx := &testContext{
		testTasks: testTasks,
	}
	task := &wfv1.DAGTask{Depends: "(task-1 || task-2.Succeeded) && !task-3"}
	depends := getTaskDependsLogic(task, ctx)
	assert.Equal(t, "(task-1 || task-2.Succeeded) && !task-3", depends)

	task = &wfv1.DAGTask{Dependencies: []string{"task-1", "task-2"}}
	depends = getTaskDependsLogic(task, ctx)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned) && (task-2.Succeeded || task-2.Skipped || task-2.Daemoned)", depends)
}

type testContext struct {
	status    map[string]time.Time
	testTasks []*wfv1.DAGTask
}

func (d *testContext) GetTask(taskName string) *wfv1.DAGTask {
	for _, task := range d.testTasks {
		if task.Name == taskName {
			return task
		}
	}
	return nil
}

func (d *testContext) GetTaskDependencies(taskName string) []string {
	return d.GetTask(taskName).Dependencies
}

func (d *testContext) GetTaskFinishedAtTime(taskName string) time.Time {
	if finished, ok := d.status[taskName]; ok {
		return finished
	}
	return time.Now()
}

func TestGetTaskAncestryForValidation(t *testing.T) {
	type args struct {
		ctx      DagContext
		taskName string
	}

	testTasks := []*wfv1.DAGTask{
		{
			Name:         "task1",
			Dependencies: make([]string, 0),
		},
		{
			Name:         "task2",
			Dependencies: []string{"task1"},
		},
		{
			Name:         "task3",
			Dependencies: []string{"task1"},
		},
		{
			Name:         "task4",
			Dependencies: []string{"task2", "task3"},
		},
	}

	ctx := &testContext{
		testTasks: testTasks,
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "one task",
			args: args{
				ctx:      ctx,
				taskName: "task2",
			},
			want: []string{"task1"},
		},
		{
			name: "multiple tasks",
			args: args{
				ctx:      ctx,
				taskName: "task4",
			},
			want: []string{"task1", "task2", "task3"},
		},
	}
	for _, tt := range tests {
		res := GetTaskAncestry(tt.args.ctx, tt.args.taskName)
		assert.Equal(t, tt.want, res)
	}
}

func TestGetTaskAncestryForGlobalArtifacts(t *testing.T) {
	type args struct {
		ctx      DagContext
		taskName string
	}

	testTasks := []*wfv1.DAGTask{
		{
			Name:         "task1",
			Dependencies: make([]string, 0),
		},
		{
			Name:         "task2",
			Dependencies: []string{"task1"},
		},
		{
			Name:         "task3",
			Dependencies: []string{"task1"},
		},
		{
			Name:         "task4",
			Dependencies: []string{"task2", "task3"},
		},
	}

	ctx := &testContext{
		testTasks: testTasks,
		status: map[string]time.Time{
			"task1": time.Now().Add(1 * time.Minute),
			"task2": time.Now().Add(3 * time.Minute),
			"task3": time.Now().Add(2 * time.Minute),
			"task4": time.Now().Add(4 * time.Minute),
		},
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "one task",
			args: args{
				ctx:      ctx,
				taskName: "task2",
			},
			want: []string{"task1"},
		},
		{
			name: "multiple tasks",
			args: args{
				ctx:      ctx,
				taskName: "task4",
			},
			want: []string{"task1", "task3", "task2"},
		},
	}
	for _, tt := range tests {
		res := GetTaskAncestry(tt.args.ctx, tt.args.taskName)
		assert.Equal(t, tt.want, res)
	}
}
