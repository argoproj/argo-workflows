package controller

import (
	"testing"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
func newWoc() *wfOperationCtx {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
		},
	}
	woc := wfOperationCtx{
		wf:      wf,
		orig:    wf.DeepCopyObject().(*wfv1.Workflow),
		updated: false,
		log: log.WithFields(log.Fields{
			"workflow":  wf.ObjectMeta.Name,
			"namespace": wf.ObjectMeta.Namespace,
		}),
		controller: &WorkflowController{
			Config: WorkflowControllerConfig{
				ExecutorImage: "executor:latest",
			},
			clientset: fake.NewSimpleClientset(),
		},
		completedPods: make(map[string]bool),
	}
	return &woc
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

func TestScriptTemplateWithVolume(t *testing.T) {
	// ensure we can a script pod with input artifacts
	tmpl := unmarshalTemplate(scriptTemplateWithInputArtifact)
	_, err := newWoc().createWorkflowPod(tmpl.Name, tmpl)
	assert.Nil(t, err)
}
