package controller

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/plugins/spec"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	pluginutil "github.com/argoproj/argo-workflows/v4/workflow/util/plugin"
)

func TestExecuteTaskSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/agent/execute-task-set-workflow.yaml")
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal("@testdata/agent/execute-task-set.yaml", &ts)

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

// Test_createAgentPod_rateLimited asserts the transient-error contract of
// createAgentPod. When the controller's resource rate limiter denies the
// reservation, createPodFromBuild returns ErrResourceRateLimitReached, which
// createAgentPod must treat as transient: requeue the workflow and return
// (nil, nil), not a pod and not a hard error.
func Test_createAgentPod_rateLimited(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/agent/agent-task-set-workflow.yaml")

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
	wf := wfv1.MustUnmarshalWorkflow("@testdata/agent/agent-task-set-workflow.yaml")
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
	wf := wfv1.MustUnmarshalWorkflow("@testdata/agent/disable-agent-pod-creation.yaml")
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
	wf := wfv1.MustUnmarshalWorkflow("@testdata/agent/workflow-defined-executor-plugins.yaml")
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

func TestAgentPodServiceAccountTokens(t *testing.T) {
	for _, tt := range []struct {
		name            string
		account         string
		podSpecPatch    string
		wantAccount     string
		workflowToken   bool
		pluginToken     bool
		source          string
		pluginsDisabled bool
		wantError       string
	}{
		{name: "workflow account without Secret", account: "workflow-sa", workflowToken: true},
		{name: "default account without Secret", workflowToken: true},
		{name: "missing workflow account", account: "missing", workflowToken: true, wantError: `serviceaccounts "missing" not found`},
		{name: "patched workflow account", account: "missing", workflowToken: true, podSpecPatch: `{"serviceAccountName":"workflow-sa"}`, wantAccount: "workflow-sa"},
		{name: "missing patched account", account: "workflow-sa", workflowToken: true, podSpecPatch: `{"serviceAccountName":"missing"}`, wantError: `serviceaccounts "missing" not found`},
		{name: "patched default account", account: "missing", workflowToken: true, podSpecPatch: `{"serviceAccountName":""}`},
		{name: "workflow plugins disabled", workflowToken: true, pluginsDisabled: true, wantError: "workflow-level executor plugins are disabled"},
		{name: "workflow plugin without opt-in", account: "workflow-sa"},
		{name: "plugin own account", account: "workflow-sa", pluginToken: true},
		{name: "HTTP only", account: "workflow-sa", source: "none"},
		{name: "ConfigMap without token", account: "workflow-sa", source: "configmap"},
		{name: "ConfigMap own account", account: "workflow-sa", source: "configmap", pluginToken: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow("@testdata/agent/workflow-defined-executor-plugins.yaml")
			wf.Spec.ServiceAccountName = tt.account
			wf.Spec.PodSpecPatch = tt.podSpecPatch
			wf.Spec.ExecutorPlugins[0].Spec.Sidecar.AutomountWorkflowServiceAccountToken = tt.workflowToken
			wf.Spec.ExecutorPlugins[0].Spec.Sidecar.AutomountServiceAccountToken = tt.pluginToken
			// No token Secrets or legacy references, even for the projected-token cases.
			cancel, controller := newController(ctx, wf,
				&apiv1.ServiceAccount{ObjectMeta: v1.ObjectMeta{Name: "workflow-sa", Namespace: wf.Namespace}},
				&apiv1.ServiceAccount{ObjectMeta: v1.ObjectMeta{Name: "default", Namespace: wf.Namespace}},
				&apiv1.ServiceAccount{ObjectMeta: v1.ObjectMeta{Name: "test-sidecar-executor-plugin", Namespace: wf.Namespace}},
			)
			defer cancel()
			switch tt.source {
			case "none":
				wf.Spec.ExecutorPlugins = nil
			case "configmap":
				plugins, err := wf.Spec.AsExecutorPluginSpec()
				require.NoError(t, err)
				cm, err := pluginutil.ToConfigMap(&plugins[0])
				require.NoError(t, err)
				// A ConfigMap cannot opt into the workflow's account.
				cm.Data["sidecar.automountWorkflowServiceAccountToken"] = "true"
				plugin, err := pluginutil.FromConfigMap(cm)
				require.NoError(t, err)
				controller.executorPlugins = map[string]map[string]*spec.Plugin{
					wf.Namespace: {plugin.Name: plugin},
				}
				wf.Spec.ExecutorPlugins = nil
			}
			controller.enableWorkflowLevelExecutorPlugins = !tt.pluginsDisabled && tt.source == ""
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			pod, err := woc.createAgentPod(ctx)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				require.Nil(t, pod)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, pod)
			wantAccount := tt.account
			if tt.podSpecPatch != "" {
				wantAccount = tt.wantAccount
			}
			assert.Equal(t, wantAccount, pod.Spec.ServiceAccountName)
			assert.Equal(t, new(false), pod.Spec.AutomountServiceAccountToken)
			expectedContainers := 2
			if tt.source == "none" {
				expectedContainers = 1
			}
			require.Len(t, pod.Spec.Containers, expectedContainers)
			account := tt.account
			if account == "" {
				account = "default"
			}
			for _, container := range slices.Concat(pod.Spec.InitContainers, pod.Spec.Containers) {
				secret := account + ".service-account-token"
				if container.Name == "test-sidecar" {
					secret = ""
					if tt.pluginToken {
						secret = "test-sidecar-executor-plugin.service-account-token"
					}
				}
				assertContainerServiceAccountToken(t, pod, container, tt.workflowToken, secret)
			}
		})
	}
}

func TestAgentPodMixedPluginServiceAccounts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/agent/workflow-defined-executor-plugins.yaml")
	wf.Spec.ServiceAccountName = "unmerged-sa"
	wf.Spec.ExecutorPlugins[0].Spec.Sidecar.AutomountWorkflowServiceAccountToken = true
	for i, name := range []string{"other", "no-token"} {
		plugin := wf.Spec.ExecutorPlugins[0].DeepCopy()
		plugin.Name = name
		plugin.Spec.Sidecar.Container.Name = name
		plugin.Spec.Sidecar.Container.Ports[0].ContainerPort += int32(i + 1)
		plugin.Spec.Sidecar.AutomountWorkflowServiceAccountToken = false
		plugin.Spec.Sidecar.AutomountServiceAccountToken = name == "other"
		wf.Spec.ExecutorPlugins = append(wf.Spec.ExecutorPlugins, *plugin)
	}
	slices.Reverse(wf.Spec.ExecutorPlugins)
	cancel, controller := newController(ctx, wf,
		&apiv1.ServiceAccount{ObjectMeta: v1.ObjectMeta{Name: "workflow-sa", Namespace: wf.Namespace}},
		&apiv1.ServiceAccount{ObjectMeta: v1.ObjectMeta{Name: "other-executor-plugin", Namespace: wf.Namespace}},
	)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	// The pod and its tokens must use the resolved spec, including inherited accounts.
	woc.execWf = woc.execWf.DeepCopy()
	woc.execWf.Spec.ServiceAccountName = "workflow-sa"
	pod, err := woc.createAgentPod(ctx)
	require.NoError(t, err)
	require.NotNil(t, pod)
	assert.Equal(t, "workflow-sa", pod.Spec.ServiceAccountName)
	require.Len(t, pod.Spec.Containers, 4)
	for _, container := range slices.Concat(pod.Spec.InitContainers, pod.Spec.Containers) {
		switch container.Name {
		case "other":
			assertContainerServiceAccountToken(t, pod, container, false, "other-executor-plugin.service-account-token")
		case "no-token":
			assertContainerServiceAccountToken(t, pod, container, false, "")
		default:
			assertContainerServiceAccountToken(t, pod, container, true, "")
		}
	}
}

func assertContainerServiceAccountToken(t *testing.T, pod *apiv1.Pod, container apiv1.Container, projected bool, secret string) {
	t.Helper()
	var tokenMounts []apiv1.VolumeMount
	for _, mount := range container.VolumeMounts {
		if mount.MountPath == common.ServiceAccountTokenMountPath {
			tokenMounts = append(tokenMounts, mount)
		}
	}
	if !projected && secret == "" {
		assert.Empty(t, tokenMounts, container.Name)
		return
	}
	require.Len(t, tokenMounts, 1, container.Name)
	mount := tokenMounts[0]
	assert.True(t, mount.ReadOnly)
	assert.Empty(t, mount.SubPath, "token rotation requires mounting the directory")
	index := slices.IndexFunc(pod.Spec.Volumes, func(v apiv1.Volume) bool { return v.Name == mount.Name })
	require.NotEqual(t, -1, index, container.Name)
	volume := pod.Spec.Volumes[index]
	if !projected {
		require.NotNil(t, volume.Secret)
		assert.Equal(t, secret, volume.Secret.SecretName, container.Name)
		assert.Nil(t, volume.Projected)
		return
	}
	assert.Nil(t, volume.Secret)
	require.NotNil(t, volume.Projected)
	require.Len(t, volume.Projected.Sources, 3)
	token := volume.Projected.Sources[0].ServiceAccountToken
	require.NotNil(t, token)
	assert.Equal(t, "token", token.Path)
	assert.Equal(t, new(int64(3600)), token.ExpirationSeconds)
	assert.Empty(t, token.Audience, "use the Kubernetes API server's default audience")
	ca := volume.Projected.Sources[1].ConfigMap
	require.NotNil(t, ca)
	assert.Equal(t, "kube-root-ca.crt", ca.Name)
	assert.Equal(t, []apiv1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}}, ca.Items)
	namespace := volume.Projected.Sources[2].DownwardAPI
	require.NotNil(t, namespace)
	require.Len(t, namespace.Items, 1)
	assert.Equal(t, "namespace", namespace.Items[0].Path)
	require.NotNil(t, namespace.Items[0].FieldRef)
	assert.Equal(t, "metadata.namespace", namespace.Items[0].FieldRef.FieldPath)
}
