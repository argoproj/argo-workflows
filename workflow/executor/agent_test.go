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
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestUnsupportedTemplateTaskWorker(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	responseQueue := make(chan response)
	defer close(responseQueue)
	ae := &AgentExecutor{
		consideredTasks: &sync.Map{},
		responseQueue:   responseQueue,
	}
	taskQueue := make(chan task)
	defer close(taskQueue)
	go ae.taskWorker(ctx, taskQueue)

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
			_, requeue, err := ae.processTask(ctx, "", *tc.template)
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

// TestAgentLogsTmpDir pins the contract archiveAgentLogs relies on: plugin-
// backed archive locations must produce a path under common.AgentPluginShareDir
// (the per-plugin slice of the shared emptyDir mounted on both the agent main
// container and that plugin's sidecar — see createAgentPod). Non-plugin
// locations return "" so os.CreateTemp falls back to os.TempDir, which is
// fine for the in-process drivers that run in the agent main container.
func TestAgentLogsTmpDir(t *testing.T) {
	t.Run("PluginLocationUsesShareDir", func(t *testing.T) {
		tmpl := &v1alpha1.Template{
			ArchiveLocation: &v1alpha1.ArtifactLocation{
				Plugin: &v1alpha1.PluginArtifact{Name: "test"},
			},
		}
		assert.Equal(t, common.AgentPluginShareDir+"/test", agentLogsTmpDir(tmpl))
	})
	t.Run("S3LocationFallsThroughToOSTempDir", func(t *testing.T) {
		tmpl := &v1alpha1.Template{
			ArchiveLocation: &v1alpha1.ArtifactLocation{
				S3: &v1alpha1.S3Artifact{},
			},
		}
		assert.Empty(t, agentLogsTmpDir(tmpl))
	})
	t.Run("NoArchiveLocationFallsThrough", func(t *testing.T) {
		assert.Empty(t, agentLogsTmpDir(&v1alpha1.Template{}))
	})
	t.Run("NilTemplateFallsThrough", func(t *testing.T) {
		assert.Empty(t, agentLogsTmpDir(nil))
	})
	t.Run("PluginWithEmptyNameFallsThrough", func(t *testing.T) {
		// An empty plugin name has no matching sidecar mount, so falling
		// through to /tmp is safer than producing common.AgentPluginShareDir
		// itself (the root view, shared with every plugin).
		tmpl := &v1alpha1.Template{
			ArchiveLocation: &v1alpha1.ArtifactLocation{
				Plugin: &v1alpha1.PluginArtifact{},
			},
		}
		assert.Empty(t, agentLogsTmpDir(tmpl))
	})
}
