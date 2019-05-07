package controller

import (
	"encoding/json"
	"fmt"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func unmarshalTemplate(yamlStr string) *wfv1.Template {
	var tmpl wfv1.Template
	err := yaml.Unmarshal([]byte(yamlStr), &tmpl)
	if err != nil {
		panic(err)
	}
	return &tmpl
}

// newWoc a new operation context suitable for testing
func newWoc(wfs ...wfv1.Workflow) *wfOperationCtx {
	var wf *wfv1.Workflow
	if len(wfs) == 0 {
		wf = unmarshalWF(helloWorldWf)
	} else {
		wf = &wfs[0]
	}
	fakeController := newController()
	_, err := fakeController.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(wf)
	if err != nil {
		panic(err)
	}
	woc := newWorkflowOperationCtx(wf, fakeController)
	return woc
}

// getPodName returns the podname of the created pod of a workflow
// Only applies to single pod workflows
func getPodName(wf *wfv1.Workflow) string {
	if len(wf.Status.Nodes) != 1 {
		panic("getPodName called against a multi-pod workflow")
	}
	for podName := range wf.Status.Nodes {
		return podName
	}
	return ""
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
	tmpl := unmarshalTemplate(scriptTemplateWithInputArtifact)
	node := newWoc().executeScript(tmpl.Name, tmpl, "")
	assert.Equal(t, node.Phase, wfv1.NodePending)
}

// TestServiceAccount verifies the ability to carry forward the service account name
// for the pod from workflow.spec.serviceAccountName.
func TestServiceAccount(t *testing.T) {
	woc := newWoc()
	woc.wf.Spec.ServiceAccountName = "foo"
	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, pod.Spec.ServiceAccountName, "foo")
}

// TestImagePullSecrets verifies the ability to carry forward imagePullSecrets from workflow.spec
func TestImagePullSecrets(t *testing.T) {
	woc := newWoc()
	woc.wf.Spec.ImagePullSecrets = []apiv1.LocalObjectReference{
		{
			Name: "secret-name",
		},
	}
	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, pod.Spec.ImagePullSecrets[0].Name, "secret-name")
}

// TestAffinity verifies the ability to carry forward affinity rules
func TestAffinity(t *testing.T) {
	woc := newWoc()
	woc.wf.Spec.Affinity = &apiv1.Affinity{
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
	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, pod.Spec.Affinity)
}

// TestTolerations verifies the ability to carry forward tolerations.
func TestTolerations(t *testing.T) {
	woc := newWoc()
	woc.wf.Spec.Templates[0].Tolerations = []apiv1.Toleration{{
		Key:      "nvidia.com/gpu",
		Operator: "Exists",
		Effect:   "NoSchedule",
	}}
	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, pod.Spec.Tolerations)
	assert.Equal(t, pod.Spec.Tolerations[0].Key, "nvidia.com/gpu")
}

// TestMetadata verifies ability to carry forward annotations and labels
func TestMetadata(t *testing.T) {
	woc := newWoc()
	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})

	assert.Nil(t, err)
	assert.NotNil(t, pod.ObjectMeta)
	assert.NotNil(t, pod.ObjectMeta.Annotations)
	assert.NotNil(t, pod.ObjectMeta.Labels)
	for k, v := range woc.wf.Spec.Templates[0].Metadata.Annotations {
		assert.Equal(t, pod.ObjectMeta.Annotations[k], v)
	}
	for k, v := range woc.wf.Spec.Templates[0].Metadata.Labels {
		assert.Equal(t, pod.ObjectMeta.Labels[k], v)
	}
}

// TestWorkflowControllerArchiveConfig verifies archive location substitution of workflow
func TestWorkflowControllerArchiveConfig(t *testing.T) {
	woc := newWoc()
	woc.controller.Config.ArtifactRepository.S3 = &S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{workflow.name}}/{{pod.name}}",
	}
	woc.operate()
	podName := getPodName(woc.wf)
	_, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.NoError(t, err)
}

// TestWorkflowControllerArchiveConfigUnresolvable verifies workflow fails when archive location has
// unresolvable variables
func TestWorkflowControllerArchiveConfigUnresolvable(t *testing.T) {
	wf := unmarshalWF(helloWorldWf)
	wf.Spec.Templates[0].Outputs = wfv1.Outputs{
		Artifacts: []wfv1.Artifact{
			{
				Name: "foo",
				Path: "/tmp/file",
			},
		},
	}
	woc := newWoc(*wf)
	woc.controller.Config.ArtifactRepository.S3 = &S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "{{workflow.unresolvable}}",
	}
	woc.operate()
	podName := getPodName(woc.wf)
	_, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Error(t, err)
}

// TestConditionalNoAddArchiveLocation verifies we do not add archive location if it is not needed
func TestConditionalNoAddArchiveLocation(t *testing.T) {
	woc := newWoc()
	woc.controller.Config.ArtifactRepository.S3 = &S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "path/in/bucket",
	}
	woc.operate()
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.NoError(t, err)
	var tmpl wfv1.Template
	err = json.Unmarshal([]byte(pod.Annotations[common.AnnotationKeyTemplate]), &tmpl)
	assert.NoError(t, err)
	assert.Nil(t, tmpl.ArchiveLocation)
}

// TestConditionalNoAddArchiveLocation verifies we add archive location when it is needed
func TestConditionalArchiveLocation(t *testing.T) {
	wf := unmarshalWF(helloWorldWf)
	wf.Spec.Templates[0].Outputs = wfv1.Outputs{
		Artifacts: []wfv1.Artifact{
			{
				Name: "foo",
				Path: "/tmp/file",
			},
		},
	}
	woc := newWoc()
	woc.controller.Config.ArtifactRepository.S3 = &S3ArtifactRepository{
		S3Bucket: wfv1.S3Bucket{
			Bucket: "foo",
		},
		KeyFormat: "path/in/bucket",
	}
	woc.operate()
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.NoError(t, err)
	var tmpl wfv1.Template
	err = json.Unmarshal([]byte(pod.Annotations[common.AnnotationKeyTemplate]), &tmpl)
	assert.NoError(t, err)
	assert.Nil(t, tmpl.ArchiveLocation)
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
	{
		woc := newWoc()
		woc.volumes = volumes
		woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorDocker

		woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
		podName := getPodName(woc.wf)
		pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, 3, len(pod.Spec.Volumes))
		assert.Equal(t, "podmetadata", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "docker-sock", pod.Spec.Volumes[1].Name)
		assert.Equal(t, "volume-name", pod.Spec.Volumes[2].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[1].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[1].VolumeMounts[0].Name)
	}

	// For Kubelet executor
	{
		woc := newWoc()
		woc.volumes = volumes
		woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorKubelet

		woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
		podName := getPodName(woc.wf)
		pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(pod.Spec.Volumes))
		assert.Equal(t, "podmetadata", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "volume-name", pod.Spec.Volumes[1].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[1].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[1].VolumeMounts[0].Name)
	}

	// For K8sAPI executor
	{
		woc := newWoc()
		woc.volumes = volumes
		woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorK8sAPI

		woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
		podName := getPodName(woc.wf)
		pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(pod.Spec.Volumes))
		assert.Equal(t, "podmetadata", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "volume-name", pod.Spec.Volumes[1].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[1].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[1].VolumeMounts[0].Name)
	}
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
	tmpStr := "test-name"
	inputParameters := []wfv1.Parameter{
		{
			Name:  "volume-name",
			Value: &tmpStr,
		},
	}

	woc := newWoc()
	woc.volumes = volumes
	woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.wf.Spec.Templates[0].Inputs.Parameters = inputParameters
	woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorDocker

	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(pod.Spec.Volumes))
	assert.Equal(t, "volume-name", pod.Spec.Volumes[2].Name)
	assert.Equal(t, "test-name", pod.Spec.Volumes[2].PersistentVolumeClaim.ClaimName)
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
		woc := newWoc()
		woc.controller.Config.KubeConfig = &KubeConfig{
			SecretName: "foo",
			SecretKey:  "bar",
		}

		woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
		podName := getPodName(woc.wf)
		pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})

		assert.Nil(t, err)
		assert.Equal(t, "kubeconfig", pod.Spec.Volumes[1].Name)
		assert.Equal(t, "foo", pod.Spec.Volumes[1].VolumeSource.Secret.SecretName)

		waitCtr := pod.Spec.Containers[0]
		verifyKubeConfigVolume(waitCtr, "kubeconfig", "/kube/config")
	}

	// custom mount path & volume name, in case name collision
	{
		woc := newWoc()
		woc.controller.Config.KubeConfig = &KubeConfig{
			SecretName: "foo",
			SecretKey:  "bar",
			MountPath:  "/some/path/config",
			VolumeName: "kube-config-secret",
		}

		woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
		podName := getPodName(woc.wf)
		pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})

		assert.Nil(t, err)
		assert.Equal(t, "kube-config-secret", pod.Spec.Volumes[1].Name)
		assert.Equal(t, "foo", pod.Spec.Volumes[1].VolumeSource.Secret.SecretName)

		// kubeconfig volume is the last one
		waitCtr := pod.Spec.Containers[0]
		verifyKubeConfigVolume(waitCtr, "kube-config-secret", "/some/path/config")
	}
}

// TestPriority verifies the ability to carry forward priorityClassName and priority.
func TestPriority(t *testing.T) {
	priority := int32(15)
	woc := newWoc()
	woc.wf.Spec.Templates[0].PriorityClassName = "foo"
	woc.wf.Spec.Templates[0].Priority = &priority
	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, pod.Spec.PriorityClassName, "foo")
	assert.Equal(t, pod.Spec.Priority, &priority)
}

// TestSchedulerName verifies the ability to carry forward schedulerName.
func TestSchedulerName(t *testing.T) {
	woc := newWoc()
	woc.wf.Spec.Templates[0].SchedulerName = "foo"
	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
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

	woc := newWoc()
	woc.volumes = volumes
	woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.wf.Spec.Templates[0].InitContainers = []wfv1.UserContainer{
		{
			MirrorVolumeMounts: &mirrorVolumeMounts,
			Container: apiv1.Container{
				Name:         "init-foo",
				VolumeMounts: initVolumeMounts,
			},
		},
	}

	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
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

	woc := newWoc()
	woc.volumes = volumes
	woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.wf.Spec.Templates[0].Sidecars = []wfv1.UserContainer{
		{
			MirrorVolumeMounts: &mirrorVolumeMounts,
			Container: apiv1.Container{
				Name:         "side-foo",
				VolumeMounts: sidecarVolumeMounts,
			},
		},
	}

	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
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

	woc := newWoc()
	woc.volumes = volumes
	woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
	woc.wf.Spec.Templates[0].Volumes = localVolumes

	woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
	podName := getPodName(woc.wf)
	pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	for _, v := range volumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
	for _, v := range localVolumes {
		assert.Contains(t, pod.Spec.Volumes, v)
	}
}
