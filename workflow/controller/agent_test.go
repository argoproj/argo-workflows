package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestExecuteTaskSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template
  namespace: default
spec:
  podSpecPatch: |
    nodeName: virtual-node
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: good
            template: http
            arguments:
              parameters: [{name: url, value: "https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json"}]
        - - name: bad
            template: http
            continueOn:
              failed: true
            arguments:
              parameters: [{name: url, value: "http://openlibrary.org/people/george08/nofound.json"}]

    - name: http
      inputs:
        parameters:
          - name: url
      http:
       url: "{{inputs.parameters.url}}"

`)
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowTaskSet
metadata:
  name: http-template-1
  namespace: default
spec:
  tasks:
    http-template-nxvtg-1265710817:
      http:
        url: http://openlibrary.org/people/george08/nofound.json
      inputs:
        parameters:
        - name: url
          value: http://openlibrary.org/people/george08/nofound.json
      name: http
status:
  nodes:
    http-template-1-3690327077:
      outputs:
        parameters:
        - name: result
          value: |
            {
              "swagger": "2.0",
              "info": {
                "title": "pkg/apis/workflow/v1alpha1/generated.proto",
                "version": "version not set"
              },
              "consumes": [
                "application/json"
              ],
              "produces": [
                "application/json"
              ],
              "paths": {},
              "definitions": {}
            }
      phase: Succeeded
    `, &ts)

	t.Run("CreateTaskSet", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
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
			assert.Equal(t, "virtual-node", pod.Spec.NodeName)
		}
	})
	t.Run("CreateTaskSetWithInstanceID", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
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
		}
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		assert.Len(t, pods.Items, 1)
		for _, pod := range pods.Items {
			assert.NotNil(t, pod)
			assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
			assert.Equal(t, "testID", pod.Labels[common.LabelKeyControllerInstanceID])
			assert.Equal(t, "virtual-node", pod.Spec.NodeName)
		}
	})
}

func TestAssessAgentPodStatus(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	t.Run("Failed", func(t *testing.T) {
		pod1 := &apiv1.Pod{
			Status: apiv1.PodStatus{Phase: apiv1.PodFailed},
		}
		nodeStatus, msg := assessAgentPodStatus(ctx, pod1)
		assert.Equal(t, wfv1.NodeFailed, nodeStatus)
		assert.Empty(t, msg)
	})
	t.Run("Running", func(t *testing.T) {
		pod1 := &apiv1.Pod{
			Status: apiv1.PodStatus{Phase: apiv1.PodRunning},
		}

		nodeStatus, msg := assessAgentPodStatus(ctx, pod1)
		assert.Equal(t, wfv1.NodePhase(""), nodeStatus)
		assert.Empty(t, msg)
	})
	t.Run("Success", func(t *testing.T) {
		pod1 := &apiv1.Pod{
			Status: apiv1.PodStatus{Phase: apiv1.PodSucceeded},
		}
		nodeStatus, msg := assessAgentPodStatus(ctx, pod1)
		assert.Equal(t, wfv1.NodePhase(""), nodeStatus)
		assert.Empty(t, msg)
	})

}

func TestGetAgentPodName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-workflow",
			Namespace: "default",
		},
		Spec: wfv1.WorkflowSpec{
			ServiceAccountName: "custom-sa",
		},
	}

	t.Run("Per-workflow agent name", func(t *testing.T) {
		cancel, controller := newController(ctx, wf)
		defer cancel()
		controller.Config.Agent = nil // Default behavior

		woc := newWorkflowOperationCtx(ctx, wf, controller)
		podName := woc.getAgentPodName()
		assert.True(t, strings.HasSuffix(podName, "-agent"))
	})

	t.Run("Global agent name with custom SA", func(t *testing.T) {
		cancel, controller := newController(ctx, wf)
		defer cancel()
		controller.Config.Agent = &config.AgentConfig{
			RunMultipleWorkflow: true,
		}

		woc := newWorkflowOperationCtx(ctx, wf, controller)
		podName := woc.getAgentPodName()
		assert.Equal(t, "argo-agent-custom-sa", podName)
	})

	t.Run("Global agent name with default SA", func(t *testing.T) {
		wfNoSA := wf.DeepCopy()
		wfNoSA.Spec.ServiceAccountName = ""

		cancel, controller := newController(ctx, wfNoSA)
		defer cancel()
		controller.Config.Agent = &config.AgentConfig{
			RunMultipleWorkflow: true,
		}

		woc := newWorkflowOperationCtx(ctx, wfNoSA, controller)
		podName := woc.getAgentPodName()
		assert.Equal(t, "argo-agent-default", podName)
	})
}

func TestComputeAgentPodSpecHash(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-workflow",
			Namespace: "default",
		},
	}

	t.Run("Computes hash successfully", func(t *testing.T) {
		cancel, controller := newController(ctx, wf)
		defer cancel()

		woc := newWorkflowOperationCtx(ctx, wf, controller)
		hash, err := woc.computeAgentPodSpecHash(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64) // SHA256 hex string length
	})

	t.Run("Same config produces same hash", func(t *testing.T) {
		cancel, controller := newController(ctx, wf)
		defer cancel()

		woc := newWorkflowOperationCtx(ctx, wf, controller)
		hash1, err := woc.computeAgentPodSpecHash(ctx)
		require.NoError(t, err)

		hash2, err := woc.computeAgentPodSpecHash(ctx)
		require.NoError(t, err)

		assert.Equal(t, hash1, hash2)
	})
}

func TestCleanupAgentPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-workflow",
			Namespace: "default",
		},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				"test-node": {
					Type: wfv1.NodeTypeHTTP,
				},
			},
		},
	}

	t.Run("Returns false when no TaskSet nodes", func(t *testing.T) {
		wfNoTasks := wf.DeepCopy()
		wfNoTasks.Status.Nodes = map[string]wfv1.NodeStatus{}

		cancel, controller := newController(ctx, wfNoTasks)
		defer cancel()

		woc := newWorkflowOperationCtx(ctx, wfNoTasks, controller)
		shouldDelete := woc.cleanupAgentPod(ctx)
		assert.False(t, shouldDelete)
	})

	t.Run("Returns false when CreatePod is false", func(t *testing.T) {
		cancel, controller := newController(ctx, wf)
		defer cancel()
		createPod := false
		controller.Config.Agent = &config.AgentConfig{
			CreatePod: &createPod,
		}

		woc := newWorkflowOperationCtx(ctx, wf, controller)
		shouldDelete := woc.cleanupAgentPod(ctx)
		assert.False(t, shouldDelete)
	})

	t.Run("Returns true for per-workflow agent with tasks", func(t *testing.T) {
		cancel, controller := newController(ctx, wf)
		defer cancel()
		controller.Config.Agent = &config.AgentConfig{
			RunMultipleWorkflow: false,
		}
		controller.Config.Agent.SetDefaults()

		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.taskSet = map[string]wfv1.Template{
			"task1": {},
		}
		shouldDelete := woc.cleanupAgentPod(ctx)
		assert.True(t, shouldDelete)
	})
}
