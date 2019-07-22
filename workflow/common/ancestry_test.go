package common

import (
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type testContext struct {
	status map[string]*wfv1.NodeStatus
}

func (c *testContext) GetTaskNode(taskName string) *wfv1.NodeStatus {
	return c.status[taskName]
}

func TestGetTaskAncestryForValidation(t *testing.T) {
	type args struct {
		ctx      Context
		taskName string
		tasks    []wfv1.DAGTask
	}

	testTasks := []wfv1.DAGTask{
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

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "one task",
			args: args{
				ctx:      nil,
				taskName: "task2",
				tasks:    testTasks,
			},
			want: []string{"task1"},
		},
		{
			name: "multiple tasks",
			args: args{
				ctx:      nil,
				taskName: "task4",
				tasks:    testTasks,
			},
			want: []string{"task1", "task2", "task3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTaskAncestry(tt.args.ctx, tt.args.taskName, tt.args.tasks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTaskAncestry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTaskAncestryForGlobalArtifacts(t *testing.T) {
	type args struct {
		ctx      Context
		taskName string
		tasks    []wfv1.DAGTask
	}

	testTasks := []wfv1.DAGTask{
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
		status: map[string]*wfv1.NodeStatus{
			"task1": {
				FinishedAt: v1.Time{time.Now().Add(1 * time.Minute)},
			},
			"task2": {
				FinishedAt: v1.Time{time.Now().Add(3 * time.Minute)},
			},
			"task3": {
				FinishedAt: v1.Time{time.Now().Add(2 * time.Minute)},
			},
			"task4": {
				FinishedAt: v1.Time{time.Now().Add(4 * time.Minute)},
			},
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
				tasks:    testTasks,
			},
			want: []string{"task1"},
		},
		{
			name: "multiple tasks",
			args: args{
				ctx:      ctx,
				taskName: "task4",
				tasks:    testTasks,
			},
			want: []string{"task1", "task3", "task2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTaskAncestry(tt.args.ctx, tt.args.taskName, tt.args.tasks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTaskAncestry() = %v, want %v", got, tt.want)
			}
		})
	}
}
