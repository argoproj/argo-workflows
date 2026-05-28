package executor

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// installKubectlStub swaps the package-level runKubectl with a stub that
// reads the manifest off disk (via the -f flag the agent passes) and
// applies the implied action against the fake dynamic client. It restores
// the original on test cleanup. The stub supports the `create` and
// `apply` verbs the agent uses today; everything else fails the test.
func installKubectlStub(t *testing.T, client *fakedynamic.FakeDynamicClient, defaultNamespace string) {
	t.Helper()
	orig := runKubectl
	t.Cleanup(func() { runKubectl = orig })

	runKubectl = func(ctx context.Context, args ...string) ([]byte, error) {
		verb := ""
		manifestPath := ""
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case verb == "" && !strings.HasPrefix(a, "-"):
				verb = a
			case a == "-f" && i+1 < len(args):
				manifestPath = args[i+1]
			}
		}
		require.NotEmptyf(t, manifestPath, "kubectl stub: expected -f flag in args %v", args)

		body, err := os.ReadFile(manifestPath)
		require.NoError(t, err, "kubectl stub: read manifest")

		obj := &unstructured.Unstructured{}
		require.NoError(t, yaml.Unmarshal(body, &obj.Object))
		if obj.GetNamespace() == "" {
			obj.SetNamespace(defaultNamespace)
		}

		gvr := inferGVR(*obj)
		ns := client.Resource(gvr).Namespace(obj.GetNamespace())

		var created *unstructured.Unstructured
		switch verb {
		case "create", "apply":
			created, err = ns.Create(ctx, obj, metav1.CreateOptions{})
			if err != nil {
				return nil, err
			}
		default:
			t.Fatalf("kubectl stub: unsupported verb %q", verb)
		}

		out, err := json.Marshal(created)
		require.NoError(t, err)
		return out, nil
	}
}

// TestProcessTask_ResourceTemplate_EndToEnd drives the full state machine
// for a non-delete resource template: processTask invokes the kubectl stub
// (which writes a Pod into the fake dynamic client), the informer fires
// handleDone, the successCondition holds against the pod's status.phase,
// and a NodeSucceeded response lands on the responseQueue.
func TestProcessTask_ResourceTemplate_EndToEnd(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	const (
		workflowName = "wf-1"
		namespace    = "ns-1"
		nodeID       = "node-1"
		podName      = "p1"
	)

	client := fakedynamic.NewSimpleDynamicClient(newInformerScheme(t))
	installKubectlStub(t, client, namespace)

	ae := &AgentExecutor{
		WorkflowName:    workflowName,
		Namespace:       namespace,
		DynamicClient:   client,
		consideredTasks: &sync.Map{},
		pendingTasks:    map[string]pendingResourceTask{},
		// Buffered: handleDone posts from an informer goroutine and there's
		// no patchWorker in this test, so an unbuffered chan would deadlock.
		responseQueue: make(chan response, 4),
	}
	ae.resourceInformer = NewMonitoredResourceInformer(client, namespace, workflowName, 0, ae.handleDone)

	manifest := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: " + podName + "\n  namespace: " + namespace + "\nspec:\n  containers:\n  - name: main\n    image: busybox\nstatus:\n  phase: Running\n"

	tmpl := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action:           "create",
			Manifest:         manifest,
			SuccessCondition: "status.phase=Succeeded",
			FailureCondition: "status.phase=Failed",
		},
	}

	result, requeue, err := ae.processTask(ctx, nodeID, tmpl)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), requeue)
	require.NotNil(t, result)
	assert.Equal(t, wfv1.NodeRunning, result.Phase, "initial dispatch should mark the node Running while the informer waits")

	// Confirm the kubectl stub created the labeled pod.
	pod, err := client.Resource(podGVR).Namespace(namespace).Get(ctx, podName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, workflowName, pod.GetLabels()[common.LabelKeyMonitoredResource])
	assert.Equal(t, nodeID, pod.GetLabels()[common.LabelKeyMonitoredResourceNodeID])

	// Drive the resource to its successCondition.
	require.NoError(t, unstructured.SetNestedField(pod.Object, "Succeeded", "status", "phase"))
	_, err = client.Resource(podGVR).Namespace(namespace).Update(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)

	select {
	case resp := <-ae.responseQueue:
		assert.Equal(t, nodeID, resp.NodeID)
		require.NotNil(t, resp.Result)
		assert.Equal(t, wfv1.NodeSucceeded, resp.Result.Phase)
	case <-time.After(3 * time.Second):
		t.Fatalf("expected NodeSucceeded on responseQueue within 3s")
	}

	// Pending entry should be cleared once the node is complete.
	ae.pendingMu.Lock()
	_, stillPending := ae.pendingTasks[nodeID]
	ae.pendingMu.Unlock()
	assert.False(t, stillPending, "pending entry should be cleared on completion")
}

// TestProcessTask_ResourceTemplate_FailureCondition exercises the failure
// branch: the same setup, but the pod transitions to status.phase=Failed.
// The agent should post NodeFailed.
func TestProcessTask_ResourceTemplate_FailureCondition(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	const (
		workflowName = "wf-1"
		namespace    = "ns-1"
		nodeID       = "node-1"
		podName      = "p1"
	)

	client := fakedynamic.NewSimpleDynamicClient(newInformerScheme(t))
	installKubectlStub(t, client, namespace)

	ae := &AgentExecutor{
		WorkflowName:    workflowName,
		Namespace:       namespace,
		DynamicClient:   client,
		consideredTasks: &sync.Map{},
		pendingTasks:    map[string]pendingResourceTask{},
		responseQueue:   make(chan response, 4),
	}
	ae.resourceInformer = NewMonitoredResourceInformer(client, namespace, workflowName, 0, ae.handleDone)

	manifest := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: " + podName + "\n  namespace: " + namespace + "\nspec:\n  containers:\n  - name: main\n    image: busybox\nstatus:\n  phase: Running\n"

	tmpl := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action:           "create",
			Manifest:         manifest,
			SuccessCondition: "status.phase=Succeeded",
			FailureCondition: "status.phase=Failed",
		},
	}

	result, _, err := ae.processTask(ctx, nodeID, tmpl)
	require.NoError(t, err)
	require.Equal(t, wfv1.NodeRunning, result.Phase)

	pod, err := client.Resource(podGVR).Namespace(namespace).Get(ctx, podName, metav1.GetOptions{})
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(pod.Object, "Failed", "status", "phase"))
	_, err = client.Resource(podGVR).Namespace(namespace).Update(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)

	select {
	case resp := <-ae.responseQueue:
		assert.Equal(t, nodeID, resp.NodeID)
		require.NotNil(t, resp.Result)
		assert.Equal(t, wfv1.NodeFailed, resp.Result.Phase)
		assert.Contains(t, resp.Result.Message, "failure condition")
	case <-time.After(3 * time.Second):
		t.Fatalf("expected NodeFailed on responseQueue within 3s")
	}
}

// TestProcessTask_ResourceTemplate_Delete is the fire-and-forget path: the
// agent issues the delete via the stub and immediately returns Succeeded
// without registering a pending task or starting an informer.
func TestProcessTask_ResourceTemplate_Delete(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	const (
		workflowName = "wf-1"
		namespace    = "ns-1"
		nodeID       = "node-1"
	)

	client := fakedynamic.NewSimpleDynamicClient(newInformerScheme(t))

	// For delete we just need runKubectl to no-op successfully; the agent
	// short-circuits before parsing kubectl's output.
	orig := runKubectl
	t.Cleanup(func() { runKubectl = orig })
	runKubectl = func(ctx context.Context, args ...string) ([]byte, error) { return nil, nil }

	ae := &AgentExecutor{
		WorkflowName:    workflowName,
		Namespace:       namespace,
		DynamicClient:   client,
		consideredTasks: &sync.Map{},
		pendingTasks:    map[string]pendingResourceTask{},
		responseQueue:   make(chan response, 4),
	}
	ae.resourceInformer = NewMonitoredResourceInformer(client, namespace, workflowName, 0, ae.handleDone)

	manifest := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: p1\n  namespace: ns-1\n"
	tmpl := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action:   "delete",
			Manifest: manifest,
		},
	}

	result, _, err := ae.processTask(ctx, nodeID, tmpl)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, result.Phase)

	ae.pendingMu.Lock()
	_, registered := ae.pendingTasks[nodeID]
	ae.pendingMu.Unlock()
	assert.False(t, registered, "delete should not register a pending task")
}
