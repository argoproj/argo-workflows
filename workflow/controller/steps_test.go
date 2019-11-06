package controller

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test"
)

// TestStepsFailedRetries ensures a steps template will recognize exhausted retries
func TestStepsFailedRetries(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/steps-failed-retries.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeFailed), string(woc.wf.Status.Phase))
}

var stepsOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-on-exit
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: steps1
        template: stepsTemplate

  - name: stepsTemplate
    onExit: exitContainer
    steps:
    - - name: leafA
        template: whalesay
    - - name: leafB
        template: whalesay

  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]

  - name: exitContainer
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
`

func TestStepsOnExit(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(stepsOnExit)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()

	wf, err = wfcset.Get(wf.ObjectMeta.Name, v1.GetOptions{})
	assert.Nil(t, err)
	onExitNodeIsPresent := false
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}