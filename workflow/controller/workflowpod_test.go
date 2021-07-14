package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/util"
	armocks "github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories/mocks"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// Deprecated
func unmarshalTemplate(yamlStr string) *wfv1.Template {
	return wfv1.MustUnmarshalTemplate(yamlStr)
}

// newWoc a new operation context suitable for testing
func newWoc(wfs ...wfv1.Workflow) *wfOperationCtx {
	var wf *wfv1.Workflow
	if len(wfs) == 0 {
		wf = wfv1.MustUnmarshalWorkflow(helloWorldWf)
	} else {
		wf = &wfs[0]
	}
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)
	return woc
}

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
	ctx := context.Background()
	tmpl := unmarshalTemplate(scriptTemplateWithInputArtifact)
	woc := newWoc()
	_, err := woc.executeScript(ctx, tmpl.Name, "", tmpl, &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
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

// TestScriptTemplateWithVolume ensure we can a script pod with input artifacts
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
	woc := newWoc()
	mainCtr := tmpl.Script.Container
	mainCtr.Args = append(mainCtr.Args, common.ExecutorScriptSourcePath)
	ctx := context.Background()
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{})
	assert.NoError(t, err)
	// Note: pod.Spec.Containers[0] is wait
	assert.Contains(t, pod.Spec.Containers[1].VolumeMounts, volumeMount)
	assert.NotContains(t, pod.Spec.Containers[1].VolumeMounts, customVolumeMount)
	assert.NotContains(t, pod.Spec.InitContainers[0].VolumeMounts, customVolumeMountForInit)

	// Ensure that volume mount is added to initContainer when artifact is provided
	// and the volume is mounted manually in the template
	tmpl = unmarshalTemplate(scriptTemplateWithOptionalInputArtifactProvidedAndOverlappedPath)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	wf.Spec.Volumes = append(wf.Spec.Volumes, apiv1.Volume{Name: "my-mount"})
	woc = newWoc(*wf)
	mainCtr = tmpl.Script.Container
	mainCtr.Args = append(mainCtr.Args, common.ExecutorScriptSourcePath)
	pod, err = woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{mainCtr}, tmpl, &createWorkflowPodOpts{includeScriptOutput: true})
	assert.NoError(t, err)
	assert.NotContains(t, pod.Spec.Containers[1].VolumeMounts, volumeMount)
	assert.Contains(t, pod.Spec.Containers[1].VolumeMounts, customVolumeMount)
	assert.Contains(t, pod.Spec.InitContainers[0].VolumeMounts, customVolumeMountForInit)
}

// TestWFLevelServiceAccount verifies the ability to carry forward the service account name
// for the pod from workflow.spec.serviceAccountName.
func TestWFLevelServiceAccount(t *testing.T) {
	woc := newWoc()
	woc.execWf.Spec.ServiceAccountName = "foo"
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, pod.Spec.ServiceAccountName, "foo")
}

// TestTmplServiceAccount verifies the ability to carry forward the Template level service account name
// for the pod from workflow.spec.serviceAccountName.
func TestTmplServiceAccount(t *testing.T) {
	woc := newWoc()
	woc.execWf.Spec.ServiceAccountName = "foo"
	woc.execWf.Spec.Templates[0].ServiceAccountName = "tmpl"
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)

	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, pod.Spec.ServiceAccountName, "tmpl")
}

// TestWFLevelAutomountServiceAccountToken verifies the ability to carry forward workflow level AutomountServiceAccountToken to Podspec.
func TestWFLevelAutomountServiceAccountToken(t *testing.T) {
	woc := newWoc()
	ctx := context.Background()
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo", "foo-token")
	assert.NoError(t, err)

	falseValue := false
	woc.execWf.Spec.AutomountServiceAccountToken = &falseValue
	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, *pod.Spec.AutomountServiceAccountToken, false)
}

// TestTmplLevelAutomountServiceAccountToken verifies the ability to carry forward template level AutomountServiceAccountToken to Podspec.
func TestTmplLevelAutomountServiceAccountToken(t *testing.T) {
	woc := newWoc()
	ctx := context.Background()
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo", "foo-token")
	assert.NoError(t, err)

	trueValue := true
	falseValue := false
	woc.execWf.Spec.AutomountServiceAccountToken = &trueValue
	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	woc.execWf.Spec.Templates[0].AutomountServiceAccountToken = &falseValue
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, *pod.Spec.AutomountServiceAccountToken, false)
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
	woc := newWoc()
	ctx := context.Background()
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo", "foo-token")
	assert.NoError(t, err)

	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "exec-sa-token", pod.Spec.Volumes[1].Name)
	assert.Equal(t, "foo-token", pod.Spec.Volumes[1].VolumeSource.Secret.SecretName)

	waitCtr := pod.Spec.Containers[0]
	verifyServiceAccountTokenVolumeMount(t, waitCtr, "exec-sa-token", "/var/run/secrets/kubernetes.io/serviceaccount")
}

// TestTmplLevelExecutorServiceAccountName verifies the ability to carry forward template level AutomountServiceAccountToken to Podspec.
func TestTmplLevelExecutorServiceAccountName(t *testing.T) {
	woc := newWoc()
	ctx := context.Background()
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo", "foo-token")
	assert.NoError(t, err)
	_, err = util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "tmpl", "tmpl-token")
	assert.NoError(t, err)

	woc.execWf.Spec.Executor = &wfv1.ExecutorConfig{ServiceAccountName: "foo"}
	woc.execWf.Spec.Templates[0].Executor = &wfv1.ExecutorConfig{ServiceAccountName: "tmpl"}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, "exec-sa-token", pod.Spec.Volumes[1].Name)
	assert.Equal(t, "tmpl-token", pod.Spec.Volumes[1].VolumeSource.Secret.SecretName)

	waitCtr := pod.Spec.Containers[0]
	verifyServiceAccountTokenVolumeMount(t, waitCtr, "exec-sa-token", "/var/run/secrets/kubernetes.io/serviceaccount")
}

// TestTmplLevelExecutorServiceAccountName verifies the ability to carry forward template level AutomountServiceAccountToken to Podspec.
func TestTmplLevelExecutorSecurityContext(t *testing.T) {
	var user int64 = 1000
	ctx := context.Background()
	woc := newWoc()
	_, err := util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "foo", "foo-token")
	assert.NoError(t, err)
	_, err = util.CreateServiceAccountWithToken(ctx, woc.controller.kubeclientset, "", "tmpl", "tmpl-token")
	assert.NoError(t, err)

	woc.controller.Config.Executor = &apiv1.Container{SecurityContext: &apiv1.SecurityContext{RunAsUser: &user}}
	woc.execWf.Spec.Templates[0].Executor = &wfv1.ExecutorConfig{ServiceAccountName: "tmpl"}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
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
	woc := newWoc()
	woc.execWf.Spec.ImagePullSecrets = []apiv1.LocalObjectReference{
		{
			Name: "secret-name",
		},
	}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, pod.Spec.ImagePullSecrets[0].Name, "secret-name")
}

// TestAffinity verifies the ability to carry forward affinity rules
func TestAffinity(t *testing.T) {
	woc := newWoc()
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
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.Affinity)
}

// TestTolerations verifies the ability to carry forward tolerations.
func TestTolerations(t *testing.T) {
	woc := newWoc()
	woc.execWf.Spec.Templates[0].Tolerations = []apiv1.Toleration{{
		Key:      "nvidia.com/gpu",
		Operator: "Exists",
		Effect:   "NoSchedule",
	}}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.Tolerations)
	assert.Equal(t, pod.Spec.Tolerations[0].Key, "nvidia.com/gpu")
}

// TestMetadata verifies ability to carry forward annotations and labels
func TestMetadata(t *testing.T) {
	woc := newWoc()
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.ObjectMeta)
	assert.NotNil(t, pod.ObjectMeta.Annotations)
	assert.NotNil(t, pod.ObjectMeta.Labels)
	for k, v := range woc.execWf.Spec.Templates[0].Metadata.Annotations {
		assert.Equal(t, pod.ObjectMeta.Annotations[k], v)
	}
	for k, v := range woc.execWf.Spec.Templates[0].Metadata.Labels {
		assert.Equal(t, pod.ObjectMeta.Labels[k], v)
	}
}

// TestWorkflowControllerArchiveConfig verifies archive location substitution of workflow
func TestWorkflowControllerArchiveConfig(t *testing.T) {
	ctx := context.Background()
	woc := newWoc()
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{workflow.name}}/{{pod.name}}",
	}})
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func setArtifactRepository(controller *WorkflowController, repo *wfv1.ArtifactRepository) {
	controller.artifactRepositories = armocks.DummyArtifactRepositories(repo)
}

// TestConditionalNoAddArchiveLocation verifies we do not add archive location if it is not needed
func TestConditionalNoAddArchiveLocation(t *testing.T) {
	ctx := context.Background()
	woc := newWoc()
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "path/in/bucket",
	}})
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	tmpl, err := getPodTemplate(&pod)
	assert.NoError(t, err)
	assert.Nil(t, tmpl.ArchiveLocation)
}

// TestConditionalNoAddArchiveLocation verifies we do  add archive location if it is needed for logs
func TestConditionalAddArchiveLocationArchiveLogs(t *testing.T) {
	ctx := context.Background()
	woc := newWoc()
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket: wfv1.S3Bucket{
				Bucket: "foo",
			},
			KeyFormat: "path/in/bucket",
		},
		ArchiveLogs: pointer.BoolPtr(true),
	})
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	tmpl, err := getPodTemplate(&pod)
	assert.NoError(t, err)
	assert.NotNil(t, tmpl.ArchiveLocation)
}

// TestConditionalNoAddArchiveLocation verifies we add archive location when it is needed
func TestConditionalArchiveLocation(t *testing.T) {
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	wf.Spec.Templates[0].Outputs = wfv1.Outputs{
		Artifacts: []wfv1.Artifact{
			{
				Name: "foo",
				Path: "/tmp/file",
			},
		},
	}
	woc := newWoc()
	setArtifactRepository(woc.controller, &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "path/in/bucket",
	}})
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	tmpl, err := getPodTemplate(&pod)
	assert.NoError(t, err)
	assert.Nil(t, tmpl.ArchiveLocation)
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
			cancel, controller := newController(wf, func(c *WorkflowController) {
				c.Config.ResourceRateLimit = &limit
			})
			defer cancel()
			woc := newWorkflowOperationCtx(wf, controller)
			woc.operate(context.Background())
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

func Test_createWorkflowPod_emissary(t *testing.T) {
	t.Run("NoCommand", func(t *testing.T) {
		woc := newWoc()
		woc.controller.containerRuntimeExecutor = common.ContainerRuntimeExecutorEmissary
		_, err := woc.createWorkflowPod(context.Background(), "", []apiv1.Container{{}}, &wfv1.Template{}, &createWorkflowPodOpts{})
		assert.Error(t, err)
	})
	t.Run("CommandNoArgs", func(t *testing.T) {
		woc := newWoc()
		woc.controller.containerRuntimeExecutor = common.ContainerRuntimeExecutorEmissary
		pod, err := woc.createWorkflowPod(context.Background(), "", []apiv1.Container{{Command: []string{"foo"}}}, &wfv1.Template{}, &createWorkflowPodOpts{})
		assert.NoError(t, err)
		assert.Equal(t, []string{"/var/run/argo/argoexec", "emissary", "--", "foo"}, pod.Spec.Containers[1].Command)
	})
	t.Run("NoCommandWithImageIndex", func(t *testing.T) {
		woc := newWoc()
		woc.controller.containerRuntimeExecutor = common.ContainerRuntimeExecutorEmissary
		pod, err := woc.createWorkflowPod(context.Background(), "", []apiv1.Container{{Image: "my-image"}}, &wfv1.Template{}, &createWorkflowPodOpts{})
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"/var/run/argo/argoexec", "emissary", "--", "my-cmd"}, pod.Spec.Containers[1].Command)
			assert.Equal(t, []string{"my-args"}, pod.Spec.Containers[1].Args)
		}
	})
	t.Run("NoCommandWithArgsWithImageIndex", func(t *testing.T) {
		woc := newWoc()
		woc.controller.containerRuntimeExecutor = common.ContainerRuntimeExecutorEmissary
		pod, err := woc.createWorkflowPod(context.Background(), "", []apiv1.Container{{Image: "my-image", Args: []string{"foo"}}}, &wfv1.Template{}, &createWorkflowPodOpts{})
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"/var/run/argo/argoexec", "emissary", "--", "my-cmd"}, pod.Spec.Containers[1].Command)
			assert.Equal(t, []string{"foo"}, pod.Spec.Containers[1].Args)
		}
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

	// For Docker executor
	t.Run("Docker", func(t *testing.T) {
		ctx := context.Background()
		woc := newWoc()
		woc.volumes = volumes
		woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorDocker

		tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
		assert.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		assert.NoError(t, err)
		pods, err := listPods(woc)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		assert.Equal(t, 2, len(pod.Spec.Volumes))
		assert.Equal(t, "docker-sock", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "volume-name", pod.Spec.Volumes[1].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[1].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[1].VolumeMounts[0].Name)
	})

	// For Kubelet executor
	t.Run("Kubelet", func(t *testing.T) {
		ctx := context.Background()
		woc := newWoc()
		woc.volumes = volumes
		woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorKubelet

		tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
		assert.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		assert.NoError(t, err)
		pods, err := listPods(woc)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		assert.Equal(t, 1, len(pod.Spec.Volumes))
		assert.Equal(t, "volume-name", pod.Spec.Volumes[0].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[1].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[1].VolumeMounts[0].Name)
	})

	// For K8sAPI executor
	t.Run("K8SAPI", func(t *testing.T) {
		ctx := context.Background()
		woc := newWoc()
		woc.volumes = volumes
		woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorK8sAPI

		tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
		assert.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		assert.NoError(t, err)
		pods, err := listPods(woc)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		assert.Equal(t, 1, len(pod.Spec.Volumes))
		assert.Equal(t, "volume-name", pod.Spec.Volumes[0].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[1].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[1].VolumeMounts[0].Name)
	})

	// For emissary executor
	t.Run("Emissary", func(t *testing.T) {
		ctx := context.Background()
		woc := newWoc()
		woc.volumes = volumes
		woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorEmissary

		tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
		assert.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		assert.NoError(t, err)
		pods, err := listPods(woc)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		if assert.Len(t, pod.Spec.Volumes, 2) {
			assert.Equal(t, "var-run-argo", pod.Spec.Volumes[0].Name)
			assert.Equal(t, "volume-name", pod.Spec.Volumes[1].Name)
		}
		if assert.Len(t, pod.Spec.InitContainers, 1) {
			init := pod.Spec.InitContainers[0]
			if assert.Len(t, init.VolumeMounts, 1) {
				assert.Equal(t, "var-run-argo", init.VolumeMounts[0].Name)
			}
		}
		containers := pod.Spec.Containers
		if assert.Len(t, containers, 2) {
			wait := containers[0]
			if assert.Len(t, wait.VolumeMounts, 2) {
				assert.Equal(t, "volume-name", wait.VolumeMounts[0].Name)
				assert.Equal(t, "var-run-argo", wait.VolumeMounts[1].Name)
			}
			main := containers[1]
			assert.Equal(t, []string{"/var/run/argo/argoexec", "emissary", "--", "cowsay"}, main.Command)
			if assert.Len(t, main.VolumeMounts, 2) {
				assert.Equal(t, "volume-name", main.VolumeMounts[0].Name)
				assert.Equal(t, "var-run-argo", main.VolumeMounts[1].Name)
			}
		}
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

	ctx := context.Background()
	woc := newWoc()
	woc.volumes = volumes
	woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.execWf.Spec.Templates[0].Inputs.Parameters = inputParameters
	woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorDocker

	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, 2, len(pod.Spec.Volumes))
	assert.Equal(t, "volume-name", pod.Spec.Volumes[1].Name)
	assert.Equal(t, "test-name", pod.Spec.Volumes[1].PersistentVolumeClaim.ClaimName)
	assert.Equal(t, 1, len(pod.Spec.Containers[1].VolumeMounts))
	assert.Equal(t, "volume-name", pod.Spec.Containers[1].VolumeMounts[0].Name)
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
		ctx := context.Background()
		woc := newWoc()
		woc.controller.Config.KubeConfig = &config.KubeConfig{
			SecretName: "foo",
			SecretKey:  "bar",
		}

		tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
		assert.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		assert.NoError(t, err)
		pods, err := listPods(woc)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		assert.Equal(t, "kubeconfig", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "foo", pod.Spec.Volumes[0].VolumeSource.Secret.SecretName)

		waitCtr := pod.Spec.Containers[0]
		verifyKubeConfigVolume(waitCtr, "kubeconfig", "/kube/config")
	}

	// custom mount path & volume name, in case name collision
	{
		ctx := context.Background()
		woc := newWoc()
		woc.controller.Config.KubeConfig = &config.KubeConfig{
			SecretName: "foo",
			SecretKey:  "bar",
			MountPath:  "/some/path/config",
			VolumeName: "kube-config-secret",
		}

		tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
		assert.NoError(t, err)
		_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
		assert.NoError(t, err)
		pods, err := listPods(woc)
		assert.NoError(t, err)
		assert.Len(t, pods.Items, 1)
		pod := pods.Items[0]
		assert.Equal(t, "kube-config-secret", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "foo", pod.Spec.Volumes[0].VolumeSource.Secret.SecretName)

		// kubeconfig volume is the last one
		waitCtr := pod.Spec.Containers[0]
		verifyKubeConfigVolume(waitCtr, "kube-config-secret", "/some/path/config")
	}
}

// TestPriority verifies the ability to carry forward priorityClassName and priority.
func TestPriority(t *testing.T) {
	priority := int32(15)
	ctx := context.Background()
	woc := newWoc()
	woc.execWf.Spec.Templates[0].PriorityClassName = "foo"
	woc.execWf.Spec.Templates[0].Priority = &priority
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, pod.Spec.PriorityClassName, "foo")
	assert.Equal(t, pod.Spec.Priority, &priority)
}

// TestSchedulerName verifies the ability to carry forward schedulerName.
func TestSchedulerName(t *testing.T) {
	ctx := context.Background()
	woc := newWoc()
	woc.execWf.Spec.Templates[0].SchedulerName = "foo"
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, pod.Spec.SchedulerName, "foo")
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

	ctx := context.Background()
	woc := newWoc()
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

	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, 1, len(pod.Spec.InitContainers))
	assert.Equal(t, "init-foo", pod.Spec.InitContainers[0].Name)
	for _, v := range volumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
	assert.Equal(t, 2, len(pod.Spec.InitContainers[0].VolumeMounts))
	assert.Equal(t, "init-volume-name", pod.Spec.InitContainers[0].VolumeMounts[0].Name)
	assert.Equal(t, "volume-name", pod.Spec.InitContainers[0].VolumeMounts[1].Name)
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

	ctx := context.Background()
	woc := newWoc()
	woc.volumes = volumes
	woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.execWf.Spec.Templates[0].Sidecars = []wfv1.UserContainer{
		{
			MirrorVolumeMounts: &mirrorVolumeMounts,
			Container: apiv1.Container{
				Name:         "side-foo",
				VolumeMounts: sidecarVolumeMounts,
			},
		},
	}

	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.Equal(t, 3, len(pod.Spec.Containers))
	assert.Equal(t, "wait", pod.Spec.Containers[0].Name)
	assert.Equal(t, "main", pod.Spec.Containers[1].Name)
	assert.Equal(t, "side-foo", pod.Spec.Containers[2].Name)
	for _, v := range volumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
	assert.Equal(t, 2, len(pod.Spec.Containers[2].VolumeMounts))
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

	ctx := context.Background()
	woc := newWoc()
	woc.volumes = volumes
	woc.execWf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.execWf.Spec.Templates[0].Volumes = localVolumes

	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
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
	ctx := context.Background()
	woc := newWoc()
	woc.execWf.Spec.HostAliases = []apiv1.HostAlias{
		{IP: "127.0.0.1"},
		{IP: "127.0.0.1"},
	}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.HostAliases)
}

// TestTmplLevelHostAliases verifies the ability to carry forward template level HostAliases to Podspec
func TestTmplLevelHostAliases(t *testing.T) {
	ctx := context.Background()
	woc := newWoc()
	woc.execWf.Spec.Templates[0].HostAliases = []apiv1.HostAlias{
		{IP: "127.0.0.1"},
		{IP: "127.0.0.1"},
	}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.HostAliases)
}

// TestWFLevelSecurityContext verifies the ability to carry forward workflow level SecurityContext to Podspec
func TestWFLevelSecurityContext(t *testing.T) {
	ctx := context.Background()
	woc := newWoc()
	runAsUser := int64(1234)
	woc.execWf.Spec.SecurityContext = &apiv1.PodSecurityContext{
		RunAsUser: &runAsUser,
	}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.SecurityContext)
	assert.Equal(t, runAsUser, *pod.Spec.SecurityContext.RunAsUser)
}

// TestTmplLevelSecurityContext verifies the ability to carry forward template level SecurityContext to Podspec
func TestTmplLevelSecurityContext(t *testing.T) {
	ctx := context.Background()
	woc := newWoc()
	runAsUser := int64(1234)
	woc.execWf.Spec.Templates[0].SecurityContext = &apiv1.PodSecurityContext{
		RunAsUser: &runAsUser,
	}
	tmplCtx, err := woc.createTemplateContext(wfv1.ResourceScopeLocal, "")
	assert.NoError(t, err)
	_, err = woc.executeContainer(ctx, woc.execWf.Spec.Entrypoint, tmplCtx.GetTemplateScope(), &woc.execWf.Spec.Templates[0], &wfv1.WorkflowStep{}, &executeTemplateOpts{})
	assert.NoError(t, err)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	assert.NotNil(t, pod.Spec.SecurityContext)
	assert.Equal(t, runAsUser, *pod.Spec.SecurityContext.RunAsUser)
}

var helloWorldWfWithPatch = `
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
`

var helloWorldWfWithWFPatch = `
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

var helloWorldWfWithWFYAMLPatch = `
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

var helloWorldWfWithInvalidPatchFormat = `
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

func TestPodSpecPatch(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfWithPatch)
	ctx := context.Background()
	woc := newWoc(*wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithWFPatch)
	woc = newWoc(*wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithWFYAMLPatch)
	woc = newWoc(*wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})

	assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "104857600", pod.Spec.Containers[1].Resources.Limits.Memory().AsDec().String())

	wf = wfv1.MustUnmarshalWorkflow(helloWorldWfWithInvalidPatchFormat)
	woc = newWoc(*wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	_, err := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.EqualError(t, err, "Failed to merge the workflow PodSpecPatch with the template PodSpecPatch due to invalid format")
}

func TestMainContainerCustomization(t *testing.T) {
	mainCtrSpec := &apiv1.Container{
		Name: common.MainContainerName,
		Resources: apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.200"),
				apiv1.ResourceMemory: resource.MustParse("512Mi"),
			},
		},
	}
	// podSpecPatch in workflow spec takes precedence over the main container's
	// configuration in controller so here we respect what's specified in podSpecPatch.
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWfWithPatch)
	woc := newWoc(*wf)
	ctx := context.Background()
	woc.controller.Config.MainContainer = mainCtrSpec
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.800", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	// The main container's resources should be changed since the existing
	// container's resources are not specified.
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc = newWoc(*wf)
	woc.controller.Config.MainContainer = mainCtrSpec
	mainCtr = woc.execWf.Spec.Templates[0].Container
	mainCtr.Resources = apiv1.ResourceRequirements{Limits: apiv1.ResourceList{}}
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.200", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	// Workflow spec's main container takes precedence over config in controller
	// so here the main container resources remain unchanged.
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc = newWoc(*wf)
	woc.controller.Config.MainContainer = mainCtrSpec
	mainCtr = woc.execWf.Spec.Templates[0].Container
	wf.Spec.Templates[0].Container.Name = common.MainContainerName
	wf.Spec.Templates[0].Container.Resources = apiv1.ResourceRequirements{
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse("0.900"),
			apiv1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.900", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())

	// If the name of the container in the workflow spec is not "main",
	// the main container resources should remain unchanged.
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc = newWoc(*wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	mainCtr.Resources = apiv1.ResourceRequirements{
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse("0.100"),
			apiv1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	wf.Spec.Templates[0].Container.Name = "non-main"
	wf.Spec.Templates[0].Container.Resources = apiv1.ResourceRequirements{
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse("0.900"),
			apiv1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "0.100", pod.Spec.Containers[1].Resources.Limits.Cpu().AsDec().String())
}

func TestIsResourcesSpecified(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWoc(*wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	mainCtr.Resources = apiv1.ResourceRequirements{Limits: apiv1.ResourceList{}}
	assert.False(t, isResourcesSpecified(mainCtr))

	mainCtr.Resources = apiv1.ResourceRequirements{
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse("0.900"),
			apiv1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	assert.True(t, isResourcesSpecified(mainCtr))
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

var helloLinuxWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-hybrid-lin
spec:
  entrypoint: hello-linux
  templates:
    - name: hello-linux
      nodeSelector:
        kubernetes.io/os: linux
      container:
        image: alpine
        command: [echo]
        args: ["Hello from Linux Container!"]
`

func TestHybridWfVolumesWindows(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWindowsWf)
	woc := newWoc(*wf)

	ctx := context.Background()
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "\\\\.\\pipe\\docker_engine", pod.Spec.Containers[0].VolumeMounts[0].MountPath)
	assert.Equal(t, false, pod.Spec.Containers[0].VolumeMounts[0].ReadOnly)
	assert.Equal(t, (*apiv1.HostPathType)(nil), pod.Spec.Volumes[0].HostPath.Type)
}

func TestHybridWfVolumesLinux(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloLinuxWf)
	woc := newWoc(*wf)

	ctx := context.Background()
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "/var/run/docker.sock", pod.Spec.Containers[0].VolumeMounts[0].MountPath)
	assert.Equal(t, true, pod.Spec.Containers[0].VolumeMounts[0].ReadOnly)
	assert.Equal(t, &hostPathSocket, pod.Spec.Volumes[0].HostPath.Type)
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
	// Ensure that volume mount is added when artifact is provided
	tmpl := unmarshalTemplate(propagateMaxDuration)
	woc := newWoc()
	deadline := time.Time{}.Add(time.Second)
	ctx := context.Background()
	pod, err := woc.createWorkflowPod(ctx, tmpl.Name, []apiv1.Container{*tmpl.Container}, tmpl, &createWorkflowPodOpts{executionDeadline: deadline})
	assert.NoError(t, err)
	v, err := getPodDeadline(pod)
	assert.NoError(t, err)
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
	ctx := context.Background()
	woc := newWoc(*wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "foo", pod.ObjectMeta.Annotations["workflow-level-pod-annotation"])
	assert.Equal(t, "bar", pod.ObjectMeta.Labels["workflow-level-pod-label"])

	wf = wfv1.MustUnmarshalWorkflow(wfWithPodMetadataAndTemplateMetadata)
	woc = newWoc(*wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	assert.Equal(t, "fizz", pod.ObjectMeta.Annotations["workflow-level-pod-annotation"])
	assert.Equal(t, "buzz", pod.ObjectMeta.Labels["workflow-level-pod-label"])
	assert.Equal(t, "hello", pod.ObjectMeta.Annotations["template-level-pod-annotation"])
	assert.Equal(t, "world", pod.ObjectMeta.Labels["template-level-pod-label"])
}

func TestGetDeadline(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := context.Background()
	woc := newWoc(*wf)
	mainCtr := woc.execWf.Spec.Templates[0].Container
	pod, _ := woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{})
	deadline, _ := getPodDeadline(pod)
	assert.Equal(t, time.Time{}, deadline)

	executionDeadline := time.Now().Add(5 * time.Minute)
	wf = wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx = context.Background()
	woc = newWoc(*wf)
	mainCtr = woc.execWf.Spec.Templates[0].Container
	pod, _ = woc.createWorkflowPod(ctx, wf.Name, []apiv1.Container{*mainCtr}, &wf.Spec.Templates[0], &createWorkflowPodOpts{executionDeadline: executionDeadline})
	deadline, _ = getPodDeadline(pod)
	assert.Equal(t, executionDeadline.Format(time.RFC3339), deadline.Format(time.RFC3339))
}
