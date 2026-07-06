package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
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

var agentTaskSetWf = `apiVersion: argoproj.io/v1alpha1
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
              parameters: [{name: url, value: "https://example.com/foo.json"}]
    - name: http
      inputs:
        parameters:
          - name: url
      http:
       url: "{{inputs.parameters.url}}"
`

// Test_createAgentPod_rateLimited asserts the transient-error contract of
// createAgentPod. When the controller's resource rate limiter denies the
// reservation, createPodFromBuild returns ErrResourceRateLimitReached, which
// createAgentPod must treat as transient: requeue the workflow and return
// (nil, nil), not a pod and not a hard error.
func Test_createAgentPod_rateLimited(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(agentTaskSetWf)

	t.Run("RateLimitedRequeuesAndReturnsNilNil", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		// Limit 0 / Burst 0 forces every Reserve() to be denied.
		cancel, controller := newController(ctx, wf, defaultServiceAccount, func(c *WorkflowController) {
			c.Config.ResourceRateLimit = &config.ResourceRateLimit{Limit: 0, Burst: 0}
		})
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		pod, err := woc.createAgentPod(ctx)

		// Transient rate-limit contract: no error, no pod.
		require.NoError(t, err)
		assert.Nil(t, pod)
		// The workflow must have been requeued for a later retry. requeue() uses
		// AddRateLimited, which schedules the add after a short backoff, so poll
		// until the item lands on the queue.
		assert.Eventually(t, func() bool {
			return woc.controller.wfQueue.Len() > 0
		}, 5*time.Second, 5*time.Millisecond, "expected the workflow to be requeued after rate-limit")
		// No agent pod should have been created in the cluster.
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, pods.Items)
	})

	t.Run("NotRateLimitedCreatesPod", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		// Limit 1 / Burst 1 allows the single reservation to succeed.
		cancel, controller := newController(ctx, wf, defaultServiceAccount, func(c *WorkflowController) {
			c.Config.ResourceRateLimit = &config.ResourceRateLimit{Limit: 1, Burst: 1}
		})
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		pod, err := woc.createAgentPod(ctx)

		require.NoError(t, err)
		require.NotNil(t, pod)
		assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
	})

	t.Run("RateLimitedRecoversExistingPod", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, defaultServiceAccount, func(c *WorkflowController) {
			c.Config.ResourceRateLimit = &config.ResourceRateLimit{Limit: 0, Burst: 0}
		})
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		// Pre-create the agent pod in the fake cluster WITHOUT populating the
		// informer store: the early informer GetPod misses it, and the rate
		// limiter denies the create before the AlreadyExists→Get recovery in
		// createPodFromBuild can run. createAgentPod must recover the existing
		// pod via a direct Get instead of requeueing with no pod.
		podName := woc.getAgentPodName()
		existing := &apiv1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      podName,
				Namespace: "default",
			},
		}
		_, err := woc.controller.kubeclientset.CoreV1().Pods("default").Create(ctx, existing, v1.CreateOptions{})
		require.NoError(t, err)

		pod, err := woc.createAgentPod(ctx)
		require.NoError(t, err)
		require.NotNil(t, pod, "rate-limited create must recover the pre-existing agent pod")
		assert.Equal(t, podName, pod.Name)
	})
}

// Test_createAgentPod_alreadyExists asserts the AlreadyExists recovery path:
// when the informer store is empty (so the early GetPod returns nil) but the
// pod already exists in the cluster, createPod returns an AlreadyExists error
// and createPodFromBuild recovers by fetching the existing pod. createAgentPod
// must return that existing pod with no error.
func Test_createAgentPod_alreadyExists(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(agentTaskSetWf)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Pre-create a pod under the deterministic agent pod name directly in the
	// fake cluster. The informer store is NOT populated with it, so the early
	// informer GetPod returns nil and createAgentPod proceeds to createPod,
	// which then hits AlreadyExists and recovers via a direct Get.
	podName := woc.getAgentPodName()
	existing := &apiv1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: "default",
		},
	}
	_, err := woc.controller.kubeclientset.CoreV1().Pods("default").Create(ctx, existing, v1.CreateOptions{})
	require.NoError(t, err)

	pod, err := woc.createAgentPod(ctx)
	require.NoError(t, err)
	require.NotNil(t, pod)
	assert.Equal(t, podName, pod.Name)
}

func TestDisableAgentPodCreation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
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
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.controller.Config.DisableAgentPodCreation = true
	defer cancel()
	woc.operate(ctx)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, pods.Items)
}
