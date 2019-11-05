package controller

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
  generateName: suspend-template-
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: steps1
        template: stepsTempalte
    - - name: steps2
        template: stepsTempalte

  - name: stepsTempalte
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
	wf := unmarshalWF(scriptOnExit)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()

	// start template is run
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(v1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(pods.Items))


	woc.operate()
	woc.operate()

	// exitContainer template is run
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(v1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(pods.Items))
}