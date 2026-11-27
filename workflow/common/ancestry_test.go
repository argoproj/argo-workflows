package common

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
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
	deps := GetTaskDependencies(ctx, task, dctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}

	task = &wfv1.DAGTask{Depends: "(task-1 || task-2.AnySucceeded) && !task-3.Succeeded"}
	deps = GetTaskDependencies(ctx, task, dctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}

	task = &wfv1.DAGTask{Depends: "(task-1||(task-2.Succeeded || task-2.Failed))&&!task-3.Failed"}
	deps = GetTaskDependencies(ctx, task, dctx)
	assert.Len(t, deps, 3)
	for _, dep := range []string{"task-1", "task-2", "task-3"} {
		assert.Contains(t, deps, dep)
	}

	task = &wfv1.DAGTask{Depends: "(task-1 || task-1.Succeeded) && !task-1.Failed"}
	deps = GetTaskDependencies(ctx, task, dctx)
	assert.Equal(t, map[string]DependencyType{"task-1": DependencyTypeTask}, deps)

	task = &wfv1.DAGTask{Depends: "task-1.Succeeded && task-1.AnySucceeded"}
	deps = GetTaskDependencies(ctx, task, dctx)
	assert.Equal(t, map[string]DependencyType{"task-1": DependencyTypeItems}, deps)
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

	// A dependency's continueOn adds the results it continues on.
	task = &wfv1.DAGTask{Dependencies: []string{"task-1"}}
	dctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Failed: true}
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Failed)", getTaskDependsLogic(ctx, task, dctx))
	dctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Error: true}
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Errored)", getTaskDependsLogic(ctx, task, dctx))
	dctx.testTasks[0].ContinueOn = &wfv1.ContinueOn{Failed: true, Error: true}
	assert.Equal(t, "(task-1.Succeeded || task-1.Skipped || task-1.Daemoned || task-1.Errored || task-1.Failed)", getTaskDependsLogic(ctx, task, dctx))
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

func TestParseDepends(t *testing.T) {
	refs, err := ParseDepends("A && (B.Failed || C.AnySucceeded) && !D")
	require.NoError(t, err)
	assert.Equal(t, []DependsRef{
		{Task: "A", Start: 0, End: 1},
		{Task: "B", Result: TaskResultFailed, Start: 6, End: 14},
		{Task: "C", Result: TaskResultAnySucceeded, Start: 18, End: 32},
		{Task: "D", Start: 38, End: 39},
	}, refs)

	rewritten := RewriteDepends("A && (B.Failed || C.AnySucceeded) && !D", refs, func(ref DependsRef) string {
		if ref.Result == "" {
			return "<" + ref.Task + ">"
		}
		return "<" + ref.Task + "." + string(ref.Result) + ">"
	})
	assert.Equal(t, "<A> && (<B.Failed> || <C.AnySucceeded>) && !<D>", rewritten)

	refs, err = ParseDepends("A.Bogus && B")
	require.EqualError(t, err, "task result 'Bogus' for task 'A' is invalid")
	assert.Len(t, refs, 2, "all references are still returned on error")

	assert.Equal(t, "(dep.Succeeded || dep.Skipped || dep.Daemoned || dep.Failed)",
		ExpandDependency("dep", &wfv1.ContinueOn{Failed: true}, func(name string) string { return name }))
	assert.Equal(t, "(X.Succeeded || X.Skipped || X.Daemoned)",
		ExpandDependency("dep", nil, func(string) string { return "X" }))
}

// mapDagContext is a DagContext over a map, so lookups stay cheap for large DAGs.
type mapDagContext struct {
	tasks      map[string]*wfv1.DAGTask
	finishedAt map[string]time.Time
}

func (d *mapDagContext) GetTask(_ context.Context, taskName string) *wfv1.DAGTask {
	return d.tasks[taskName]
}

func (d *mapDagContext) GetTaskDependencies(_ context.Context, taskName string) []string {
	return d.tasks[taskName].Dependencies
}

func (d *mapDagContext) GetTaskFinishedAtTime(_ context.Context, taskName string) time.Time {
	return d.finishedAt[taskName]
}

// chainDagContext returns a DAG of n tasks where each task depends on the one before it.
func chainDagContext(n int) *mapDagContext {
	now := time.Now()
	d := &mapDagContext{tasks: make(map[string]*wfv1.DAGTask, n), finishedAt: make(map[string]time.Time, n)}
	for i := range n {
		task := &wfv1.DAGTask{Name: fmt.Sprintf("task%d", i)}
		if i > 0 {
			task.Dependencies = []string{fmt.Sprintf("task%d", i-1)}
		}
		d.tasks[task.Name] = task
		d.finishedAt[task.Name] = now.Add(time.Duration(i) * time.Second)
	}
	return d
}

func TestGetTaskAncestryLongChain(t *testing.T) {
	const n = 30000
	ctx := logging.TestContext(t.Context())
	res := GetTaskAncestry(ctx, chainDagContext(n), fmt.Sprintf("task%d", n-1))
	require.Len(t, res, n-1)
	for i, name := range res {
		assert.Equal(t, fmt.Sprintf("task%d", i), name)
	}
}

// insertionTaskAncestry is the previous GetTaskAncestry, which kept the
// ancestry ordered by inserting each task into place.
func insertionTaskAncestry(ctx context.Context, dctx DagContext, taskName string) []string {
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
	for newTaskName, finishedAt := range visited {
		for i, name := range ancestry {
			if name == "" {
				ancestry[i] = newTaskName
				break
			}
			if finishedAt.Before(visited[name]) {
				copy(ancestry[i+1:], ancestry[i:])
				ancestry[i] = newTaskName
				break
			}
		}
	}
	return ancestry
}

func TestGetTaskAncestryMatchesInsertion(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	rng := rand.New(rand.NewPCG(1, 2))
	now := time.Now()
	for range 200 {
		n := 1 + rng.IntN(40)
		// A small range of finish times gives ties, which both versions break by map iteration order.
		spread := 1 + rng.IntN(2*n)
		d := &mapDagContext{tasks: make(map[string]*wfv1.DAGTask, n), finishedAt: make(map[string]time.Time, n)}
		for i := range n {
			task := &wfv1.DAGTask{Name: fmt.Sprintf("task%d", i)}
			for j := range i {
				if rng.IntN(4) == 0 {
					task.Dependencies = append(task.Dependencies, fmt.Sprintf("task%d", j))
				}
			}
			d.tasks[task.Name] = task
			d.finishedAt[task.Name] = now.Add(time.Duration(rng.IntN(spread)) * time.Second)
		}
		byTime := func(a, b string) int { return d.finishedAt[a].Compare(d.finishedAt[b]) }
		byTimeThenName := func(a, b string) int {
			if c := byTime(a, b); c != 0 {
				return c
			}
			return strings.Compare(a, b)
		}
		for name := range d.tasks {
			got := GetTaskAncestry(ctx, d, name)
			want := insertionTaskAncestry(ctx, d, name)
			assert.True(t, slices.IsSortedFunc(got, byTime), "ancestry of %s not ordered by finish time: %v", name, got)
			assert.True(t, slices.IsSortedFunc(want, byTime))
			slices.SortFunc(got, byTimeThenName)
			slices.SortFunc(want, byTimeThenName)
			assert.Equal(t, want, got, "ancestry of %s", name)
		}
	}
}

func BenchmarkGetTaskAncestry(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(fmt.Sprintf("chain-%d", n), func(b *testing.B) {
			ctx := logging.TestContext(b.Context())
			d := chainDagContext(n)
			last := fmt.Sprintf("task%d", n-1)
			for b.Loop() {
				GetTaskAncestry(ctx, d, last)
			}
		})
	}
}
