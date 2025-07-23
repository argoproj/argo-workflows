package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/util"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	armocks "github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories/mocks"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	wfutil "github.com/argoproj/argo-workflows/v3/workflow/util"
)

// Deprecated
func unmarshalTemplate(yamlStr string) *wfv1.Template {
	return wfv1.MustUnmarshalTemplate(yamlStr)
}

// newWoc a new operation context suitable for testing
func newWoc(ctx context.Context, wfs ...wfv1.Workflow) *wfOperationCtx {
	var wf *wfv1.Workflow
	if len(wfs) == 0 {
		wf = wfv1.MustUnmarshalWorkflow(helloWorldWf)
	} else {
		wf = &wfs[0]
	}
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	return woc
}

var scriptWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: script-with-input-artifact
    inputs:
      artifacts:
      - name: kubectl
        path: /bin/kubectl
        http:
          url: https://storage.googleapis.com/kubernetes-release/release/v1.8.0/bin/linux/amd64/kubectl
    script:
      image: alpine:latest
      command: [sh]
      source: |
        ls /bin/kubectl
`

var scriptTemplateWithInputArtifact = `
name: script-with-input-artifact
inputs:
  artifacts:
  - name: kubectl
    path: /bin/kubectl
    http:
      url: https://storage.googleapis.com/kubernetes-release/release/v1.8.0/bin/linux/amd64/kubectl
script:
  image: alpine:latest
  command: [sh]
  source: |
    ls /bin/kubectl
`

// TestScriptTemplateWithVolume ensure we can a script pod with input artifacts
func TestScriptTemplateWithVolume(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := unmarshalTemplate(scriptTemplateWithInputArtifact)
	woc := newWoc(ctx)
	_, err := woc.executeScript(ctx, tmpl.Name, "", tmpl, &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
}

var scriptTemplateWithOptionalInputArtifactProvided = `
name: script-with-input-artifact
inputs:
  artifacts:
  - name: manifest
    path: /manifest
    optional: true
    http:
        url: https://raw.githubusercontent.com/argoproj/argo-workflows/stable/manifests/install.yaml
script:
  image: alpine:latest
  command: [sh]
  source: |
    ls -al
`

var scriptTemplateWithOptionalInputArtifactProvidedAndOverlappedPath = `
name: script-with-input-artifact
inputs:
  artifacts:
  - name: manifest
    path: /manifest
    optional: true
    http:
        url: https://raw.githubusercontent.com/argoproj/argo-workflows/stable/manifests/install.yaml
script:
  volumeMounts:
  - mountPath: /manifest
    name: my-mount
  image: alpine:latest
  command: [sh]
  source: |
    ls -al
`

// TestScriptTemplateWithoutVolumeOptionalArtifact ensure we can a script pod with input artifacts
func TestScriptTemplateWithoutVolumeOptionalArtifact(t *testing.T) {
	volumeMount := apiv1.VolumeMount{
		Name:             "input-artifacts",
		ReadOnly:         false,
		MountPath:        "/manifest",
		SubPath:          "manifest",
		MountPropagation: nil,
		SubPathExpr:      "",
	}

	customVolumeMount := apiv1.VolumeMount{
		Name:             "my-mount",
		ReadOnly:         false,
		MountPath:        "/manifest",
		SubPath:          "",
		MountPropagation: nil,
		SubPathExpr:      "",
	}

	customVolumeMountForInit := apiv1.VolumeMount{
		Name:             "my-mount",
		ReadOnly:         false,
		MountPath:        filepath.Join(common.ExecutorMainFilesystemDir, "/manifest"),
		SubPath:          "",
		MountPropagation: nil,
		SubPathExpr:      "",
	}

	// Ensure that volume mount is added when artifact is provided
	tmpl := unmarshalTemplate(scriptTemplateWithOptionalInputArtifactProvided)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	mainCtr := tmpl.Script.Container
	mainCtr.Args = append(mainCtr.Args, common.ExecutorScriptSourcePath)
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{})
	require.NoError(t, err)
	// Note: pod.Spec.Containers[0] is wait
	assert.Contains(t, pod.Spec.Containers[1].VolumeMounts, volumeMount)
	assert.NotContains(t, pod.Spec.Containers[1].VolumeMounts, customVolumeMount)
	assert.NotContains(t, pod.Spec.InitContainers[0].VolumeMounts, customVolumeMountForInit)

	// Ensure that volume mount is added to initContainer when artifact is provided
	// and the volume is mounted manually in the template
	tmpl = unmarshalTemplate(scriptTemplateWithOptionalInputArtifactProvidedAndOverlappedPath)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	wf.Spec.Volumes = append(wf.Spec.Volumes, apiv1.Volume{Name: "my-mount"})
	woc = newWoc(ctx, *wf)
	mainCtr = tmpl.Script.Container
	mainCtr.Args = append(mainCtr.Args, common.ExecutorScriptSourcePath)
	pod, err = woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{includeScriptOutput: true})
	require.NoError(t, err)
	assert.NotContains(t, pod.Spec.Containers[1].VolumeMounts, volumeMount)
	assert.Contains(t, pod.Spec.Containers[1].VolumeMounts, customVolumeMount)
	assert.Contains(t, pod.Spec.InitContainers[0].VolumeMounts, customVolumeMountForInit)
}

// TestWFLevelServiceAccount verifies the ability to carry forward the service account name
// for the pod from workflow.spec.serviceAccountName.
func TestWFLevelServiceAccount(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.ServiceAccountName = "foo"
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "foo", pod.Spec.ServiceAccountName)
}

// TestTmplServiceAccount verifies the ability to carry forward the Template level service account name
// for the pod from workflow.spec.serviceAccountName.
func TestTmplServiceAccount(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.ServiceAccountName = "foo"
	woc.execWf.Spec.Templates[0].ServiceAccountName = "tmpl"
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "tmpl", pod.Spec.ServiceAccountName)
}

// TestWFLevelAutomountServiceAccountToken verifies the ability to carry forward workflow level AutomountServiceAccountToken to Podspec.
func TestWFLevelAutomountServiceAccountToken(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo")
	require.NoError(t, err)

	falseValue := false
	woc.execWf.Spec.AutomountServiceAccountToken = &falseValue
	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.False(t, *pod.Spec.AutomountServiceAccountToken)
}

// TestTmplLevelAutomountServiceAccountToken verifies the ability to carry forward template level AutomountServiceAccountToken to Podspec.
func TestTmplLevelAutomountServiceAccountToken(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo")
	require.NoError(t, err)

	trueValue := true
	falseValue := false
	woc.execWf.Spec.AutomountServiceAccountToken = &trueValue
	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	woc.execWf.Spec.Templates[0].AutomountServiceAccountToken = &falseValue
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.False(t, *pod.Spec.AutomountServiceAccountToken)
}

// verifyServiceAccountTokenVolumeMount is a helper function to verify service account token volume in a container.
func verifyServiceAccountTokenVolumeMount(t *testing.T, ctr apiv1.Container, volName, mountPath string) {
	for _, vol := range ctr.VolumeMounts {
		if vol.Name == volName && vol.MountPath == mountPath {
			return
		}
	}
	t.Fatalf("%v does not have serviceAccountToken mounted properly (name: %s, mountPath: %s)", ctr, volName, mountPath)
}

// TestWFLevelExecutorServiceAccountName verifies the ability to carry forward workflow level AutomountServiceAccountToken to Podspec.
func TestWFLevelExecutorServiceAccountName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo")
	require.NoError(t, err)

	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "exec-sa-token", pod.Spec.Volumes[2].Name)

	waitCtr := pod.Spec.Containers[0]
	verifyServiceAccountTokenVolumeMount(t, waitCtr, "exec-sa-token", "/var/run/secrets/kubernetes.io/serviceaccount")
}

// TestTmplLevelExecutorServiceAccountName verifies the ability to carry forward template level AutomountServiceAccountToken to Podspec.
func TestTmplLevelExecutorServiceAccountName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo")
	require.NoError(t, err)
	_, err = util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "tmpl")
	require.NoError(t, err)

	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	woc.execWf.Spec.Templates[0].Executor = &wfv1.ExecutorConfig{ServiceAccountName: "tmpl"}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "exec-sa-token", pod.Spec.Volumes[2].Name)

	waitCtr := pod.Spec.Containers[0]
	verifyServiceAccountTokenVolumeMount(t, waitCtr, "exec-sa-token", "/var/run/secrets/kubernetes.io/serviceaccount")
}

// TestCtrlLevelExecutorSecurityContext verifies the ability to carry forward Controller level SecurityContext to Podspec.
func TestCtrlLevelExecutorSecurityContext(t *testing.T) {
	var user int64 = 1000
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo")
	require.NoError(t, err)
	_, err = util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "tmpl")
	require.NoError(t, err)

	woc.controller.Config.Executor = &apiv1.Container{SecurityContext: &apiv1.SecurityContext{RunAsUser: &user}}
	woc.execWf.Spec.Templates[0].Executor = &wfv1.ExecutorConfig{ServiceAccountName: "tmpl"}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]

	waitCtr := pod.Spec.Containers[0]
	assert.NotNil(t, waitCtr.SecurityContext)
	if waitCtr.SecurityContext != nil {
		assert.NotNil(t, waitCtr.SecurityContext.RunAsUser)
		if waitCtr.SecurityContext.RunAsUser != nil {
			assert.Equal(t, int64(1000), *waitCtr.SecurityContext.RunAsUser)
		}
	}
}

// TestImagePullSecrets verifies the ability to carry forward imagePullSecrets from workflow.spec
func TestImagePullSecrets(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.ImagePullSecrets = []apiv1.LocalObjectReference{
		{
			Name: "secret-name",
		},
	}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "secret-name", pod.Spec.ImagePullSecrets[0].Name)
}

// TestAffinity verifies the ability to carry forward affinity rules
func TestAffinity(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.Affinity = &apiv1.Affinity{
		NodeAffinity: &apiv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
				NodeSelectorTerms: []apiv1.NodeSelectorTerm{
					{
						MatchExpressions: []apiv1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/e2e-az-name",
								Operator: apiv1.NodeSelectorOpIn,
								Values: []string{
									"e2e-az1",
									"e2e-az2",
								},
							},
						},
					},
				},
			},
		},
	}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.Affinity)
}

// TestTolerations verifies the ability to carry forward tolerations.
func TestTolerations(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.Templates[0].Tolerations = []apiv1.Toleration{{
		Key:      "nvidia.com/gpu",
		Operator: "Exists",
		Effect:   "NoSchedule",
	}}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.Tolerations)
	assert.Equal(t, "nvidia.com/gpu", pod.Spec.Tolerations[0].Key)
}

// TestMetadata verifies ability to carry forward annotations and labels
func TestMetadata(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.ObjectMeta)
	assert.NotNil(t, pod.Annotations)
	assert.NotNil(t, pod.Labels)
	for k, v := range woc.execWf.Spec.Templates[0].Metadata.Annotations {
		assert.Equal(t, pod.Annotations[k], v)
	}
	for k, v := range woc.execWf.Spec.Templates[0].Metadata.Labels {
		assert.Equal(t, pod.Labels[k], v)
	}
}

// TestWorkflowControllerArchiveConfig verifies archive location substitution of workflow
func TestWorkflowControllerArchiveConfig(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{workflow.name}}/{{pod.name}}",
	}})
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func setArtifactRepository(controller *WorkflowController, repo *wfv1.ArtifactRepository) {
	controller.artifactRepositories = armocks.DummyArtifactRepositories(repo)
}

// TestConditionalNoAddArchiveLocation verifies we do not add archive location if it is not needed
func TestConditionalNoAddArchiveLocation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "path/in/bucket",
	}})
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	tmpl, err := getPodTemplate(&pod)
	require.NoError(t, err)
	assert.Nil(t, tmpl.ArchiveLocation)
}

// TestConditionalAddArchiveLocationArchiveLogs verifies we do  add archive location if it is needed for logs
func TestConditionalAddArchiveLocationArchiveLogs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket: wfv1.S3Bucket{
				Bucket: "foo",
			},
			KeyFormat: "path/in/bucket",
		},
		ArchiveLogs: ptr.To(true),
	})
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	tmpl, err := getPodTemplate(&pod)
	require.NoError(t, err)
	assert.NotNil(t, tmpl.ArchiveLocation)
}

// TestConditionalArchiveLocation verifies we add archive location when it is needed
func TestConditionalArchiveLocation(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	wf.Spec.Templates[0].Outputs = wfv1.Outputs{
		Artifacts: []wfv1.Artifact{
			{
				Name: "foo",
				Path: "/tmp/file",
			},
		},
	}
	woc := newWoc(ctx)
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "path/in/bucket",
	}})
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	tmpl, err := getPodTemplate(&pod)
	require.NoError(t, err)
	assert.Nil(t, tmpl.ArchiveLocation)
}

// TestConditionalAddArchiveLocationTemplateArchiveLogs verifies we do  add archive location if it is needed for logs
func TestConditionalAddArchiveLocationTemplateArchiveLogs(t *testing.T) {
	tests := []struct {
		controllerArchiveLog bool
		workflowArchiveLog   string
		templateArchiveLog   string
		finalArchiveLog      bool
	}{
		{true, "true", "true", true},
		{true, "true", "false", true},
		{true, "false", "true", true},
		{true, "false", "false", true},
		{false, "true", "true", true},
		{false, "true", "false", false},
		{false, "false", "true", true},
		{false, "false", "false", false},
		{true, "true", "", true},
		{true, "false", "", true},
		{true, "", "true", true},
		{true, "", "false", true},
		{false, "true", "", true},
		{false, "false", "", false},
		{false, "", "true", true},
		{false, "", "false", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("controllerArchiveLog: %t, workflowArchiveLog: %s, templateArchiveLog: %s, finalArchiveLog: %t", tt.controllerArchiveLog, tt.workflowArchiveLog, tt.templateArchiveLog, tt.finalArchiveLog), func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
			if tt.workflowArchiveLog != "" {
				workflowArchiveLog, _ := strconv.ParseBool(tt.workflowArchiveLog)
				wf.Spec.ArchiveLogs = ptr.To(workflowArchiveLog)
			}
			if tt.templateArchiveLog != "" {
				templateArchiveLog, _ := strconv.ParseBool(tt.templateArchiveLog)
				wf.Spec.Templates[0].ArchiveLocation = &wfv1.ArtifactLocation{
					ArchiveLogs: ptr.To(templateArchiveLog),
				}
			}
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{
				ArchiveLogs: ptr.To(tt.controllerArchiveLog),
				S3: &wfv1.S3ArtifactRepository{
					S3Bucket: wfv1.S3Bucket{
						Bucket: "foo",
					},
					KeyFormat: "path/in/bucket",
				},
			})
			woc.operate(ctx)
			pods, err := listPods(ctx, woc)
			require.NoError(t, err)
			assert.Len(t, pods.Items, 1)
			pod := pods.Items[0]
			tmpl, err := getPodTemplate(&pod)
			require.NoError(t, err)
			assert.Equal(t, tt.finalArchiveLog, tmpl.ArchiveLocation.IsArchiveLogs())
		})
	}
}

func Test_createWorkflowPod_rateLimited(t *testing.T) {
	for limit, limited := range map[config.ResourceRateLimit]bool{
		{Limit: 0, Burst: 0}: true,
		{Limit: 1, Burst: 1}: false,
		{Limit: 0, Burst: 1}: false,
		{Limit: 1, Burst: 1}: false,
	} {
		t.Run(fmt.Sprintf("%v", limit), func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, wf, func(c *WorkflowController) {
				c.Config.ResourceRateLimit = &limit
			})
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			x := woc.wf.Status.Nodes[woc.wf.Name]
			assert.Equal(t, wfv1.NodePending, x.Phase)
			if limited {
				assert.Equal(t, "resource creation rate-limit reached", x.Message)
			} else {
				assert.Empty(t, x.Message)
			}
		})
	}
}

func Test_createWorkflowPod_containerName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	pod, err := woc.createWorkflowPod(ctx, "", []apiv1.Container{{Name: "invalid", Command: []string{""}}}, &wfv1.Template{}, &createWorkflowPodOpts{})
	require.NoError(t, err)
	assert.Equal(t, common.MainContainerName, pod.Spec.Containers[1].Name)
}

var emissaryCmd = []string{"/var/run/argo/argoexec", "emissary"}

func Test_createWorkflowPod_emissary(t *testing.T) {

	t.Run("NoCommand", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		_, err := woc.createWorkflowPod(ctx, "", []apiv1.Container{{Image: "docker/whalesay:nope"}}, &wfv1.Template{Name: "my-tmpl"}, &createWorkflowPodOpts{})
		require.EqualError(t, err, "failed to look-up entrypoint/cmd for image \"docker/whalesay:nope\", you must either explicitly specify the command, or list the image's command in the index: https://argo-workflows.readthedocs.io/en/latest/workflow-executors/#emissary-emissary: GET https://index.docker.io/v2/docker/whalesay/manifests/nope: MANIFEST_UNKNOWN: manifest unknown; unknown tag=nope")
	})
	t.Run("CommandNoArgs", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		pod, err := woc.createWorkflowPod(ctx, "", []apiv1.Container{{Command: []string{"foo"}}}, &wfv1.Template{}, &createWorkflowPodOpts{})
		require.NoError(t, err)
		cmd := append(append(emissaryCmd, woc.getExecutorLogOpts(ctx)...), "--", "foo")
		assert.Equal(t, cmd, pod.Spec.Containers[1].Command)
	})
	t.Run("NoCommandWithImageIndex", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		pod, err := woc.createWorkflowPod(ctx, "", []apiv1.Container{{Image: "my-image"}}, &wfv1.Template{}, &createWorkflowPodOpts{})
		require.NoError(t, err)
		cmd := append(append(emissaryCmd, woc.getExecutorLogOpts(ctx)...), "--", "my-entrypoint")
		assert.Equal(t, cmd, pod.Spec.Containers[1].Command)
		assert.Equal(t, []string{"my-cmd"}, pod.Spec.Containers[1].Args)
	})
	t.Run("NoCommandWithArgsWithImageIndex", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		pod, err := woc.createWorkflowPod(ctx, "", []apiv1.Container{{Image: "my-image", Args: []string{"foo"}}}, &wfv1.Template{}, &createWorkflowPodOpts{})
		require.NoError(t, err)
		cmd := append(append(emissaryCmd, woc.getExecutorLogOpts(ctx)...), "--", "my-entrypoint")
		assert.Equal(t, cmd, pod.Spec.Containers[1].Command)
		assert.Equal(t, []string{"foo"}, pod.Spec.Containers[1].Args)
	})
	t.Run("CommandFromPodMetadataPatch", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		podMetadata := &metav1.ObjectMeta{
			Annotations: map[string]string{"test-annotation": "test-annotation-value"},
			Labels:      map[string]string{"test-label": "test-label-value"},
		}
		podMetadataPatch, err := json.Marshal(podMetadata)
		require.NoError(t, err)
		pod, err := woc.createWorkflowPod(ctx, "", []apiv1.Container{{Command: []string{"foo"}}}, &wfv1.Template{PodMetadataPatch: string(podMetadataPatch)}, &createWorkflowPodOpts{})
		require.NoError(t, err)
		assert.Equal(t, "test-annotation-value", pod.Annotations["test-annotation"])
		assert.Equal(t, "test-label-value", pod.Labels["test-label"])

	})
	t.Run("CommandFromPodSpecPatch", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		podSpec := &apiv1.PodSpec{}
		podSpec.Containers = []apiv1.Container{{
			Name:    "main",
			Command: []string{"bar"},
		}}
		podSpecPatch, err := json.Marshal(podSpec)
		require.NoError(t, err)
		pod, err := woc.createWorkflowPod(ctx, "", []apiv1.Container{{Command: []string{"foo"}}}, &wfv1.Template{PodSpecPatch: string(podSpecPatch)}, &createWorkflowPodOpts{})
		require.NoError(t, err)
		cmd := append(append(emissaryCmd, woc.getExecutorLogOpts(ctx)...), "--", "bar")
		assert.Equal(t, cmd, pod.Spec.Containers[1].Command)
	})
}

// TestVolumeAndVolumeMounts verifies the ability to carry forward volumes and volumeMounts from workflow.spec
func TestVolumeAndVolumeMounts(t *testing.T) {
	volumes := []apiv1.Volume{
		{
			Name: "volume-name",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	volumeMounts := []apiv1.VolumeMount{
		{
			Name:      "volume-name",
			MountPath: "/test",
		},
	}

	// For emissary executor
	t.Run("Emissary", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		woc.volumes = volumes
		woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts

		tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
		require.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		require.NoError(t, err)
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		require.Len(t, pod.Spec.Volumes, 3)
		assert.Equal(t, "var-run-argo", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "tmp-dir-argo", pod.Spec.Volumes[1].Name)
		assert.Equal(t, "volume-name", pod.Spec.Volumes[2].Name)

		require.Len(t, pod.Spec.InitContainers, 1)
		init := pod.Spec.InitContainers[0]
		require.Len(t, init.VolumeMounts, 1)
		assert.Equal(t, "var-run-argo", init.VolumeMounts[0].Name)

		containers := pod.Spec.Containers
		require.Len(t, containers, 2)
		wait := containers[0]
		require.Len(t, wait.VolumeMounts, 3)
		assert.Equal(t, "volume-name", wait.VolumeMounts[0].Name)
		assert.Equal(t, "tmp-dir-argo", wait.VolumeMounts[1].Name)
		assert.Equal(t, "var-run-argo", wait.VolumeMounts[2].Name)
		main := containers[1]
		cmd := append(append(emissaryCmd, woc.getExecutorLogOpts(ctx)...), "--", "cowsay")
		assert.Equal(t, cmd, main.Command)
		require.Len(t, main.VolumeMounts, 2)
		assert.Equal(t, "volume-name", main.VolumeMounts[0].Name)
		assert.Equal(t, "var-run-argo", main.VolumeMounts[1].Name)
	})
}

func TestVolumesPodSubstitution(t *testing.T) {
	volumes := []apiv1.Volume{
		{
			Name: "volume-name",
			VolumeSource: apiv1.VolumeSource{
				PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "{{inputs.parameters.volume-name}}",
				},
			},
		},
	}
	volumeMounts := []apiv1.VolumeMount{
		{
			Name:      "volume-name",
			MountPath: "/test",
		},
	}
	inputParameters := []wfv1.Parameter{
		{
			Name:  "volume-name",
			Value: wfv1.AnyStringPtr("test-name"),
		},
	}

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.volumes = volumes
	woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.execWf.Spec.Templates[0].Inputs.Parameters = inputParameters

	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Len(t, pod.Spec.Volumes, 3)
	assert.Equal(t, "volume-name", pod.Spec.Volumes[2].Name)
	assert.Equal(t, "test-name", pod.Spec.Volumes[2].PersistentVolumeClaim.ClaimName)
	assert.Len(t, pod.Spec.Containers[1].VolumeMounts, 2)
	assert.Equal(t, "volume-name", pod.Spec.Containers[0].VolumeMounts[0].Name)
}

func TestOutOfCluster(t *testing.T) {
	verifyKubeConfigVolume := func(ctr apiv1.Container, volName, mountPath string) {
		for _, vol := range ctr.VolumeMounts {
			if vol.Name == volName && vol.MountPath == mountPath {
				for _, arg := range ctr.Args {
					if arg == fmt.Sprintf("--kubeconfig=%s", mountPath) {
						return
					}
				}
			}
		}
		t.Fatalf("%v does not have kubeconfig mounted properly (name: %s, mountPath: %s)", ctr, volName, mountPath)
	}

	// default mount path & volume name
	{
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		woc.controller.Config.KubeConfig = &config.KubeConfig{
			SecretName: "foo",
			SecretKey:  "bar",
		}

		tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
		require.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		require.NoError(t, err)
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		assert.Equal(t, "kubeconfig", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "foo", pod.Spec.Volumes[0].Secret.SecretName)

		waitCtr := pod.Spec.Containers[0]
		verifyKubeConfigVolume(waitCtr, "kubeconfig", "/kube/config")
	}

	// custom mount path & volume name, in case name collision
	{
		ctx := logging.TestContext(t.Context())
		woc := newWoc(ctx)
		woc.controller.Config.KubeConfig = &config.KubeConfig{
			SecretName: "foo",
			SecretKey:  "bar",
			MountPath:  "/some/path/config",
			VolumeName: "kube-config-secret",
		}

		tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
		require.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		require.NoError(t, err)
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		assert.Equal(t, "kube-config-secret", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "foo", pod.Spec.Volumes[0].Secret.SecretName)

		// kubeconfig volume is the last one
		waitCtr := pod.Spec.Containers[0]
		verifyKubeConfigVolume(waitCtr, "kube-config-secret", "/some/path/config")
	}
}

// TestPriorityClass verifies the ability to carry forward priorityClassName
func TestPriorityClass(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.Templates[0].PriorityClassName = "foo"
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "foo", pod.Spec.PriorityClassName)
}

// TestSchedulerName verifies the ability to carry forward schedulerName.
func TestSchedulerName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.Templates[0].SchedulerName = "foo"
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "foo", pod.Spec.SchedulerName)
}

// TestInitContainers verifies the ability to set up initContainers
func TestInitContainers(t *testing.T) {
	volumes := []apiv1.Volume{
		{
			Name: "volume-name",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "init-volume-name",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	volumeMounts := []apiv1.VolumeMount{
		{
			Name:      "volume-name",
			MountPath: "/test",
		},
	}
	initVolumeMounts := []apiv1.VolumeMount{
		{
			Name:      "init-volume-name",
			MountPath: "/init-test",
		},
	}
	mirrorVolumeMounts := true

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.volumes = volumes
	woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.execWf.Spec.Templates[0].InitContainers = []wfv1.UserContainer{
		{
			MirrorVolumeMounts: &mirrorVolumeMounts,
			Container: apiv1.Container{
				Name:         "init-foo",
				VolumeMounts: initVolumeMounts,
			},
		},
	}

	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Len(t, pod.Spec.InitContainers, 2)
	foo := pod.Spec.InitContainers[1]
	assert.Equal(t, "init-foo", foo.Name)
	for _, v := range volumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
	assert.Len(t, foo.VolumeMounts, 3)
	assert.Equal(t, "init-volume-name", foo.VolumeMounts[0].Name)
	assert.Equal(t, "volume-name", foo.VolumeMounts[1].Name)
	assert.Equal(t, "var-run-argo", foo.VolumeMounts[2].Name)
}

// TestSidecars verifies the ability to set up sidecars
func TestSidecars(t *testing.T) {
	volumes := []apiv1.Volume{
		{
			Name: "volume-name",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "sidecar-volume-name",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	volumeMounts := []apiv1.VolumeMount{
		{
			Name:      "volume-name",
			MountPath: "/test",
		},
	}
	sidecarVolumeMounts := []apiv1.VolumeMount{
		{
			Name:      "sidecar-volume-name",
			MountPath: "/sidecar-test",
		},
	}
	mirrorVolumeMounts := true

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.volumes = volumes
	woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.execWf.Spec.Templates[0].Sidecars = []wfv1.UserContainer{
		{
			MirrorVolumeMounts: &mirrorVolumeMounts,
			Container: apiv1.Container{
				Name:         "side-foo",
				VolumeMounts: sidecarVolumeMounts,
				Image:        "argoproj/argosay:v2",
			},
		},
	}

	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Len(t, pod.Spec.Containers, 3)
	assert.Equal(t, "wait", pod.Spec.Containers[0].Name)
	assert.Equal(t, "main", pod.Spec.Containers[1].Name)
	assert.Equal(t, "side-foo", pod.Spec.Containers[2].Name)
	for _, v := range volumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
	assert.Len(t, pod.Spec.Containers[2].VolumeMounts, 3)
	assert.Equal(t, "sidecar-volume-name", pod.Spec.Containers[2].VolumeMounts[0].Name)
	assert.Equal(t, "volume-name", pod.Spec.Containers[2].VolumeMounts[1].Name)
}

func TestTemplateLocalVolumes(t *testing.T) {
	volumes := []apiv1.Volume{
		{
			Name: "volume-name",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	localVolumes := []apiv1.Volume{
		{
			Name: "local-volume-name",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	volumeMounts := []apiv1.VolumeMount{
		{
			Name:      "volume-name",
			MountPath: "/test",
		},
		{
			Name:      "local-volume-name",
			MountPath: "/local-test",
		},
	}

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.volumes = volumes
	woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.execWf.Spec.Templates[0].Volumes = localVolumes

	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	for _, v := range volumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
	for _, v := range localVolumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
}

// TestWFLevelHostAliases verifies the ability to carry forward workflow level HostAliases to Podspec
func TestWFLevelHostAliases(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.HostAliases = []apiv1.HostAlias{
		{IP: "127.0.0.1"},
		{IP: "127.0.0.1"},
	}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.HostAliases)
}

// TestTmplLevelHostAliases verifies the ability to carry forward template level HostAliases to Podspec
func TestTmplLevelHostAliases(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.execWf.Spec.Templates[0].HostAliases = []apiv1.HostAlias{
		{IP: "127.0.0.1"},
		{IP: "127.0.0.1"},
	}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.HostAliases)
}

// TestWFLevelSecurityContext verifies the ability to carry forward workflow level SecurityContext to Podspec
func TestWFLevelSecurityContext(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	runAsUser := int64(1234)
	woc.execWf.Spec.SecurityContext = &apiv1.PodSecurityContext{
		RunAsUser: &runAsUser,
	}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.SecurityContext)
	assert.Equal(t, runAsUser, *pod.Spec.SecurityContext.RunAsUser)
}

// TestTmplLevelSecurityContext verifies the ability to carry forward template level SecurityContext to Podspec
func TestTmplLevelSecurityContext(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	runAsUser := int64(1234)
	woc.execWf.Spec.Templates[0].SecurityContext = &apiv1.PodSecurityContext{
		RunAsUser: &runAsUser,
	}
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	require.NoError(t, err)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.SecurityContext)
	assert.Equal(t, runAsUser, *pod.Spec.SecurityContext.RunAsUser)
}

func Test_createSecretVolumesFromArtifactLocations_SSECUsed(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	cancel, controller := newControllerWithComplexDefaults(ctx)
	defer cancel()

	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	wf.Spec.Templates[0].Inputs = wfv1.Inputs{
		Artifacts: []wfv1.Artifact{
			{
				Name: "foo",
				Path: "/tmp/file",
				ArtifactLocation: wfv1.ArtifactLocation{
					S3: &wfv1.S3Artifact{
						Key: "/foo/key",
					},
				},
				Archive: &wfv1.ArchiveStrategy{
					None: &wfv1.NoneStrategy{},
				},
			},
		},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	setArtifactRepository(woc.controller,
		&wfv1.ArtifactRepository{
			S3: &wfv1.S3ArtifactRepository{
				S3Bucket: wfv1.S3Bucket{
					Bucket: "foo",
					AccessKeySecret: &apiv1.SecretKeySelector{
						LocalObjectReference: apiv1.LocalObjectReference{
							Name: "accesskey",
						},
						Key: "aws-keys",
					},
					SecretKeySecret: &apiv1.SecretKeySelector{
						LocalObjectReference: apiv1.LocalObjectReference{
							Name: "secretkey",
						},
						Key: "aws-keys",
					},
					EncryptionOptions: &wfv1.S3EncryptionOptions{
						EnableEncryption: true,
						ServerSideCustomerKeySecret: &apiv1.SecretKeySelector{
							LocalObjectReference: apiv1.LocalObjectReference{
								Name: "enckey",
							},
							Key: "aws-sse-c",
						},
					},
				},
			},
		},
	)

	wantVolume := apiv1.Volume{
		Name: "enckey",
		VolumeSource: apiv1.VolumeSource{
			Secret: &apiv1.SecretVolumeSource{
				SecretName: "enckey",
				Items: []apiv1.KeyToPath{
					{
						Key:  "aws-sse-c",
						Path: "aws-sse-c",
					},
				},
			},
		},
	}
	wantInitContainerVolumeMount := apiv1.VolumeMount{
		Name:      "enckey",
		ReadOnly:  true,
		MountPath: path.Join(common.SecretVolMountPath, "enckey"),
	}

	err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	woc.operate(ctx)

	mainCtr := woc.execWf.Spec.Templates[0].Container
	for i := 1; i < 5; i++ {
		pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		if pod != nil {
			assert.Contains(t, pod.Spec.Volumes, wantVolume)
			assert.Len(t, pod.Spec.InitContainers, 1)
			assert.Contains(t, pod.Spec.InitContainers[0].VolumeMounts, wantInitContainerVolumeMount)
			break
		}
	}
}

func TestCreateSecretVolumesFromArtifactLocationsSessionToken(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	cancel, controller := newControllerWithComplexDefaults(ctx)
	defer cancel()

	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	wf.Spec.Templates[0].Inputs = wfv1.Inputs{
		Artifacts: []wfv1.Artifact{
			{
				Name: "foo",
				Path: "/tmp/file",
				ArtifactLocation: wfv1.ArtifactLocation{
					S3: &wfv1.S3Artifact{
						Key: "/foo/key",
					},
				},
				Archive: &wfv1.ArchiveStrategy{
					None: &wfv1.NoneStrategy{},
				},
			},
		},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	setArtifactRepository(woc.controller,
		&wfv1.ArtifactRepository{
			S3: &wfv1.S3ArtifactRepository{
				S3Bucket: wfv1.S3Bucket{
					Bucket: "foo",
					AccessKeySecret: &apiv1.SecretKeySelector{
						LocalObjectReference: apiv1.LocalObjectReference{
							Name: "accesskey",
						},
						Key: "access-key",
					},
					SecretKeySecret: &apiv1.SecretKeySelector{
						LocalObjectReference: apiv1.LocalObjectReference{
							Name: "secretkey",
						},
						Key: "secret-key",
					},
					SessionTokenSecret: &apiv1.SecretKeySelector{
						LocalObjectReference: apiv1.LocalObjectReference{
							Name: "sessiontoken",
						},
						Key: "session-token",
					},
				},
			},
		},
	)

	wantedKeysVolume := apiv1.Volume{
		Name: "sessiontoken",
		VolumeSource: apiv1.VolumeSource{
			Secret: &apiv1.SecretVolumeSource{
				SecretName: "sessiontoken",
				Items: []apiv1.KeyToPath{
					{
						Key:  "session-token",
						Path: "session-token",
					},
				},
			},
		},
	}
	wantedInitContainerVolumeMount := apiv1.VolumeMount{
		Name:      "sessiontoken",
		ReadOnly:  true,
		MountPath: path.Join(common.SecretVolMountPath, "sessiontoken"),
	}

	err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	woc.operate(ctx)

	mainCtr := woc.execWf.Spec.Templates[0].Container
	for i := 1; i < 5; i++ {
		pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		if pod != nil {
			assert.Contains(t, pod.Spec.Volumes, wantedKeysVolume)
			assert.Len(t, pod.Spec.InitContainers, 1)
			assert.Contains(t, pod.Spec.InitContainers[0].VolumeMounts, wantedInitContainerVolumeMount)
			break
		}
	}
}

var helloWorldWfWithPodSpecPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    podSpecPatch: '{"containers":[{"name":"main", "resources":{"limits":{"cpu": "800m"}}}]}'
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
    outputs:
      parameters:
      - name: pod-name
        value: "{{pod.name}}"
`

var helloWorldWfWithPodMetadataPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint:
  templates:
  - name: whalesay
    podMetadataPatch: '{"annotations": {"test-annotation": "value"},"labels": {"test-label": "value"}}'
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithWFPodSpecPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  podSpecPatch: '{"containers":[{"name":"main", "resources":{"limits":{"cpu": "800m"}}}]}'
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithWFPodMetadataPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  podMetadataPatch: '{"annotations": {"test-annotation": "value"},"labels": {"test-label": "value"}}'
  templates:
  - name: whalesay
    container: 
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithWFYAMLPodSpecPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  podSpecPatch: |
    containers:
      - name: main
        resources:
          limits:
            cpu: "800m"
  templates:
  - name: whalesay
    podSpecPatch: '{"containers":[{"name":"main", "resources":{"limits":{"memory": "100Mi"}}}]}'
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithWFYAMLPodMetadataPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    podMetadataPatch: '{"annotations": {"test-annotation": "value"},"labels": {"test-label": "value"}}'
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithTmplAndWFPodSpecPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  podSpecPatch: |
    containers:
      - name: main
        securityContext:
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
  templates:
  - name: whalesay
    podSpecPatch: '{"containers":[{"name":"main", "securityContext":{"capabilities":{"add":["ALL"],"drop":null}}}]}'
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithTmplAndWFPodMetadataPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    podMetadataPatch: '{"annotations": {"test-annotation": "value"},"labels": {"test-label": "value"}}'
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithInvalidPodSpecPatchFormat = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    podSpecPatch: '{"containers"}' # not a valid JSON here
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var helloWorldWfWithInvalidPodMetadataPatchFormat = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    podMetadataPatch: '{"annotations"}' # not a valid JSON here
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func TestPodMetadataPatch(t *testing.T) {
	// validate pod metadata with template-level priority
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfWithPodMetadataPatch)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "value", pod.Annotations["test-annotation"])
	assert.Equal(t, "value", pod.Labels["test-label"])

	// validate pod metadata with workflow-level priority
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithWFPodMetadataPatch)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "value", pod.Annotations["test-annotation"])
	assert.Equal(t, "value", pod.Labels["test-label"])

	// validate pod metadata set at the template level in YAML format (as opposed to string json)
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithWFYAMLPodMetadataPatch)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "value", pod.Annotations["test-annotation"])
	assert.Equal(t, "value", pod.Labels["test-label"])

	// validate pod metadata with both template and workflow-level priority (should use two separate and an override, check all)
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithTmplAndWFPodMetadataPatch)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "value", pod.Annotations["test-annotation"])
	assert.Equal(t, "value", pod.Labels["test-label"])

	// validate error is thrown when invalid pod metadata patch is set.
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithInvalidPodMetadataPatchFormat)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	_, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	require.EqualError(t, err, "Error applying PodMetadataPatch")
	require.EqualError(t, errors.Cause(err), "invalid character '}' after object key")
}

func TestPodSpecPatch(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfWithPodSpecPatch)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithWFPodSpecPatch)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithWFYAMLPodSpecPatch)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "104857600", pod.Spec.Containers[1].Resources.Limits.Memory().AsDec().String())

	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithTmplAndWFPodSpecPatch)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, ptr.To(true), pod.Spec.Containers[1].SecurityContext.RunAsNonRoot)
	assert.Equal(t, apiv1.Capability("ALL"), pod.Spec.Containers[1].SecurityContext.Capabilities.Add[0])
	assert.Equal(t, []apiv1.Capability(nil), pod.Spec.Containers[1].SecurityContext.Capabilities.Drop)

	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithInvalidPodSpecPatchFormat)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	_, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	require.EqualError(t, err, "Error applying PodSpecPatch")
	require.EqualError(t, errors.Cause(err), "invalid character '}' after object key")
}

var helloWorldStepWfWithPatch = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: hello
  templates:
  - name: hello
    steps:
    - - name: hello
        template: whalesay
  - name: whalesay
    podSpecPatch: '{"containers":[{"name":"main", "resources":{"limits":{"cpu": "800m"}}}]}'
    podMetadataPatch: '{"annotations": {"test-annotation": "value"},"labels": {"test-label": "value"}}'
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
    outputs:
      parameters:
      - name: pod-name
        value: "{{pod.name}}"
`

func TestPodSpecPatchPodName(t *testing.T) {
	tests := []struct {
		podNameVersion string
		wantPodName    string
		workflowYaml   string
	}{
		{"v1", "hello-world", helloWorldWfWithPodSpecPatch},
		{"v2", "hello-world", helloWorldWfWithPodSpecPatch},
		{"v1", "hello-world-3731220306", helloWorldStepWfWithPatch},
		{"v2", "hello-world-whalesay-3731220306", helloWorldStepWfWithPatch},
	}
	for _, tt := range tests {
		t.Setenv("POD_NAMES", tt.podNameVersion)
		ctx := logging.TestContext(t.Context())
		wf := wfv1.MustUnmarshalWorkflow(tt.workflowYaml)
		woc := newWoc(ctx, *wf)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items, "pod was not created successfully")
		template, err := getPodTemplate(&pods.Items[0])
		require.NoError(t, err)
		parameterValue := template.Outputs.Parameters[0].Value
		assert.NotNil(t, parameterValue)
		assert.Equal(t, tt.wantPodName, parameterValue.String())
	}
}

func TestMainContainerCustomization(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mainCtrSpec := &apiv1.Container{
		Name:            common.MainContainerName,
		SecurityContext: &apiv1.SecurityContext{},
		Resources: apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.200"),
				apiv1.ResourceMemory: resource.MustParse("512Mi"),
			},
		},
	}
	// podSpecPatch in workflow spec takes precedence over the main container's
	// configuration in controller so here we respect what's specified in podSpecPatch.
	t.Run("PodSpecPatchPrecedence", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(helloWorldWfWithPodSpecPatch)
		woc := newWoc(ctx, *wf)
		woc.controller.Config.MainContainer = mainCtrSpec
		mainCtr := woc.execWf.Spec.Templates[0].Container
		pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())
	})
	// The main container's resources should be changed since the existing
	// container's resources are not specified.
	t.Run("Default", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
		woc := newWoc(ctx, *wf)
		woc.controller.Config.MainContainer = mainCtrSpec
		mainCtr := woc.execWf.Spec.Templates[0].Container
		mainCtr.Resources = apiv1.ResourceRequirements{Limits: apiv1.ResourceList{}}
		pod, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		require.NoError(t, err)
		ctr := pod.Spec.Containers[1]
		assert.NotNil(t, ctr.SecurityContext)
		require.NotNil(t, pod.Spec.Containers[1].Resources)
		assert.Equal(t, "0.200", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())
	})

	// Workflow spec's main container takes precedence over config in controller
	// so here the main container resources remain unchanged.
	t.Run("ContainerPrecedence", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
		woc := newWoc(ctx, *wf)
		woc.controller.Config.MainContainer = mainCtrSpec
		mainCtr := wf.Spec.Templates[0].Container
		mainCtr.Name = common.MainContainerName
		mainCtr.Resources = apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.900"),
				apiv1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
		pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		assert.Equal(t, "0.900", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	})
	// If script template has limits then they take precedence over config in controller
	t.Run("ScriptPrecedence", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(scriptWf)
		woc := newWoc(ctx, *wf)
		woc.controller.Config.MainContainer = mainCtrSpec
		mainCtr := &woc.execWf.Spec.Templates[0].Script.Container
		mainCtr.Resources = apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("1"),
				apiv1.ResourceMemory: resource.MustParse("123Mi"),
			},
		}
		pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		assert.Equal(t, "1", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())
		assert.Equal(t, "128974848", pod.Spec.Containers[1].Resources.Limits.Memory().AsDec().String())
	})
}

func TestExecutorContainerCustomization(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	woc.controller.Config.Executor = &apiv1.Container{
		Args: []string{"foo"},
		Resources: apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.900"),
				apiv1.ResourceMemory: resource.MustParse("512Mi"),
			},
		},
	}

	pod, err := woc.createWorkflowPod(ctx, "", nil, &wfv1.Template{}, &createWorkflowPodOpts{})
	require.NoError(t, err)
	waitCtr := pod.Spec.Containers[0]
	assert.Equal(t, []string{"foo"}, waitCtr.Args)
	assert.Equal(t, "0.900", waitCtr.Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "536870912", waitCtr.Resources.Limits.Memory().AsDec().String())
}

var helloWindowsWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-hybrid-win
spec:
  entrypoint: hello-win
  templates:
    - name: hello-win
      nodeSelector:
        kubernetes.io/os: windows
      container:
        image: mcr.microsoft.com/windows/nanoserver:1809
        command: ["cmd", "/c"]
        args: ["echo", "Hello from Windows Container!"]
`

func TestWindowsUNCPathsAreRemoved(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWindowsWf)
	ctx := logging.TestContext(t.Context())
	uncVolume := apiv1.Volume{
		Name: "unc",
		VolumeSource: apiv1.VolumeSource{
			HostPath: &apiv1.HostPathVolumeSource{
				Path: "\\\\.\\pipe\\test",
			},
		},
	}
	uncMount := apiv1.VolumeMount{
		Name:      "unc",
		MountPath: "\\\\.\\pipe\\test",
	}

	// Add artifacts so that initContainer volumeMount logic is run
	inp := wfv1.Artifact{
		Name: "kubectl",
		Path: "C:\\kubectl",
		ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{
			URL: "https://dl.k8s.io/release/v1.22.0/bin/windows/amd64/kubectl.exe"},
		},
	}
	wf.Spec.Volumes = append(wf.Spec.Volumes, uncVolume)
	wf.Spec.Templates[0].Container.VolumeMounts = append(wf.Spec.Templates[0].Container.VolumeMounts, uncMount)
	wf.Spec.Templates[0].Inputs.Artifacts = append(wf.Spec.Templates[0].Inputs.Artifacts, inp)
	woc := newWoc(ctx, *wf)

	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	waitCtrIdx, err := wfutil.FindWaitCtrIndex(pod)

	if err != nil {
		require.Errorf(t, err, "could not find wait ctr index")
	}
	for _, mnt := range pod.Spec.Containers[waitCtrIdx].VolumeMounts {
		assert.NotEqual(t, "unc", mnt.Name)
	}
	for _, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			for _, mnt := range initCtr.VolumeMounts {
				assert.NotEqual(t, "unc", mnt.Name)
			}
		}
	}
}

var propagateMaxDuration = `
name: retry-backoff
retryStrategy:
  limit: 10
  backoff:
    duration: "1"
    factor: 1
    maxDuration: "20"
container:
  image: alpine
  command: [sh, -c]
  args: ["sleep $(( {{retries}} * 100 )); exit 1"]

`

func TestPropagateMaxDuration(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// Ensure that volume mount is added when artifact is provided
	tmpl := unmarshalTemplate(propagateMaxDuration)
	woc := newWoc(ctx)
	deadline := time.Time{}.Add(time.Second)
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{*tmpl.Container}, tmpl, &createWorkflowPodOpts{executionDeadline: deadline})
	require.NoError(t, err)
	v, err := getPodDeadline(pod)
	require.NoError(t, err)
	assert.Equal(t, v, deadline)
}

var wfWithPodMetadata = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  podMetadata:
    annotations:
      workflow-level-pod-annotation: foo
    labels:
      workflow-level-pod-label: bar
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var wfWithPodMetadataAndTemplateMetadata = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  podMetadata:
    annotations:
      workflow-level-pod-annotation: foo
    labels:
      workflow-level-pod-label: bar
  templates:
  - name: whalesay
    metadata:
      annotations:
        workflow-level-pod-annotation: fizz
        template-level-pod-annotation: hello
      labels:
        workflow-level-pod-label: buzz
        template-level-pod-label: world
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func TestPodMetadata(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithPodMetadata)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "foo", pod.Annotations["workflow-level-pod-annotation"])
	assert.Equal(t, "bar", pod.Labels["workflow-level-pod-label"])

	wf = wfv1.MustUnmarshalWorkflow(wfWithPodMetadataAndTemplateMetadata)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "fizz", pod.Annotations["workflow-level-pod-annotation"])
	assert.Equal(t, "buzz", pod.Labels["workflow-level-pod-label"])
	assert.Equal(t, "hello", pod.Annotations["template-level-pod-annotation"])
	assert.Equal(t, "world", pod.Labels["template-level-pod-label"])
}

var wfWithContainerSet = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-with-container-set
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    containerSet:
      containers:
        - name: a
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["hello world"]
        - name: b
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["hello world"]
`

func TestPodDefaultContainer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(wfWithContainerSet)
	// change first container name to main
	wf.Spec.Templates[0].ContainerSet.Containers[0].Name = common.MainContainerName
	woc := newWoc(ctx, *wf)
	template := woc.execWf.Spec.Templates[0]
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, template.ContainerSet.GetContainers(), &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, common.MainContainerName, pod.Annotations[common.AnnotationKeyDefaultContainer])

	wf = wfv1.MustUnmarshalWorkflow(wfWithContainerSet)
	woc = newWoc(ctx, *wf)
	template = woc.execWf.Spec.Templates[0]
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, template.ContainerSet.GetContainers(), &template, &createWorkflowPodOpts{})
	assert.Equal(t, "b", pod.Annotations[common.AnnotationKeyDefaultContainer])
}

func TestGetDeadline(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	deadline, _ := getPodDeadline(pod)
	assert.Equal(t, time.Time{}, deadline)

	executionDeadline := time.Now().Add(5 * time.Minute)
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc = newWoc(ctx, *wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{executionDeadline: executionDeadline})
	deadline, _ = getPodDeadline(pod)
	assert.Equal(t, executionDeadline.Format(time.RFC3339), deadline.Format(time.RFC3339))
}

func TestPodMetadataWithWorkflowDefaults(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	wfDefaultAnnotations := make(map[string]string)
	wfDefaultAnnotations["controller-level-pod-annotation"] = "annotation-value"
	wfDefaultAnnotations["workflow-level-pod-annotation"] = "set-by-controller"
	wfDefaultLabels := make(map[string]string)
	wfDefaultLabels["controller-level-pod-label"] = "label-value"
	wfDefaultLabels["workflow-level-pod-label"] = "set-by-controller"
	controller.Config.WorkflowDefaults = &wfv1.Workflow{
		Spec: wfv1.WorkflowSpec{
			PodMetadata: &wfv1.Metadata{
				Annotations: wfDefaultAnnotations,
				Labels:      wfDefaultLabels,
			},
		},
	}

	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "annotation-value", pod.Annotations["controller-level-pod-annotation"])
	assert.Equal(t, "set-by-controller", pod.Annotations["workflow-level-pod-annotation"])
	assert.Equal(t, "label-value", pod.Labels["controller-level-pod-label"])
	assert.Equal(t, "set-by-controller", pod.Labels["workflow-level-pod-label"])
	cancel() // need to cancel to spin up pods with the same name

	cancel, controller = newController(ctx)
	defer cancel()
	controller.Config.WorkflowDefaults = &wfv1.Workflow{
		Spec: wfv1.WorkflowSpec{
			PodMetadata: &wfv1.Metadata{
				Annotations: wfDefaultAnnotations,
				Labels:      wfDefaultLabels,
			},
		},
	}
	wf = wfv1.MustUnmarshalWorkflow(wfWithPodMetadata)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	err = woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "foo", pod.Annotations["workflow-level-pod-annotation"])
	assert.Equal(t, "bar", pod.Labels["workflow-level-pod-label"])
	assert.Equal(t, "annotation-value", pod.Annotations["controller-level-pod-annotation"])
	assert.Equal(t, "label-value", pod.Labels["controller-level-pod-label"])
	cancel()
}

func TestPodExists(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	require.NoError(t, err)
	assert.NotNil(t, pod)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	// Sleep 1 second to wait for informer getting pod info
	time.Sleep(time.Second)
	existingPod, doesExist, err := woc.podExists(pod.Name)
	require.NoError(t, err)
	assert.NotNil(t, existingPod)
	assert.True(t, doesExist)
	assert.Equal(t, pod, existingPod)
}

func TestPodFinalizerExits(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	require.NoError(t, err)
	assert.NotNil(t, pod)

	assert.Equal(t, []string{common.FinalizerPodStatus}, pod.GetFinalizers())
}

func TestPodFinalizerDoesNotExist(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	require.NoError(t, err)
	assert.NotNil(t, pod)

	assert.Equal(t, []string(nil), pod.GetFinalizers())
}

func TestProgressEnvVars(t *testing.T) {
	setup := func(t *testing.T, options ...interface{}) (context.CancelFunc, *apiv1.Pod) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, options...)

		wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err := woc.setExecWorkflow(ctx)
		require.NoError(t, err)
		mainCtr := woc.execWf.Spec.Templates[0].Container
		pod, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		require.NoError(t, err)
		assert.NotNil(t, pod)
		return cancel, pod
	}

	t.Run("default settings use self reporting progress with defaults", func(t *testing.T) {
		cancel, pod := setup(t)
		defer cancel()

		assert.Contains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFile,
			Value: common.ArgoProgressPath,
		})
		assert.Contains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressPatchTickDuration,
			Value: "1m0s",
		})
		assert.Contains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFileTickDuration,
			Value: "3s",
		})
	})

	t.Run("setting patch tick duration to 0 disables self reporting progress.", func(t *testing.T) {
		cancel, pod := setup(t, func(workflowController *WorkflowController) {
			workflowController.progressPatchTickDuration = 0
		})
		defer cancel()

		assert.NotContains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFile,
			Value: common.ArgoProgressPath,
		})
		assert.NotContains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressPatchTickDuration,
			Value: "1m0s",
		})
		assert.NotContains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFileTickDuration,
			Value: "3s",
		})
	})

	t.Run("setting read file tick duration to 0 disables self reporting progress.", func(t *testing.T) {
		cancel, pod := setup(t, func(workflowController *WorkflowController) {
			workflowController.progressFileTickDuration = 0
		})
		defer cancel()

		assert.NotContains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFile,
			Value: common.ArgoProgressPath,
		})
		assert.NotContains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressPatchTickDuration,
			Value: "1m0s",
		})
		assert.NotContains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFileTickDuration,
			Value: "3s",
		})
	})

	t.Run("tick durations are configurable", func(t *testing.T) {
		cancel, pod := setup(t, func(workflowController *WorkflowController) {
			workflowController.progressPatchTickDuration = 30 * time.Second
			workflowController.progressFileTickDuration = 1 * time.Second
		})
		defer cancel()

		assert.Contains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFile,
			Value: common.ArgoProgressPath,
		})
		assert.Contains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressPatchTickDuration,
			Value: "30s",
		})
		assert.Contains(t, pod.Spec.Containers[0].Env, apiv1.EnvVar{
			Name:  common.EnvVarProgressFileTickDuration,
			Value: "1s",
		})
	})
}

var helloWorldWfWithEnvReferSecret = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metadata:
      annotations:
        annotationKey1: "annotationValue1"
        annotationKey2: "annotationValue2"
      labels:
        labelKey1: "labelValue1"
        labelKey2: "labelValue2"
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
      env:
      - name: ENV3
        valueFrom:
          secretKeyRef:
            name: mysecret
            key: sec
`

func TestMergeEnvVars(t *testing.T) {
	setup := func(t *testing.T, options ...interface{}) (context.CancelFunc, *apiv1.Pod) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, options...)

		wf := wfv1.MustUnmarshalWorkflow(helloWorldWfWithEnvReferSecret)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err := woc.setExecWorkflow(ctx)
		require.NoError(t, err)
		mainCtrSpec := &apiv1.Container{
			Name:            common.MainContainerName,
			SecurityContext: &apiv1.SecurityContext{},
			Env: []apiv1.EnvVar{
				{
					Name:  "ENV1",
					Value: "env1",
				},
				{
					Name:  "ENV2",
					Value: "env2",
				},
			},
		}
		woc.controller.Config.MainContainer = mainCtrSpec
		mainCtr := woc.execWf.Spec.Templates[0].Container

		pod, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
		require.NoError(t, err)
		assert.NotNil(t, pod)
		return cancel, pod
	}

	t.Run("test merge envs", func(t *testing.T) {
		cancel, pod := setup(t)
		defer cancel()
		assert.Contains(t, pod.Spec.Containers[1].Env, apiv1.EnvVar{
			Name:  "ENV1",
			Value: "env1",
		})
		assert.Contains(t, pod.Spec.Containers[1].Env, apiv1.EnvVar{
			Name:  "ENV2",
			Value: "env2",
		})
		assert.Contains(t, pod.Spec.Containers[1].Env, apiv1.EnvVar{
			Name: "ENV3",
			ValueFrom: &apiv1.EnvVarSource{
				SecretKeyRef: &apiv1.SecretKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: "mysecret",
					},
					Key: "sec",
				},
			},
		})
	})
}
