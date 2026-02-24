package common

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
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

	dctx := &testContext{
		testTasks: testTasks,
	}
	ctx := logging.TestContext(t.Context())

	task := &wfv1.DAGTask{Depends: "(task-1 || task-2.Succeeded) && !task-3.Succeeded"}
	deps, logic := GetTaskDependencies(ctx, task, dctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}
	assert.Equal(t, "((task-1.Succeeded || task-1.Skipped || task-1.Daemoned) || task-2.Succeeded) && !task-3.Succeeded", logic)

	task = &wfv1.DAGTask{Depends: "(task-1 || task-2.AnySucceeded) && !task-3.Succeeded"}
	deps, logic = GetTaskDependencies(ctx, task, dctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}
	assert.Equal(t, "((task-1.Succeeded || task-1.Skipped || task-1.Daemoned) || task-2.AnySucceeded) && !task-3.Succeeded", logic)

	task = &wfv1.DAGTask{Depends: "(task-1||(task-2.Succeeded || task-2.Failed))&&!task-3.Failed"}
	deps, logic = GetTaskDependencies(ctx, task, dctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}
	assert.Equal(t, "((task-1.Succeeded || task-1.Skipped || task-1.Daemoned)||(task-2.Succeeded || task-2.Failed))&&!task-3.Failed", logic)

	task = &wfv1.DAGTask{Depends: "(task-1 || task-1.Succeeded) && !task-1.Failed"}
	deps, logic = GetTaskDependencies(ctx, task, dctx)
	assert.Equal(t, map[string]DependencyType{"task-1": DependencyTypeTask}, deps)
	assert.Equal(t, "((task-1.Succeeded || task-1.Skipped || task-1.Daemoned) || task-1.Succeeded) && !task-1.Failed", logic)

	task = &wfv1.DAGTask{Depends: "task-1.Succeeded && task-1.AnySucceeded"}
	deps, logic = GetTaskDependencies(ctx, task, dctx)
	assert.Equal(t, map[string]DependencyType{"task-1": DependencyTypeItems}, deps)
	assert.Equal(t, "task-1.Succeeded && task-1.AnySucceeded", logic)

	dctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Failed: true}
	task = &wfv1.DAGTask{Depends: "task-1"}
	deps, logic = GetTaskDependencies(ctx, task, dctx)
	assert.Equal(t, map[string]DependencyType{"task-1": DependencyTypeTask}, deps)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Failed)", logic)

	dctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Error: true}
	task = &wfv1.DAGTask{Depends: "task-1"}
	deps, logic = GetTaskDependencies(ctx, task, dctx)
	assert.Equal(t, map[string]DependencyType{"task-1": DependencyTypeTask}, deps)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Errored)", logic)

	dctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Failed: true, Error: true}
	task = &wfv1.DAGTask{Depends: "task-1"}
	deps, logic = GetTaskDependencies(ctx, task, dctx)
	assert.Equal(t, map[string]DependencyType{"task-1": DependencyTypeTask}, deps)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Errored || task-1.Failed)", logic)
}

func TestValidateTaskResults(t *testing.T) {
	task := &wfv1.DAGTask{Depends: "(task-1 || task-2.Succeeded) && !task-3"}
	err := ValidateTaskResults(task)
	require.NoError(t, err)

	task = &wfv1.DAGTask{Depends: "((task-1.Succeeded || task-1.Failed) || task-2.Succeeded) && !task-3.Skipped && task-2.Failed || task-6.Succeeded"}
	err = ValidateTaskResults(task)
	require.NoError(t, err)

	task = &wfv1.DAGTask{Depends: "((task-1.Succeeded || task-1.Omitted) || task-2.Succeeded) && !task-3.Skipped && task-2.Failed || task-6.Succeeded"}
	err = ValidateTaskResults(task)
	require.NoError(t, err)

	task = &wfv1.DAGTask{Depends: "(task-1.DoeNotExist || task-2.Succeeded)"}
	err = ValidateTaskResults(task)
	require.Error(t, err, "task result 'DoeNotExist' for task 'task-1' is invalid")
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

	dctx := &testContext{
		testTasks: testTasks,
	}
	ctx := logging.TestContext(t.Context())
	task := &wfv1.DAGTask{Depends: "(task-1 || task-2.Succeeded) && !task-3"}
	depends := getTaskDependsLogic(ctx, task, dctx)
	assert.Equal(t, "(task-1 || task-2.Succeeded) && !task-3", depends)

	task = &wfv1.DAGTask{Dependencies: []string{"task-1", "task-2"}}
	depends = getTaskDependsLogic(ctx, task, dctx)
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned) && (task-2.Succeeded || task-2.Skipped || task-2.Daemoned)", depends)
}

type testContext struct {
	status    map[string]time.Time
	testTasks []*wfv1.DAGTask
}

func (d *testContext) GetTask(ctx context.Context, taskName string) *wfv1.DAGTask {
	for _, task := range d.testTasks {
		if task.Name == taskName {
			return task
		}
	}
	return nil
}

func (d *testContext) GetTaskDependencies(ctx context.Context, taskName string) []string {
	return d.GetTask(ctx, taskName).Dependencies
}

func (d *testContext) GetTaskFinishedAtTime(ctx context.Context, taskName string) time.Time {
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

	now := time.Now()
	dctx := &testContext{
		testTasks: testTasks,
		status: map[string]time.Time{
			"task1": now.Add(1 * time.Minute),
			"task2": now.Add(2 * time.Minute),
			"task3": now.Add(3 * time.Minute),
			"task4": now.Add(4 * time.Minute),
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
				ctx:      dctx,
				taskName: "task2",
			},
			want: []string{"task1"},
		},
		{
			name: "multiple tasks",
			args: args{
				ctx:      dctx,
				taskName: "task4",
			},
			want: []string{"task1", "task2", "task3"},
		},
	}
	ctx := logging.TestContext(t.Context())
	for _, tt := range tests {
		res := GetTaskAncestry(ctx, tt.args.ctx, tt.args.taskName)
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

	dctx := &testContext{
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
				ctx:      dctx,
				taskName: "task2",
			},
			want: []string{"task1"},
		},
		{
			name: "multiple tasks",
			args: args{
				ctx:      dctx,
				taskName: "task4",
			},
			want: []string{"task1", "task3", "task2"},
		},
	}
	ctx := logging.TestContext(t.Context())
	for _, tt := range tests {
		res := GetTaskAncestry(ctx, tt.args.ctx, tt.args.taskName)
		assert.Equal(t, tt.want, res)
	}
}
