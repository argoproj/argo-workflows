package executor

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
)

func TestUnsupportedTemplateTaskWorker(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	ae := &AgentExecutor{
		consideredTasks: &sync.Map{},
	}
	taskQueue := make(chan task)
	defer close(taskQueue)
	responseQueue := make(chan response)
	defer close(responseQueue)
	go ae.taskWorker(ctx, taskQueue, responseQueue)

	taskQueue <- task{
		NodeID: "a",
		// This template type is not supported
		Template: v1alpha1.Template{
			DAG: &v1alpha1.DAGTemplate{},
		},
	}

	response := <-responseQueue
	assert.Equal(t, v1alpha1.NodeError, response.Result.Phase)
	assert.Contains(t, response.Result.Message, "agent cannot execute: unknown task type")
}

func TestAgentPluginExecuteTaskSet(t *testing.T) {
	tests := []struct {
		name          string
		template      *v1alpha1.Template
		plugin        executorplugins.TemplateExecutor
		expectRequeue time.Duration
	}{
		{
			name: "never requeue after plugin execute succeeded (requeue duration 0)",
			template: &v1alpha1.Template{
				Plugin: &v1alpha1.Plugin{
					Object: v1alpha1.Object{Value: json.RawMessage(`{"key": "value"}`)},
				},
			},
			plugin:        &alwaysSucceededPlugin{requeue: time.Duration(0)},
			expectRequeue: time.Duration(0),
		},
		{
			name: "never requeue after plugin execute succeeded (requeue duration 1h)",
			template: &v1alpha1.Template{
				Plugin: &v1alpha1.Plugin{
					Object: v1alpha1.Object{Value: json.RawMessage(`{"key": "value"}`)},
				},
			},
			plugin:        &alwaysSucceededPlugin{requeue: time.Hour},
			expectRequeue: time.Duration(0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			ae := &AgentExecutor{
				consideredTasks: &sync.Map{},
				plugins:         []executorplugins.TemplateExecutor{tc.plugin},
			}
			_, requeue, err := ae.processTask(ctx, *tc.template)
			if err != nil {
				t.Errorf("expect nil, but got %v", err)
			}
			if requeue != tc.expectRequeue {
				t.Errorf("expect requeue after %s, but got %v", tc.expectRequeue, requeue)
			}
		})
	}
}

func TestBatchNodeResults(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]v1alpha1.NodeResult
		batchSize int

		expectBatchCount int
		expectBatchSizes []int
	}{
		{
			name:      "empty collection",
			input:     map[string]v1alpha1.NodeResult{},
			batchSize: 3,

			expectBatchCount: 0,
			expectBatchSizes: []int{},
		},
		{
			name: "less than batch size",
			input: map[string]v1alpha1.NodeResult{
				"a": {}, "b": {},
			},
			batchSize: 3,

			expectBatchCount: 1,
			expectBatchSizes: []int{2},
		},
		{
			name: "equal to batch size",
			input: map[string]v1alpha1.NodeResult{
				"a": {}, "b": {}, "c": {},
			},
			batchSize: 3,

			expectBatchCount: 1,
			expectBatchSizes: []int{3},
		},
		{
			name: "greater than batch size",
			input: map[string]v1alpha1.NodeResult{
				"a": {}, "b": {}, "c": {}, "d": {}, "e": {},
			},
			batchSize: 3,

			expectBatchCount: 2,
			expectBatchSizes: []int{3, 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			batches := batchNodeResults(tc.input, tc.batchSize)

			if len(batches) != tc.expectBatchCount {
				t.Fatalf("expected %d batches, got %d", tc.expectBatchCount, len(batches))
			}

			for i, b := range batches {
				if len(b) != tc.expectBatchSizes[i] {
					t.Errorf("batch %d: expected size %d, got %d", i, tc.expectBatchSizes[i], len(b))
				}

				// optional: ensure no data loss
				for k, v := range b {
					if _, ok := tc.input[k]; !ok {
						t.Errorf("batch %d contains unknown key %s", i, k)
					}
					_ = v
				}
			}
		})
	}
}

type alwaysSucceededPlugin struct {
	requeue time.Duration
}

func (a alwaysSucceededPlugin) ExecuteTemplate(_ context.Context, _ executorplugins.ExecuteTemplateArgs, reply *executorplugins.ExecuteTemplateReply) error {
	reply.Node = &v1alpha1.NodeResult{
		Phase: v1alpha1.NodeSucceeded,
	}
	reply.Requeue = &metav1.Duration{Duration: a.requeue}
	return nil
}
