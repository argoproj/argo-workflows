package controller

import (
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
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
	woc := wfOperationCtx{
		wf:      wf,
		orig:    wf.DeepCopyObject().(*wfv1.Workflow),
		updated: false,
		log: log.WithFields(log.Fields{
			"workflow":  wf.ObjectMeta.Name,
			"namespace": wf.ObjectMeta.Namespace,
		}),
		controller:    newController(),
		completedPods: make(map[string]bool),
	}
	if woc.wf.Status.Nodes == nil {
		woc.wf.Status.Nodes = make(map[string]wfv1.NodeStatus)
	}
	return &woc
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
	assert.Equal(t, node.Phase, wfv1.NodeRunning)
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
