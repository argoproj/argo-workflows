package controller

import (
	"testing"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	err := newWoc().executeScript(tmpl.Name, tmpl)
	assert.Nil(t, err)
}

// TestServiceAccount verifies the ability to carry forward the service account name
// for the pod from workflow.spec.serviceAccountName.
func TestServiceAccount(t *testing.T) {
	woc := newWoc()
	woc.wf.Spec.ServiceAccountName = "foo"
	err := woc.executeContainer(woc.wf.Spec.Entrypoint, &woc.wf.Spec.Templates[0])
	assert.Nil(t, err)
	podName := getPodName(woc.wf)
	pod, err := woc.controller.clientset.CoreV1().Pods("").Get(podName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, pod.Spec.ServiceAccountName, "foo")
}
