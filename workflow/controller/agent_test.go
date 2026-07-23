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

func TestWorkflowDefinedExecutorPluginsUsage(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template
  namespace: default
spec:
  podSpecPatch: |
    nodeName: virtual-node
  entrypoint: main
  executorPlugins:
  - spec:
      sidecar:
        container:
          name: test-sidecar
          image: busybox:1.35
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "200m"
              memory: "256Mi"
          securityContext:
            runAsUser: 1000
            runAsGroup: 1000
            runAsNonRoot: true
    metadata:
      name: test-sidecar
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
	assert.NotNil(t, wf)
	assert.Len(t, wf.Spec.ExecutorPlugins, 1)

	t.Run("ExecutorPluginLoadedFromWorkflowSpec", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		var ts wfv1.WorkflowTaskSet
		cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
		defer cancel()

		controller.Config.InstanceID = "testID"
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		sidecars, volumes, err := woc.getExecutorPlugins(ctx)
		require.NoError(t, err)
		assert.Len(t, sidecars, 1)
		assert.Equal(t, "test-sidecar", sidecars[0].Name)
		assert.Equal(t, "busybox:1.35", sidecars[0].Image)

		assert.Nil(t, volumes)
	})

	t.Run("AgentPodCreatedSuccessfully", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		var ts wfv1.WorkflowTaskSet
		cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
		defer cancel()

		controller.Config.InstanceID = "testID"
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		pod, err := woc.createAgentPod(ctx)
		require.NoError(t, err)
		require.NotNil(t, pod)

		containers := pod.Spec.Containers
		assert.NotEmpty(t, containers)

		var executorSidecar *apiv1.Container
		for _, container := range containers {
			if container.Name == "test-sidecar" && container.Image == "busybox:1.35" {
				executorSidecar = &container
			}
		}
		assert.NotNil(t, pod.Spec.Volumes)
		assert.NotNil(t, executorSidecar)
	})

	t.Run("AgentPodPluginFailedDueToSAMount", func(t *testing.T) {
		// NOTE: We do NOT create a ServiceAccount beforehand.
		// This test verifies that CreateAgentPod attempts to select
		// the correct ServiceAccount if the plugin has AutomountServiceAccountToken set
		wfCopy := wf.DeepCopy()
		var ts wfv1.WorkflowTaskSet
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wfCopy, ts, defaultServiceAccount)
		defer cancel()

		wfCopy.Spec.ExecutorPlugins[0].Spec.Sidecar.AutomountServiceAccountToken = true
		controller.Config.InstanceID = "testID"
		woc := newWorkflowOperationCtx(ctx, wfCopy, controller)
		pod, err := woc.createAgentPod(ctx)
		// agent tried to mount the service account with the correct name "test-sidecar-executor-plugin",
		// according to executorPlugin.metadata.name.
		require.ErrorContains(t, err, "serviceaccounts \"test-sidecar-executor-plugin\" not found")
		require.Nil(t, pod)
	})
}

// Test_createResourceAgentPod_artifactPluginSidecars asserts that every artifact driver
// registered in the controller config is installed into the resource-agent pod: one
// sidecar per driver carrying the plugin socket, var-run-argo (argoexec), the shared
// /tmp, and the service-account token; the socket volume on the pod; the socket mount
// on main; and the init container that copies argoexec for the sidecars. With no
// drivers registered the pod carries none of that.
func Test_createResourceAgentPod_artifactPluginSidecars(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(agentTaskSetWf)
	resourceAgentServiceAccount := &apiv1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "default-resource-agent",
			Namespace: "default",
		},
		Secrets: []apiv1.ObjectReference{{}},
	}

	t.Run("NoDriversRegistered", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, defaultServiceAccount, resourceAgentServiceAccount)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		pod, err := woc.createResourceAgentPod(ctx)
		require.NoError(t, err)
		require.NotNil(t, pod)

		assert.Len(t, pod.Spec.Containers, 1)
		assert.Empty(t, pod.Spec.InitContainers)
		for _, v := range pod.Spec.Volumes {
			assert.NotContains(t, v.Name, common.ArtifactPluginSidecarPrefix)
		}
	})

	t.Run("DriversRegistered", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf, defaultServiceAccount, resourceAgentServiceAccount, func(c *WorkflowController) {
			c.Config.ArtifactDrivers = []config.ArtifactDriver{
				{Name: "test", Image: "my-driver"},
				{Name: "other", Image: "my-driver"},
			}
			c.Config.Images["my-driver"] = config.Image{Entrypoint: []string{"/plugin-server"}}
		})
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		pod, err := woc.createResourceAgentPod(ctx)
		require.NoError(t, err)
		require.NotNil(t, pod)

		// main + one sidecar per registered driver
		require.Len(t, pod.Spec.Containers, 3)

		// the init container delivers argoexec into var-run-argo for the sidecars
		require.Len(t, pod.Spec.InitContainers, 1)
		initCtr := pod.Spec.InitContainers[0]
		require.GreaterOrEqual(t, len(initCtr.Command), 2)
		assert.Equal(t, "init", initCtr.Command[1])
		assert.Equal(t, []apiv1.VolumeMount{volumeMountVarArgo}, initCtr.VolumeMounts)

		podVolumeNames := make(map[string]bool, len(pod.Spec.Volumes))
		for _, v := range pod.Spec.Volumes {
			podVolumeNames[v.Name] = true
		}

		var mainCtr *apiv1.Container
		for i, c := range pod.Spec.Containers {
			if c.Name == common.MainContainerName {
				mainCtr = &pod.Spec.Containers[i]
			}
		}
		require.NotNil(t, mainCtr)
		mainMounts := make(map[string]string, len(mainCtr.VolumeMounts))
		for _, m := range mainCtr.VolumeMounts {
			mainMounts[m.Name] = m.MountPath
		}

		for _, driver := range []wfv1.ArtifactPluginName{"test", "other"} {
			var sidecar *apiv1.Container
			for i, c := range pod.Spec.Containers {
				if c.Name == common.ArtifactPluginSidecarPrefix+string(driver) {
					sidecar = &pod.Spec.Containers[i]
				}
			}
			require.NotNil(t, sidecar, "sidecar for driver %s not found", driver)
			assert.Equal(t, "my-driver", sidecar.Image)

			mounts := make(map[string]string, len(sidecar.VolumeMounts))
			for _, m := range sidecar.VolumeMounts {
				mounts[m.Name] = m.MountPath
			}
			socketVolume := driver.Volume()
			assert.Equal(t, driver.SocketDir(), mounts[socketVolume.Name], "sidecar must serve on its socket dir")
			assert.Equal(t, volumeMountVarArgo.MountPath, mounts[volumeMountVarArgo.Name], "sidecar must reach argoexec")
			assert.Equal(t, "/tmp", mounts[volumeMountTmpDir.Name], "sidecar must share main's /tmp for artifact downloads")
			// workload pods hand drivers the pod SA token via automount; the agent pod
			// disables automount so the token must be mounted explicitly
			foundToken := false
			for _, m := range sidecar.VolumeMounts {
				if m.MountPath == common.ServiceAccountTokenMountPath {
					foundToken = true
				}
			}
			assert.True(t, foundToken, "sidecar must carry the service-account token for secret resolution")

			// the socket is an emptyDir volume on the pod, mounted by main so it can dial the plugin
			assert.True(t, podVolumeNames[socketVolume.Name], "socket volume for %s missing from pod", driver)
			assert.Equal(t, driver.SocketDir(), mainMounts[socketVolume.Name], "main must mount %s's socket dir", driver)
		}
	})
}
