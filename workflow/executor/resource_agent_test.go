package executor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// TestResourceAgent runs a task set through the resource agent: the
// resource task creates its object and claims it, the informer sees the
// claimed object meet its success condition, and the result with its
// output parameters reaches the task set's status. The HTTP task in the
// same set is not the resource agent's.
func TestResourceAgent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "foos"}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{gvr: "FooList"})
	created := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "example.com/v1",
		"kind":       "Foo",
		"metadata":   map[string]any{"name": "my-foo", "namespace": "my-ns"},
		"status":     map[string]any{"phase": "Done"},
	}}
	kubectlRunnerFn = func(ctx context.Context, args ...string) ([]byte, error) {
		require.Equal(t, "create", args[1])
		obj, err := dynamicClient.Resource(gvr).Namespace("my-ns").Create(ctx, created, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		return json.Marshal(obj)
	}
	t.Cleanup(func() { kubectlRunnerFn = runKubectl })

	taskSet := &wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Namespace: "my-ns"},
		Spec: wfv1.WorkflowTaskSetSpec{Tasks: map[string]wfv1.Template{
			"res": {
				Resource: &wfv1.ResourceTemplate{
					Action:           "create",
					Manifest:         "apiVersion: example.com/v1\nkind: Foo\nmetadata:\n  name: my-foo\n",
					SuccessCondition: "status.phase == Done",
				},
				Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "phase", ValueFrom: &wfv1.ValueFrom{JSONPath: "{.status.phase}"}}}},
			},
			"http": {HTTP: &wfv1.HTTP{URL: "http://example.com"}},
		}},
	}
	wfClient := argofake.NewClientset(taskSet)
	watcher := watch.NewFake()
	wfClient.PrependWatchReactor("workflowtasksets", k8stesting.DefaultWatchReactor(watcher, nil))

	tr, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	process := &Process{
		Namespace:  "my-ns",
		Tracing:    tr,
		newRuntime: func(TaskPaths) ContainerRuntimeExecutor { return nil },
	}
	re := NewResourceAgentExecutor(process, wfClient, dynamicClient, "my-wf", "my-uid", 2, 10*time.Millisecond)
	done := make(chan error, 1)
	go func() { done <- re.Run(ctx) }()

	// The same tasks twice: each runs once.
	watcher.Add(taskSet.DeepCopy())
	watcher.Modify(taskSet.DeepCopy())

	taskSets := wfClient.ArgoprojV1alpha1().WorkflowTaskSets("my-ns")
	var nodes map[string]wfv1.NodeResult
	require.Eventually(t, func() bool {
		got, err := taskSets.Get(ctx, "my-wf", metav1.GetOptions{})
		require.NoError(t, err)
		nodes = got.Status.Nodes
		return nodes["res"].Phase == wfv1.NodeSucceeded
	}, 10*time.Second, 10*time.Millisecond)
	require.NotNil(t, nodes["res"].Outputs)
	assert.Equal(t, "Done", nodes["res"].Outputs.Parameters[0].Value.String())
	assert.NotContains(t, nodes, "http")

	obj, err := dynamicClient.Resource(gvr).Namespace("my-ns").Get(ctx, "my-foo", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "res", obj.GetAnnotations()[common.AnnotationKeyNodeID])
	assert.Equal(t, "my-uid", obj.GetLabels()[common.LabelKeyAgentResource])

	completed := taskSet.DeepCopy()
	completed.Labels = map[string]string{common.LabelKeyCompleted: "true"}
	watcher.Modify(completed)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("the resource agent did not stop when the workflow completed")
	}
}
