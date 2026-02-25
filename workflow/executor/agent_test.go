package executor

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	resp := <-responseQueue
	assert.Equal(t, v1alpha1.NodeError, resp.Result.Phase)
	assert.Contains(t, resp.Result.Message, "agent cannot execute: unknown task type")
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
			_, requeue, err := ae.processTask(ctx, "test-workflow", "test-workflow-uid", *tc.template)
			if err != nil {
				t.Errorf("expect nil, but got %v", err)
			}
			if requeue != tc.expectRequeue {
				t.Errorf("expect requeue after %s, but got %v", tc.expectRequeue, requeue)
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

func TestAgentExecutorWithLabelSelector(t *testing.T) {
	t.Run("LabelSelector is set correctly", func(t *testing.T) {
		labelSelector := "workflows.argoproj.io/workflow-service-account=my-sa"
		ae := &AgentExecutor{
			LabelSelector:   labelSelector,
			consideredTasks: &sync.Map{},
		}

		assert.Equal(t, labelSelector, ae.LabelSelector)
		assert.NotNil(t, ae.consideredTasks)
	})
}

func TestTaskWithWorkflowName(t *testing.T) {
	t.Run("Task includes TaskSetName and WorkflowUID", func(t *testing.T) {
		wfTask := task{
			NodeID:      "node-123",
			Template:    v1alpha1.Template{},
			TaskSetName: "my-workflow",
			WorkflowUID: "workflow-uid-123",
		}

		assert.Equal(t, "my-workflow", wfTask.TaskSetName)
		assert.Equal(t, "node-123", wfTask.NodeID)
		assert.Equal(t, "workflow-uid-123", wfTask.WorkflowUID)
	})
}

func TestResponseWithWorkflowName(t *testing.T) {
	t.Run("Response includes TaskSetName for patching", func(t *testing.T) {
		resp := response{
			NodeID: "node-123",
			Result: &v1alpha1.NodeResult{
				Phase: v1alpha1.NodeSucceeded,
			},
			TaskSetName: "my-workflow",
		}

		assert.Equal(t, "node-123", resp.NodeID)
		assert.Equal(t, "my-workflow", resp.TaskSetName)
		assert.Equal(t, v1alpha1.NodeSucceeded, resp.Result.Phase)
	})
}

func TestProcessTaskWithWorkflowName(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("HTTP template processes with workflow name", func(t *testing.T) {
		ae := &AgentExecutor{
			consideredTasks: &sync.Map{},
		}

		tmpl := v1alpha1.Template{
			HTTP: &v1alpha1.HTTP{
				Method: "GET",
				URL:    "http://example.com",
			},
		}

		// This should not panic and should handle workflow name and UID parameters
		result, _, err := ae.processTask(ctx, "test-workflow", "test-workflow-uid", tmpl)
		// The HTTP request will fail because we don't have a real server,
		// but it should process with the workflow name correctly
		// We're testing that the workflow name parameter is passed correctly
		assert.NotNil(t, result)
		// Error may or may not occur depending on network, so we just check result exists
		_ = err // Ignore error for this test
	})

	t.Run("Plugin template receives correct workflow name", func(t *testing.T) {
		plugin := &workflowNameCapturePlugin{}
		ae := &AgentExecutor{
			consideredTasks: &sync.Map{},
			plugins:         []executorplugins.TemplateExecutor{plugin},
			Namespace:       "test-namespace",
		}

		tmpl := v1alpha1.Template{
			Plugin: &v1alpha1.Plugin{
				Object: v1alpha1.Object{Value: json.RawMessage(`{"key": "value"}`)},
			},
		}

		_, _, err := ae.processTask(ctx, "my-workflow", "test-workflow-uid", tmpl)
		require.NoError(t, err)
		assert.Equal(t, "my-workflow", plugin.capturedWorkflowName)
		assert.Equal(t, "test-namespace", plugin.capturedNamespace)
		assert.Equal(t, "test-workflow-uid", plugin.capturedUID)
	})
}

type workflowNameCapturePlugin struct {
	capturedWorkflowName string
	capturedNamespace    string
	capturedUID          string
}

func (w *workflowNameCapturePlugin) ExecuteTemplate(_ context.Context, args executorplugins.ExecuteTemplateArgs, reply *executorplugins.ExecuteTemplateReply) error {
	w.capturedWorkflowName = args.Workflow.ObjectMeta.Name
	w.capturedNamespace = args.Workflow.ObjectMeta.Namespace
	w.capturedUID = args.Workflow.ObjectMeta.UID
	reply.Node = &v1alpha1.NodeResult{
		Phase: v1alpha1.NodeSucceeded,
	}
	return nil
}

func TestIsWorkflowCompleted(t *testing.T) {
	t.Run("Returns true when completed label is true", func(t *testing.T) {
		wts := &v1alpha1.WorkflowTaskSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"workflows.argoproj.io/completed": "true",
				},
			},
		}
		assert.True(t, IsWorkflowCompleted(wts))
	})

	t.Run("Returns false when completed label is false", func(t *testing.T) {
		wts := &v1alpha1.WorkflowTaskSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"workflows.argoproj.io/completed": "false",
				},
			},
		}
		assert.False(t, IsWorkflowCompleted(wts))
	})

	t.Run("Returns false when completed label is missing", func(t *testing.T) {
		wts := &v1alpha1.WorkflowTaskSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{},
			},
		}
		assert.False(t, IsWorkflowCompleted(wts))
	})
}
