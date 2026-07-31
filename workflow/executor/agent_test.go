package executor

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wffake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	executorplugins "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
	argoerr "github.com/argoproj/argo-workflows/v4/util/errors"
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

func TestAgentSkipsResourceTasks(t *testing.T) {
	// Resource templates share the taskset but are served by the resource agent. The plain agent must
	// ignore them (empty phase => nothing patched), not error them. processTask skips them by template
	// type — not resource.mode and without touching consideredTasks — so it holds for every mode.
	ctx := logging.TestContext(t.Context())
	ae := &AgentExecutor{}
	for _, mode := range []v1alpha1.ResourceTemplateMode{v1alpha1.ResourceTemplateModeAgent, v1alpha1.ResourceTemplateModePod, ""} {
		result, requeue, err := ae.processTask(ctx, v1alpha1.Template{
			Resource: &v1alpha1.ResourceTemplate{Action: "create", Mode: mode},
		})
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), requeue)
		assert.Empty(t, string(result.Phase))
	}
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

// TestPatchTaskSetStatusNodesTimeoutIsTransient asserts the bound on the patch retry loop
// surfaces as a transient error: after a slow or hung API server exhausts the window, both
// patch workers must keep their pending results and retry next tick — never mark every
// pending node errored, which is reserved for genuine payload/permission failures.
func TestPatchTaskSetStatusNodesTimeoutIsTransient(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	permanentErr := errors.New("some permanent, non-transient failure")
	clientset := wffake.NewClientset()
	clientset.PrependReactor("patch", "workflowtasksets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, permanentErr
	})
	tsIface := clientset.ArgoprojV1alpha1().WorkflowTaskSets("default")
	nodes := map[string]v1alpha1.NodeResult{"n1": {Phase: v1alpha1.NodeSucceeded}}

	// With a live context, a permanent failure must classify non-transient so the
	// workers escalate it to the nodes.
	err := patchTaskSetStatusNodes(ctx, tsIface, "wf", nodes)
	require.Error(t, err)
	assert.False(t, argoerr.IsTransientErr(ctx, err))

	// With the patch window already exhausted, the same failure must classify transient:
	// an expired deadline says nothing about the payload.
	expired, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
	defer cancel()
	err = patchTaskSetStatusNodes(expired, tsIface, "wf", nodes)
	require.Error(t, err)
	assert.True(t, argoerr.IsTransientErr(ctx, err))
}
