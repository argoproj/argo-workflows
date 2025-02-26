package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	cancel, controller := newController(wf, defaultServiceAccount)
	defer cancel()

	t.Run("ExecuteHTTPTemplate", func(t *testing.T) {
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		pod, err := controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, pod)
		ts, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Len(t, ts.Spec.Tasks, 1)

		// simulate agent pod failure scenario
		pod.Status.Phase = v1.PodFailed
		pod.Status.Message = "manual termination"
		pod, err = controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, v1.PodFailed, pod.Status.Phase)
		// sleep 1 second to wait for informer getting pod info
		time.Sleep(time.Second)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Equal(t, `agent pod failed with reason:"manual termination"`, woc.wf.Status.Message)
		assert.Len(t, woc.wf.Status.Nodes, 1)
		assert.Equal(t, wfv1.NodeError, woc.wf.Status.Nodes["hello-world"].Phase)
		assert.Equal(t, `agent pod failed with reason:"manual termination"`, woc.wf.Status.Nodes["hello-world"].Message)
		ts, err = controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Empty(t, ts.Spec.Tasks)
		assert.Empty(t, ts.Status.Nodes)
	})
}

func TestHTTPTemplateWithoutServiceAccount(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(httpwf)
	cancel, controller := newController(wf)
	defer cancel()

	t.Run("ExecuteHTTPTemplateWithoutServiceAccount", func(t *testing.T) {
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
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

func TestHTTPTemplateWhenAgentPodFailed(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(httpwf)
	cancel, controller := newController(wf, defaultServiceAccount)
	defer cancel()

	t.Run("ExecuteHTTPTemplateWhenAgentPodFailed", func(t *testing.T) {
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		pod, err := controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, pod)
		ts, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Len(t, ts.Spec.Tasks, 1)
		ts.Status.Nodes = make(map[string]wfv1.NodeResult)
		ts.Status.Nodes["hello-world"] = wfv1.NodeResult{
			Phase:   wfv1.NodePending,
			Message: "Queuing",
		}
		ts, err = controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).UpdateStatus(ctx, ts, metav1.UpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodePending, ts.Status.Nodes["hello-world"].Phase)
		wf, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)

		// simulate agent pod failure scenario
		pod.Status.Phase = v1.PodFailed
		pod.Status.Message = "manual termination"
		pod, err = controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		assert.Equal(t, v1.PodFailed, pod.Status.Phase)
		// sleep 1 second to wait for informer getting pod info
		time.Sleep(time.Second)
		woc = newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Equal(t, `agent pod failed with reason:"manual termination"`, woc.wf.Status.Message)
		assert.Len(t, woc.wf.Status.Nodes, 1)
		assert.Equal(t, wfv1.NodeError, woc.wf.Status.Nodes["hello-world"].Phase)
		assert.Equal(t, `agent pod failed with reason:"manual termination"`, woc.wf.Status.Nodes["hello-world"].Message)
		ts, err = controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, "hello-world", metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Empty(t, ts.Spec.Tasks)
		assert.Empty(t, ts.Status.Nodes)
	})
}
