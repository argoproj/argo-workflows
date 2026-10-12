package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	armocks "github.com/argoproj/argo-workflows/v4/workflow/artifactrepositories/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

const pluginArtifactsWorkflow = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: plugin-artifacts
  namespace: default
spec:
  entrypoint: main
  executorPlugins:
    - metadata:
        name: hello
      spec:
        sidecar:
          container:
            name: hello-executor-plugin
            image: python:alpine3.23
            ports:
              - containerPort: 4355
            resources:
              requests:
                cpu: 100m
                memory: 32Mi
              limits:
                cpu: 200m
                memory: 64Mi
            securityContext:
              runAsUser: 1000
              runAsNonRoot: true
  templates:
    - name: main
      steps:
        - - name: with-artifacts
            template: with-artifacts
          - name: without-artifacts
            template: without-artifacts
    - name: with-artifacts
      plugin:
        hello: {}
      inputs:
        artifacts:
          - name: in
            path: /in.txt
            s3:
              key: some/input.txt
      outputs:
        artifacts:
          - name: out
            path: /out.txt
    - name: without-artifacts
      plugin:
        hello: {}
`

func TestPluginTemplateArchiveLocation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(pluginArtifactsWorkflow)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	taskSet, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, taskSet.Spec.Tasks, 2)

	withArtifacts, err := woc.wf.GetNodeByName("plugin-artifacts[0].with-artifacts")
	require.NoError(t, err)
	task := taskSet.Spec.Tasks[withArtifacts.ID]
	// the key has the name the node's pod would have, as a pod's artifacts do
	podName := woc.getPodName(withArtifacts.Name, "with-artifacts")
	assert.Equal(t, &wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{
		S3Bucket: wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
		Key:      "plugin-artifacts/" + podName,
	}}, task.ArchiveLocation)

	withoutArtifacts, err := woc.wf.GetNodeByName("plugin-artifacts[0].without-artifacts")
	require.NoError(t, err)
	assert.Nil(t, taskSet.Spec.Tasks[withoutArtifacts.ID].ArchiveLocation)
}

func TestPluginTemplateArchiveLocationUnresolved(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(pluginArtifactsWorkflow)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.artifactRepositories = armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket:  wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
			KeyFormat: "{{workflow.name}}/{{node.unknown}}",
		},
	})
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	node, err := woc.wf.GetNodeByName("plugin-artifacts[0].with-artifacts")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Contains(t, node.Message, "failed to resolve {{node.unknown}}")
}

// The archive location's key can use the template's input parameters.
func TestPluginTemplateArchiveLocationInputParameter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(pluginArtifactsWorkflow)
	tmpl := &wf.Spec.Templates[1]
	require.Equal(t, "with-artifacts", tmpl.Name)
	tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr("my-value")}}
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.artifactRepositories = armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket:  wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
			KeyFormat: "{{workflow.name}}/{{inputs.parameters.p}}",
		},
	})
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	taskSet, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	node, err := woc.wf.GetNodeByName("plugin-artifacts[0].with-artifacts")
	require.NoError(t, err)
	require.Contains(t, taskSet.Spec.Tasks, node.ID)
	location := taskSet.Spec.Tasks[node.ID].ArchiveLocation
	require.NotNil(t, location)
	require.NotNil(t, location.S3)
	assert.Equal(t, "plugin-artifacts/my-value", location.S3.Key)
}

func TestAgentPodPluginWorkspace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(pluginArtifactsWorkflow)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	controller.artifactRepositories = armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket: wfv1.S3Bucket{
				Endpoint:        "my-endpoint",
				Bucket:          "my-bucket",
				AccessKeySecret: &apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: "my-minio-cred"}, Key: "accesskey"},
				SecretKeySecret: &apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: "my-minio-cred"}, Key: "secretkey"},
			},
		},
	})
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pod, err := controller.kubeclientset.CoreV1().Pods("default").Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
	require.NoError(t, err)

	volumes := map[string]apiv1.Volume{}
	for _, v := range pod.Spec.Volumes {
		volumes[v.Name] = v
	}
	require.Contains(t, volumes, common.AgentWorkspaceVolumeName)
	assert.NotNil(t, volumes[common.AgentWorkspaceVolumeName].EmptyDir)

	mounts := func(name string) map[string]apiv1.VolumeMount {
		for _, c := range pod.Spec.Containers {
			if c.Name == name {
				byPath := map[string]apiv1.VolumeMount{}
				for _, m := range c.VolumeMounts {
					byPath[m.MountPath] = m
				}
				return byPath
			}
		}
		t.Fatalf("no container %s", name)
		return nil
	}

	agent := mounts(common.MainContainerName)
	assert.Equal(t, common.AgentWorkspaceVolumeName, agent[common.AgentWorkspacePath].Name)
	assert.Equal(t, volumeTmpDir.Name, agent["/tmp"].Name)
	secretMount, ok := agent[common.SecretVolMountPath+"/my-minio-cred"]
	require.True(t, ok, "the agent reads the artifact repository's secret where argoexec reads artifact secrets")
	assert.True(t, secretMount.ReadOnly)
	assert.Contains(t, secretMount.Name, "artifact-secret-", "the volume's name is not the secret's, for plugins to guess")
	secretVolume := volumes[secretMount.Name]
	require.NotNil(t, secretVolume.Secret)
	assert.Equal(t, "my-minio-cred", secretVolume.Secret.SecretName)
	assert.ElementsMatch(t, []apiv1.KeyToPath{{Key: "accesskey", Path: "accesskey"}, {Key: "secretkey", Path: "secretkey"}}, secretVolume.Secret.Items)
	assert.Equal(t, new(true), secretVolume.Secret.Optional)

	plugin := mounts("hello-executor-plugin")
	assert.Equal(t, common.AgentWorkspaceVolumeName, plugin[common.AgentWorkspacePath].Name)
	assert.NotContains(t, plugin, "/tmp", "plugins do not see the agent's archived artifacts")
	for path := range plugin {
		assert.NotContains(t, path, common.SecretVolMountPath, "plugins do not get the artifact secrets")
	}
}

func TestAgentPodWithoutPluginsHasNoWorkspace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template
  namespace: default
spec:
  entrypoint: http
  templates:
    - name: http
      http:
        url: http://example.com
`)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pod, err := controller.kubeclientset.CoreV1().Pods("default").Get(ctx, woc.getAgentPodName(), metav1.GetOptions{})
	require.NoError(t, err)
	for _, v := range pod.Spec.Volumes {
		assert.NotEqual(t, common.AgentWorkspaceVolumeName, v.Name)
		assert.NotEqual(t, volumeTmpDir.Name, v.Name)
		assert.NotContains(t, v.Name, "artifact-secret-")
	}
}

// pluginArtifactGCWorkflow has a plugin node with an artifact its template
// declares, and one the plugin reported that it only refers to.
const pluginArtifactGCWorkflow = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: plugin-artifacts
  namespace: default
spec:
  entrypoint: plugin
  artifactGC:
    strategy: OnWorkflowDeletion
  templates:
    - name: plugin
      plugin:
        hello: {}
      outputs:
        artifacts:
          - name: declared
            path: /declared.txt
    - name: http
      http:
        url: http://example.com
status:
  nodes:
    plugin-artifacts:
      id: plugin-artifacts
      name: plugin-artifacts
      type: Plugin
      templateName: plugin
      phase: Succeeded
      outputs:
        artifacts:
          - name: declared
            s3:
              key: plugin-artifacts/plugin-artifacts/declared.tgz
          - name: undeclared
            s3:
              key: data/the/plugin/referred/to
    plugin-artifacts-1:
      id: plugin-artifacts-1
      name: plugin-artifacts-1
      type: HTTP
      templateName: http
      phase: Succeeded
      outputs:
        artifacts:
          - name: other
            s3:
              key: other
`

func TestFindArtifactsToGCPluginNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(pluginArtifactGCWorkflow)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	results := woc.findArtifactsToGC(wfv1.ArtifactGCOnWorkflowDeletion)
	require.Len(t, results, 1)
	assert.Equal(t, "plugin-artifacts", results[0].NodeID)
	assert.Equal(t, "declared", results[0].Name)
}

// The artifact GC finalizer stays until the artifacts a plugin's template
// declares are deleted, and no longer waits on those it only refers to.
func TestAllArtifactsDeletedPluginNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(pluginArtifactGCWorkflow)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.False(t, woc.allArtifactsDeleted())

	node, err := woc.wf.Status.Nodes.Get("plugin-artifacts")
	require.NoError(t, err)
	require.Equal(t, "declared", node.Outputs.Artifacts[0].Name)
	node.Outputs.Artifacts[0].Deleted = true
	woc.wf.Status.Nodes.Set(ctx, node.ID, *node)
	assert.True(t, woc.allArtifactsDeleted())
}

// Templates of the same name in different WorkflowTemplates, and in the
// workflow itself, declare different artifacts. Artifact GC leaves alone the
// artifacts of a node whose template the workflow did not store, as it cannot
// tell which of them the template declares.
func TestFindArtifactsToGCPluginNodesTemplateRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	outputs := `
      outputs:
        artifacts:
          - name: local
            s3:
              key: local
          - name: a
            s3:
              key: a
          - name: b
            s3:
              key: b`
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: plugin-artifacts
  namespace: default
spec:
  entrypoint: plugin
  artifactGC:
    strategy: OnWorkflowDeletion
  templates:
    - name: plugin
      plugin:
        hello: {}
      outputs:
        artifacts:
          - name: local
            path: /local.txt
status:
  storedTemplates:
    namespaced/wt-a/plugin:
      name: plugin
      plugin:
        hello: {}
      outputs:
        artifacts:
          - name: a
            path: /a.txt
    namespaced/wt-b/plugin:
      name: plugin
      plugin:
        hello: {}
      outputs:
        artifacts:
          - name: b
            path: /b.txt
  nodes:
    local:
      id: local
      name: local
      type: Plugin
      templateName: plugin
      phase: Succeeded` + outputs + `
    ref:
      id: ref
      name: ref
      type: Plugin
      templateRef:
        name: wt-b
        template: plugin
      phase: Succeeded` + outputs + `
    scoped:
      id: scoped
      name: scoped
      type: Plugin
      templateName: plugin
      templateScope: namespaced/wt-a
      phase: Succeeded` + outputs + `
    unstored:
      id: unstored
      name: unstored
      type: Plugin
      templateRef:
        name: wt-c
        template: plugin
      phase: Succeeded` + outputs + `
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	gc := map[string]string{}
	for _, r := range woc.findArtifactsToGC(wfv1.ArtifactGCOnWorkflowDeletion) {
		gc[r.NodeID] += r.Name
	}
	assert.Equal(t, map[string]string{"local": "local", "ref": "b", "scoped": "a"}, gc)
}
