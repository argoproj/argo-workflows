package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	executorplugins "github.com/argoproj/argo-workflows/v4/pkg/plugins/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// TestAgent runs a task set through the agent: each task runs once however
// many events list it, a plugin's requeue runs its task again, a task the
// agent can't run or that panics is errored, and all results reach the
// task set's status.
func TestAgent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	plugin := &namedPlugin{calls: map[string]int{}}
	pluginTemplate := func(name string) v1alpha1.Template {
		return v1alpha1.Template{Name: name, Plugin: &v1alpha1.Plugin{Object: v1alpha1.Object{Value: json.RawMessage(`{"my-plugin": {}}`)}}}
	}
	taskSet := &v1alpha1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Namespace: "my-ns"},
		Spec: v1alpha1.WorkflowTaskSetSpec{Tasks: map[string]v1alpha1.Template{
			"http":    {HTTP: &v1alpha1.HTTP{Method: http.MethodGet, URL: server.URL}},
			"plugin":  pluginTemplate("succeed"),
			"requeue": pluginTemplate("requeue"),
			"panic":   pluginTemplate("panic"),
			"dag":     {DAG: &v1alpha1.DAGTemplate{}},
		}},
	}
	wfClient := argofake.NewClientset(taskSet)
	watcher := watch.NewFake()
	wfClient.PrependWatchReactor("workflowtasksets", k8stesting.DefaultWatchReactor(watcher, nil))

	tr, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	process := &Process{
		Namespace:       "my-ns",
		Tracing:         tr,
		newRuntime:      func(TaskPaths) ContainerRuntimeExecutor { return nil },
		executorPlugins: []executorplugins.TemplateExecutor{plugin},
	}
	ae := NewAgentExecutor(process, wfClient, "my-wf", "my-uid", 3, 10*time.Millisecond)
	done := make(chan error)
	go func() { done <- ae.Agent(ctx) }()

	// The same tasks twice: each runs once.
	watcher.Add(taskSet.DeepCopy())
	watcher.Modify(taskSet.DeepCopy())

	taskSets := wfClient.ArgoprojV1alpha1().WorkflowTaskSets("my-ns")
	var nodes map[string]v1alpha1.NodeResult
	require.Eventually(t, func() bool {
		got, err := taskSets.Get(ctx, "my-wf", metav1.GetOptions{})
		require.NoError(t, err)
		nodes = got.Status.Nodes
		return len(nodes) == 5 && nodes["requeue"].Phase == v1alpha1.NodeSucceeded
	}, 10*time.Second, 10*time.Millisecond)

	assert.Equal(t, v1alpha1.NodeSucceeded, nodes["http"].Phase)
	require.NotNil(t, nodes["http"].Outputs)
	assert.Equal(t, "hello", *nodes["http"].Outputs.Result)
	assert.Equal(t, v1alpha1.NodeSucceeded, nodes["plugin"].Phase)
	assert.Equal(t, v1alpha1.NodeError, nodes["panic"].Phase)
	assert.Equal(t, "error processing task: kaboom", nodes["panic"].Message)
	assert.Equal(t, v1alpha1.NodeError, nodes["dag"].Phase)
	assert.Contains(t, nodes["dag"].Message, "error processing task: agent cannot execute: unknown task type")
	assert.Equal(t, map[string]int{"succeed": 1, "requeue": 2, "panic": 1}, plugin.callCounts())

	completed := taskSet.DeepCopy()
	completed.Labels = map[string]string{common.LabelKeyCompleted: "true"}
	watcher.Modify(completed)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("the agent did not stop when the workflow completed")
	}
}

// namedPlugin executes plugin templates by name: "succeed" succeeds,
// "requeue" asks to be run again once before succeeding, and "panic"
// panics.
type namedPlugin struct {
	mu    sync.Mutex
	calls map[string]int
}

func (p *namedPlugin) callCounts() map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return maps.Clone(p.calls)
}

func (p *namedPlugin) ExecuteTemplate(_ context.Context, args executorplugins.ExecuteTemplateArgs, reply *executorplugins.ExecuteTemplateReply) error {
	name := args.Template.Name
	p.mu.Lock()
	p.calls[name]++
	calls := p.calls[name]
	p.mu.Unlock()
	switch {
	case name == "panic":
		panic("kaboom")
	case name == "requeue" && calls == 1:
		reply.Node = &v1alpha1.NodeResult{Phase: v1alpha1.NodeRunning}
		reply.Requeue = &metav1.Duration{Duration: 10 * time.Millisecond}
	case name == "succeed", name == "requeue":
		reply.Node = &v1alpha1.NodeResult{Phase: v1alpha1.NodeSucceeded}
	default:
		return fmt.Errorf("unexpected template %q", name)
	}
	return nil
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
			we := &WorkflowExecutor{
				Process: &Process{executorPlugins: []executorplugins.TemplateExecutor{tc.plugin}},
			}
			_, requeue, err := we.processTask(ctx, *tc.template)
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
