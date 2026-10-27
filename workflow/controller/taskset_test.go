package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestCreateTaskSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/taskset/create-task-set-workflow.yaml")
	ctx := logging.TestContext(t.Context())
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal("@testdata/taskset/create-task-set-taskset.yaml", &ts)

	t.Run("CreateTaskSet", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)
		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Len(t, ts.Spec.Tasks, 1)
		}
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		assert.Len(t, pods.Items, 1)
		for _, pod := range pods.Items {
			assert.NotNil(t, pod)
			assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
		}
	})
	t.Run("CreateTaskSetWithInstanceID", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
		defer cancel()
		controller.Config.InstanceID = "testID"
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)
		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Len(t, ts.Spec.Tasks, 1)
			assert.Equal(t, "testID", ts.Labels[common.LabelKeyControllerInstanceID], "WorkflowTaskSet should have instanceID label")
		}
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		assert.Len(t, pods.Items, 1)
		for _, pod := range pods.Items {
			assert.NotNil(t, pod)
			assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
			assert.Equal(t, "testID", pod.Labels[common.LabelKeyControllerInstanceID])
		}
	})
}

func TestRemoveCompletedTaskSetStatus(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/taskset/remove-completed-task-set-status-workflow.yaml")
	ctx := logging.TestContext(t.Context())
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal("@testdata/taskset/remove-completed-task-set-status-taskset.yaml", &ts)
	t.Run("RemoveCompletedTaskSetStatus", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts)
		defer cancel()
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").Create(ctx, &ts, v1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err = woc.removeCompletedTaskSetStatus(ctx, woc.wf.Status.Nodes)
		require.NoError(t, err)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)

		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Empty(t, ts.Spec.Tasks)
			assert.Empty(t, ts.Status.Nodes)
		}
	})
}

func TestNonHTTPTemplateScenario(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	t.Run("reconcileTaskSet", func(t *testing.T) {
		woc.operate(ctx)
		err := woc.reconcileTaskSet(ctx)
		require.NoError(t, err)
	})
	t.Run("removeCompletedTaskSetStatus", func(t *testing.T) {
		woc.operate(ctx)
		err := woc.removeCompletedTaskSetStatus(ctx, woc.wf.Status.Nodes)
		require.NoError(t, err)
	})
}

func TestReconcileTaskSetWithMemoization(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/taskset/reconcile-task-set-with-memoization-workflow.yaml")
	ctx := logging.TestContext(t.Context())
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal("@testdata/taskset/reconcile-task-set-with-memoization-taskset.yaml", &ts)
	t.Run("MemoizeOnTaskSetSucceeded", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts)
		defer cancel()
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").Create(ctx, &ts, v1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		time.Sleep(1 * time.Second)
		err = woc.reconcileTaskSet(ctx)
		require.NoError(t, err)
		memo, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "cache-demo-1", v1.GetOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, memo.Data["cache-demo-1"])
	})
}

var taskSetTooLargeWf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: taskset-too-large
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: running
            template: http
          - name: new
            template: http
    - name: http
      http:
        url: https://www.google.com/
`

func TestTaskSetRequestEntityTooLarge(t *testing.T) {
	tooLarge := apierr.NewRequestEntityTooLargeError("limit is 3145728")

	t.Run("Create", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		wf := wfv1.MustUnmarshalWorkflow(httpwf)
		cancel, controller := newController(ctx, wf, defaultServiceAccount)
		defer cancel()
		creates := 0
		controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("create", "workflowtasksets", func(k8stesting.Action) (bool, runtime.Object, error) {
			creates++
			return true, nil, tooLarge
		})
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)

		node := woc.wf.Status.Nodes[wf.Name]
		assert.Equal(t, wfv1.NodeError, node.Phase)
		assert.Contains(t, node.Message, "size limit")
		assert.Contains(t, node.Message, "limit is 3145728")
		_, err := controller.kubeclientset.CoreV1().Pods(wf.Namespace).Get(ctx, woc.getAgentPodName(), v1.GetOptions{})
		assert.True(t, apierr.IsNotFound(err), "no agent pod is created for a taskset that was never written")

		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Equal(t, 1, creates, "the errored task is not written again")
	})

	t.Run("Patch", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		wf := wfv1.MustUnmarshalWorkflow(taskSetTooLargeWf)
		runningID := wf.NodeID(wf.Name + "[0].running")
		newID := wf.NodeID(wf.Name + "[0].new")
		ts := &wfv1.WorkflowTaskSet{
			ObjectMeta: v1.ObjectMeta{Name: wf.Name, Namespace: wf.Namespace},
			Spec: wfv1.WorkflowTaskSetSpec{
				Tasks: map[string]wfv1.Template{runningID: *wf.GetTemplateByName("http")},
			},
		}
		cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
		defer cancel()
		controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("patch", "workflowtasksets", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, tooLarge
		})
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)

		running := woc.wf.Status.Nodes[runningID]
		assert.Equal(t, wfv1.NodePending, running.Phase, "a task already in the taskset is left to the agent")
		added := woc.wf.Status.Nodes[newID]
		assert.Equal(t, wfv1.NodeError, added.Phase)
		assert.Contains(t, added.Message, "size limit")
	})
}
