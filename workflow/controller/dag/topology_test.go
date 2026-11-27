package dag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// taskNames returns the names of tasks in order, for asserting PullOrder's result.
func taskNames(tasks []Task) []string {
	names := make([]string, len(tasks))
	for i, task := range tasks {
		names[i] = task.GetName()
	}
	return names
}

func TestPullOrder(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []wfv1.DAGTask
		targets []string
		want    []string
	}{
		{
			name: "independent tasks with no target are visited leaf-name order",
			tasks: []wfv1.DAGTask{
				{Name: "c"},
				{Name: "b"},
				{Name: "a"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "a dependency chain declared in reverse is walked dependency-first",
			tasks: []wfv1.DAGTask{
				{Name: "C", Depends: "B"},
				{Name: "B", Depends: "A"},
				{Name: "A"},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "legacy dependencies list, declared in reverse",
			tasks: []wfv1.DAGTask{
				{Name: "C", Dependencies: []string{"B"}},
				{Name: "B", Dependencies: []string{"A"}},
				{Name: "A"},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "a diamond dependency visits the shared ancestor once",
			tasks: []wfv1.DAGTask{
				{Name: "d", Depends: "b && c"},
				{Name: "c", Depends: "a"},
				{Name: "b", Depends: "a"},
				{Name: "a"},
			},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "an explicit target pulls only its own ancestry",
			tasks: []wfv1.DAGTask{
				{Name: "z", Depends: "y"},
				{Name: "y", Depends: "w"},
				{Name: "x"},
				{Name: "w"},
			},
			targets: []string{"y"},
			want:    []string{"w", "y"},
		},
		{
			name: "several targets are visited in the order given",
			tasks: []wfv1.DAGTask{
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
			},
			targets: []string{"c", "a", "b"},
			want:    []string{"c", "a", "b"},
		},
		{
			name: "an unknown target name pulls nothing",
			tasks: []wfv1.DAGTask{
				{Name: "a"},
				{Name: "b"},
			},
			targets: []string{"nonexistent", "b"},
			want:    []string{"b"},
		},
		{
			name:  "no tasks",
			tasks: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := taskNames(PullOrder(toTasks(tt.tasks), tt.targets))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewWorkflowTasks_SharesEqualDependencies(t *testing.T) {
	// As executeSteps builds them: every step of a group carries the names of
	// all the steps of the group before it.
	group0 := []string{"[0].a", "[0].b"}
	group1 := []string{"[1].c", "[1].d"}
	tasks := toTasks([]wfv1.DAGTask{
		{Name: "[0].a"},
		{Name: "[0].b", ContinueOn: &wfv1.ContinueOn{Failed: true}},
		{Name: "[1].c", Dependencies: group0},
		{Name: "[1].d", Dependencies: group0},
		{Name: "[2].e", Dependencies: group1},
		{Name: "bad", Depends: "x.Unknown"},
		{Name: "bad-too", Depends: "x.Unknown"},
		{Name: "good", Depends: "x"},
	})
	w := newWorkflowTasks(tasks)

	// Every task resolves as it does on its own.
	for _, task := range tasks {
		name := task.GetName()
		deps, logic, err := resolveTaskDepends(task, w.GetTask)
		assert.Equal(t, deps, w.topology.dependencies[name], name)
		assert.Equal(t, logic, w.topology.dependsLogic[name], name)
		assert.Equal(t, err, w.GetDependsError(name), name)
	}
	require.Error(t, w.GetDependsError("bad"))
	require.Error(t, w.GetDependsError("bad-too"))
	require.NoError(t, w.GetDependsError("good"))

	// The steps of a group hold one resolution between them, not one each.
	c, d := w.topology.dependencies["[1].c"], w.topology.dependencies["[1].d"]
	require.Equal(t, group0, c)
	assert.Same(t, &c[0], &d[0])
}
