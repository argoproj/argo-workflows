package dag

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
