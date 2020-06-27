package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

var MockParamValue string = "Hello world"

var MockParam = wfv1.Parameter{
	Name: "hello",
	Value: &MockParamValue,
}

var workflowCached = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: memoized-workflow-test
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: message
      value: hi there world
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
    memoize:
      key: "{{inputs.parameters.message}}"
      maxAge: 1d
      cache:
        configMapName:
          name: whalesay-cache
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 10; cowsay {{inputs.parameters.message}} > /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: hello
        valueFrom:
          path: /tmp/hello_world.txt
`

func TestCacheLoad(t *testing.T) {
	wf := unmarshalWF(workflowCached)
	woc := newWoc(*wf)
	woc.operate()

	result := wf.Status.Outputs
	entry, ok := woc.controller.cache.Load("hi-there-world")
	assert.False(t, ok)
	assert.Equal(t, result, entry)
}

func TestCacheSave(t *testing.T) {
	wf := unmarshalWF(workflowCached)
	woc := newWoc(*wf)
	woc.operate()

	outputs := wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, MockParam)
	ok := woc.controller.cache.Save("hello", &outputs)
	assert.False(t, ok)
}

