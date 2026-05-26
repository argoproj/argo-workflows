package controller

import (
	"strings"
	"testing"

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

// TestAgentPluginShareWiring pins the cross-container handoff invariant that
// makes archiveAgentLogs work for plugin-backed archive locations: the agent
// main container and each artifact-plugin sidecar must agree on the same path
// string for the same bytes. With this wiring, what the agent writes to
// /argo/plugin-share/<name>/<file> the sidecar reads at the same path via its
// SubPath=<name> mount.
//
// Regression guard for the TestResourceLogPlugin/Basic e2e failure where the
// agent wrote /tmp/agent-main-logs-XXX.log in its own emptyDir and the plugin
// sidecar's RPC handler got ENOENT trying to open the same path from its own
// (unshared) filesystem.
func TestAgentPluginShareWiring(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: agent-plugin-share-wf
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      archiveLocation:
        archiveLogs: true
        plugin:
          name: test
      resource:
        action: create
        manifest: |
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: noop
      outputs:
        artifacts:
          - name: main-logs
            plugin:
              name: test
`)

	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.Config.ArtifactDrivers = []config.ArtifactDriver{
		{Name: "test", Image: "busybox"},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.artifactRepository = &wfv1.ArtifactRepository{}

	pod, err := woc.createAgentPod(ctx)
	require.NoError(t, err)
	require.NotNil(t, pod)

	// 1. The shared emptyDir is on the pod.
	var shareVol *apiv1.Volume
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == agentPluginShareVolumeName {
			shareVol = &pod.Spec.Volumes[i]
			break
		}
	}
	require.NotNil(t, shareVol, "agent-plugin-share volume missing from agent pod")
	require.NotNil(t, shareVol.EmptyDir, "agent-plugin-share must be an emptyDir")

	// Find the agent main and the test-plugin sidecar.
	var mainCtr, sidecarCtr *apiv1.Container
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		switch c.Name {
		case common.MainContainerName:
			mainCtr = c
		case common.ArtifactPluginSidecarPrefix + "test":
			sidecarCtr = c
		}
	}
	require.NotNil(t, mainCtr, "agent main container missing")
	require.NotNil(t, sidecarCtr, "test plugin sidecar missing")

	// 2. Agent main mounts the volume at the root (no SubPath) so it can
	//    write into any plugin's subdirectory.
	var mainMount *apiv1.VolumeMount
	for i := range mainCtr.VolumeMounts {
		if mainCtr.VolumeMounts[i].Name == agentPluginShareVolumeName {
			mainMount = &mainCtr.VolumeMounts[i]
			break
		}
	}
	require.NotNil(t, mainMount, "agent main is missing agent-plugin-share mount")
	assert.Equal(t, common.AgentPluginShareDir, mainMount.MountPath)
	assert.Empty(t, mainMount.SubPath, "agent main must see the whole volume, not a SubPath")

	// 3. Sidecar mounts SubPath=<plugin-name> at the same path the agent
	//    main writes to. This is what makes driver.Save(path) work: the
	//    path string the agent passes over RPC resolves to the same bytes
	//    inside the sidecar.
	var sidecarMount *apiv1.VolumeMount
	for i := range sidecarCtr.VolumeMounts {
		if sidecarCtr.VolumeMounts[i].Name == agentPluginShareVolumeName {
			sidecarMount = &sidecarCtr.VolumeMounts[i]
			break
		}
	}
	require.NotNil(t, sidecarMount, "test plugin sidecar is missing agent-plugin-share mount")
	assert.Equal(t, "test", sidecarMount.SubPath, "sidecar mount must be scoped to its plugin name")
	assert.Equal(t, common.AgentPluginShareDir+"/test", sidecarMount.MountPath,
		"sidecar mount path must equal what the agent writes to (common.AgentPluginShareDir/<plugin-name>)")
}

// TestAgentPluginShareSkippedWithoutPluginSidecars asserts the share volume
// is only added when the agent pod actually has artifact-plugin sidecars —
// otherwise it's dead weight on a frequently-created pod.
func TestAgentPluginShareSkippedWithoutPluginSidecars(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: agent-noplugin-wf
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      resource:
        action: create
        manifest: |
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: noop
`)

	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.artifactRepository = &wfv1.ArtifactRepository{}

	pod, err := woc.createAgentPod(ctx)
	require.NoError(t, err)
	require.NotNil(t, pod)

	for _, v := range pod.Spec.Volumes {
		assert.NotEqual(t, agentPluginShareVolumeName, v.Name,
			"share volume must not be added when no artifact-plugin sidecars exist")
	}
	for _, c := range pod.Spec.Containers {
		for _, vm := range c.VolumeMounts {
			assert.NotEqual(t, agentPluginShareVolumeName, vm.Name,
				"share volume must not be mounted on any container when no artifact-plugin sidecars exist")
		}
	}
}
