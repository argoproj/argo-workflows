package controller

import (
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
	woc := newWoc()
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
		woc.wf.Spec.Volumes = volumes
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
		assert.Equal(t, 1, len(pod.Spec.Containers[0].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[0].VolumeMounts[0].Name)
	}

	// For Kubelet executor
	{
		woc := newWoc()
		woc.wf.Spec.Volumes = volumes
		woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorKubelet

		woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
		podName := getPodName(woc.wf)
		pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(pod.Spec.Volumes))
		assert.Equal(t, "podmetadata", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "volume-name", pod.Spec.Volumes[1].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[0].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[0].VolumeMounts[0].Name)
	}

	// For K8sAPI executor
	{
		woc := newWoc()
		woc.wf.Spec.Volumes = volumes
		woc.wf.Spec.Templates[0].Container.VolumeMounts = volumeMounts
		woc.controller.Config.ContainerRuntimeExecutor = common.ContainerRuntimeExecutorK8sAPI

		woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0], "")
		podName := getPodName(woc.wf)
		pod, err := woc.controller.kubeclientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(pod.Spec.Volumes))
		assert.Equal(t, "podmetadata", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "volume-name", pod.Spec.Volumes[1].Name)
		assert.Equal(t, 1, len(pod.Spec.Containers[0].VolumeMounts))
		assert.Equal(t, "volume-name", pod.Spec.Containers[0].VolumeMounts[0].Name)
	}
}

func TestOutOfCluster(t *testing.T) {
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

		// kubeconfig volume is the last one
		idx := len(pod.Spec.Containers[1].VolumeMounts) - 1
		assert.Equal(t, "kubeconfig", pod.Spec.Containers[1].VolumeMounts[idx].Name)
		assert.Equal(t, "/kube/config", pod.Spec.Containers[1].VolumeMounts[idx].MountPath)
		assert.Equal(t, "--kubeconfig=/kube/config", pod.Spec.Containers[1].Args[1])
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
		idx := len(pod.Spec.Containers[1].VolumeMounts) - 1
		assert.Equal(t, "kube-config-secret", pod.Spec.Containers[1].VolumeMounts[idx].Name)
		assert.Equal(t, "/some/path/config", pod.Spec.Containers[1].VolumeMounts[idx].MountPath)
		assert.Equal(t, "--kubeconfig=/some/path/config", pod.Spec.Containers[1].Args[1])
	}
}
