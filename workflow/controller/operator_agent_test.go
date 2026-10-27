package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

var httpwf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
  namespace: default
spec:
  entrypoint: http
  templates:
    - name: http
      http:
        url: https://www.google.com/
`

func TestHTTPTemplate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(httpwf)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()

	t.Run("ExecuteHTTPTemplate", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		pod, err := controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, pod)
		ts, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Len(t, ts.Spec.Tasks, 1)

		// simulate agent pod failure scenario: the pod is replaced, and its task kept
		pod.Status.Phase = v1.PodFailed
		pod.Status.Message = "manual termination"
		pod, err = controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, v1.PodFailed, pod.Status.Phase)
		// sleep 1 second to wait for informer getting pod info
		time.Sleep(time.Second)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		assert.Len(t, woc.wf.Status.Nodes, 1)
		assert.False(t, woc.wf.Status.Nodes["hello-world"].Fulfilled())
		assert.Contains(t, agentPodFailedEvents(controller), `Warning AgentPodFailed agent pod failed with reason:"manual termination"; recreating it`)
		ts, err = controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Len(t, ts.Spec.Tasks, 1)
	})
}

// agentPodFailedEvents drains the events the controller has recorded and
// returns the AgentPodFailed ones.
func agentPodFailedEvents(controller *WorkflowController) []string {
	recorder := controller.eventRecorderManager.(*testEventRecorderManager).eventRecorder
	var events []string
	for {
		select {
		case event := <-recorder.Events:
			if strings.Contains(event, "AgentPodFailed") {
				events = append(events, event)
			}
		default:
			return events
		}
	}
}

func TestHTTPTemplateWithoutServiceAccount(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(httpwf)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	t.Run("ExecuteHTTPTemplateWithoutServiceAccount", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		_, err := controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
		require.Error(t, err, `pods "%s" not found`, woc.getAgentPodName())
		ts, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Empty(t, ts.Spec.Tasks)
		assert.Empty(t, ts.Status.Nodes)
		assert.Len(t, woc.wf.Status.Nodes, 1)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		assert.Equal(t, wfv1.NodeError, woc.wf.Status.Nodes["hello-world"].Phase)
		assert.Equal(t, `create agent pod failed with reason:"failed to get token volumes: serviceaccounts "default" not found"`, woc.wf.Status.Nodes["hello-world"].Message)
	})
}

// failAgentPod makes the workflow's agent pod Failed, as an eviction does, and
// waits for the pod informer to see it.
func failAgentPod(ctx context.Context, t *testing.T, woc *wfOperationCtx, mutate func(pod *v1.Pod)) *v1.Pod {
	t.Helper()
	pods := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace)
	pod, err := pods.Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
	require.NoError(t, err)
	pod.Status.Phase = v1.PodFailed
	pod.Status.Reason = "Evicted"
	pod.Status.Message = "The node was low on resource: memory."
	if mutate != nil {
		mutate(pod)
	}
	pod, err = pods.Update(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, woc.controller.PodController.TestingPodInformer(), pod, func(obj any) bool {
		return obj.(*v1.Pod).Status.Phase == v1.PodFailed
	})
	return pod
}

// A Failed agent pod is replaced, as a crashed agent container is restarted,
// rather than erroring the agent's nodes and leaving every later HTTP or
// plugin node to be errored against the same Failed pod.
func TestAgentPodRecreatedWhenFailed(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(httpwf)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	failed := failAgentPod(ctx, t, woc, nil)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	require.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	require.False(t, woc.wf.Status.Nodes["hello-world"].Fulfilled(), "the agent's node must not be errored")

	// the controller deletes the Failed pod, by its UID
	require.True(t, controller.PodController.TestingProcessNextItem(ctx))
	pods := controller.kubeclientset.CoreV1().Pods(wf.Namespace)
	_, err := pods.Get(ctx, failed.Name, metav1.GetOptions{})
	require.True(t, apierr.IsNotFound(err), "the Failed agent pod must be deleted")
	waitForPodGone(ctx, t, woc, failed.Name)

	// and the next reconciliation creates its replacement
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	replacement, err := pods.Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
	require.NoError(t, err)
	assert.Empty(t, replacement.Status.Phase)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.False(t, woc.wf.Status.Nodes["hello-world"].Fulfilled())
}

func waitForPodGone(ctx context.Context, t *testing.T, woc *wfOperationCtx, name string) {
	t.Helper()
	err := kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		pod, err := woc.controller.PodController.GetPod(woc.wf.Namespace, name)
		return pod == nil, err
	})
	require.NoError(t, err)
}

// A later node, here a retry of the HTTP node, is not errored against the
// Failed agent pod either.
func TestAgentPodFailureDoesNotErrorRetries(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
  namespace: default
spec:
  entrypoint: http
  templates:
    - name: http
      retryStrategy:
        limit: 2
        retryPolicy: Always
      http:
        url: https://www.google.com/
`)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	failAgentPod(ctx, t, woc, nil)
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	for _, node := range woc.wf.Status.Nodes {
		assert.False(t, node.FailedOrError(), "node %s: %s", node.Name, node.Message)
	}
}

// newFailedAgentPodWoc runs the HTTP workflow until it has an agent pod, and
// returns a fresh operation context and that pod, Failed.
func newFailedAgentPodWoc(ctx context.Context, t *testing.T) (*wfOperationCtx, *v1.Pod) {
	t.Helper()
	wf := wfv1.MustUnmarshalWorkflow(httpwf)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	t.Cleanup(cancel)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pod, err := controller.kubeclientset.CoreV1().Pods(wf.Namespace).Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
	require.NoError(t, err)
	pod.Status.Phase = v1.PodFailed
	pod.Status.Message = "The node was low on resource: memory."
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	return woc, pod
}

// A Failed agent pod is replaced only once agentPodRecreationDelay has passed
// since it was created, and its nodes are never errored for it, however often
// it fails.
func TestAgentPodRecreationDelay(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc, pod := newFailedAgentPodWoc(ctx, t)

	pod.CreationTimestamp = metav1.NewTime(time.Now().Add(-agentPodRecreationDelay / 2))
	woc.updateAgentPodStatus(ctx, pod)
	assert.False(t, woc.wf.Status.Nodes["hello-world"].Fulfilled())
	assert.Empty(t, agentPodFailedEvents(woc.controller), "a pod that failed young must wait to be replaced")

	pod.CreationTimestamp = metav1.NewTime(time.Now().Add(-2 * agentPodRecreationDelay))
	for range 10 {
		woc.updateAgentPodStatus(ctx, pod)
	}
	assert.False(t, woc.wf.Status.Nodes["hello-world"].Fulfilled())
	assert.Len(t, agentPodFailedEvents(woc.controller), 10)
}

func TestAgentPodNotRecreatedWhenWorkflowEnding(t *testing.T) {
	for name, ending := range map[string]func(woc *wfOperationCtx){
		"Stop":      func(woc *wfOperationCtx) { woc.execWf.Spec.Shutdown = wfv1.ShutdownStrategyStop },
		"Terminate": func(woc *wfOperationCtx) { woc.execWf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate },
		"DeadlineExceeded": func(woc *wfOperationCtx) {
			deadline := time.Now().Add(-time.Second)
			woc.workflowDeadline = &deadline
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			woc, pod := newFailedAgentPodWoc(ctx, t)
			ending(woc)
			woc.updateAgentPodStatus(ctx, pod)
			node := woc.wf.Status.Nodes["hello-world"]
			assert.Equal(t, wfv1.NodeError, node.Phase)
			assert.Equal(t, `agent pod failed with reason:"The node was low on resource: memory."`, node.Message)
			assert.Empty(t, agentPodFailedEvents(woc.controller))
		})
	}
}
