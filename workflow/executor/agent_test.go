package executor

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
)

func TestUnsupportedTemplateTaskWorker(t *testing.T) {
	ae := &AgentExecutor{
		consideredTasks: &sync.Map{},
	}
	taskQueue := make(chan task)
	defer close(taskQueue)
	responseQueue := make(chan response)
	defer close(responseQueue)
	go ae.taskWorker(context.Background(), taskQueue, responseQueue)

	taskQueue <- task{
		NodeId: "a",
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
		pluginName    string
		plugin        executorplugins.TemplateExecutor
		expectRequeue time.Duration
		expectResult  *v1alpha1.NodeResult
	}{
		{
			name: "hello plugin execute succeeded with requeue duration 0",
			template: &v1alpha1.Template{
				Plugin: &v1alpha1.Plugin{
					Object: v1alpha1.Object{Value: json.RawMessage(`{"hello": "hello world"}`)},
				},
			},
			pluginName:    "hello",
			plugin:        &alwaysSucceededPlugin{requeue: time.Duration(0)},
			expectRequeue: time.Duration(0),
			expectResult: &v1alpha1.NodeResult{
				Phase: v1alpha1.NodeSucceeded,
			},
		},
		{
			name: "hello plugin execute succeeded with requeue duration 1h",
			template: &v1alpha1.Template{
				Plugin: &v1alpha1.Plugin{
					Object: v1alpha1.Object{Value: json.RawMessage(`{"hello": "hello world"}`)},
				},
			},
			pluginName:    "hello",
			plugin:        &alwaysSucceededPlugin{requeue: time.Hour},
			expectRequeue: time.Duration(0),
			expectResult: &v1alpha1.NodeResult{
				Phase: v1alpha1.NodeSucceeded,
			},
		},
		{
			name: "nonexistent plugin execute succeeded with requeue duration 0",
			template: &v1alpha1.Template{
				Plugin: &v1alpha1.Plugin{
					Object: v1alpha1.Object{Value: json.RawMessage(`{"nonexistent": "hello world"}`)},
				},
			},
			pluginName:    "hello",
			plugin:        &alwaysSucceededPlugin{requeue: time.Duration(0)},
			expectRequeue: time.Duration(0),
			expectResult: &v1alpha1.NodeResult{
				Phase: v1alpha1.NodeSucceeded,
			},
		},
		{
			name: "nonexistent plugin execute failed with requeue duration 0",
			template: &v1alpha1.Template{
				Plugin: &v1alpha1.Plugin{
					Object: v1alpha1.Object{Value: json.RawMessage(`{"nonexistent": "hello world"}`)},
				},
			},
			pluginName:    "hello",
			plugin:        &dummyPlugin{},
			expectRequeue: time.Duration(0),
			expectResult: &v1alpha1.NodeResult{
				Phase:   v1alpha1.NodeFailed,
				Message: "no plugin executed the template",
			},
		},
		{
			name: "dummy plugin execute failed with requeue duration 0",
			template: &v1alpha1.Template{
				Plugin: &v1alpha1.Plugin{
					Object: v1alpha1.Object{Value: json.RawMessage(`{"dummy": "hello world"}`)},
				},
			},
			pluginName:    "dummy",
			plugin:        &dummyPlugin{},
			expectRequeue: time.Duration(0),
			expectResult: &v1alpha1.NodeResult{
				Phase:   v1alpha1.NodeFailed,
				Message: "plugin:'dummy' could not execute the template",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ae := &AgentExecutor{
				consideredTasks: &sync.Map{},
				plugins:         map[string]executorplugins.TemplateExecutor{tc.pluginName: tc.plugin},
			}
			result, requeue, err := ae.processTask(context.Background(), *tc.template)
			require.NoError(t, err)
			assert.Equal(t, tc.expectResult, result)
			assert.Equal(t, tc.expectRequeue, requeue)
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

type dummyPlugin struct {
}

func (d dummyPlugin) ExecuteTemplate(_ context.Context, _ executorplugins.ExecuteTemplateArgs, _ *executorplugins.ExecuteTemplateReply) error {
	return nil
}
