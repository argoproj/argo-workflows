package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var origWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  arguments:
    parameters:
    - name: message
      value: original
  entrypoint: start
  onExit: end
  serviceAccountName: argo
  workflowTemplateRef:
    name: workflow-template-submittable
`
var patchWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  arguments:
    parameters:
    - name: message
      value: patch
  serviceAccountName: argo1
  podGC:
    strategy: OnPodSuccess
`

func TestMergeWorkflows(t *testing.T) {
	patchWf := unmarshalWF(origWF)
	targetWf := unmarshalWF(patchWF)

	err := MergeTo(patchWf, targetWf)
	assert.NoError(t, err)
	assert.Equal(t, "start", targetWf.Spec.Entrypoint)
	assert.Equal(t, "argo1", targetWf.Spec.ServiceAccountName)
	assert.Equal(t, "message", targetWf.Spec.Arguments.Parameters[0].Name)
	assert.Equal(t, "patch", targetWf.Spec.Arguments.Parameters[0].Value.String())
}

func TestMergeMetaDataTo(t *testing.T) {
	assert := assert.New(t)
	meta1 := &metav1.ObjectMeta{
		Labels: map[string]string{
			"test": "test", "welcome": "welcome",
		},
		Annotations: map[string]string{
			"test": "test", "welcome": "welcome",
		},
	}
	meta2 := &metav1.ObjectMeta{
		Labels: map[string]string{
			"test1": "test", "welcome1": "welcome",
		},
		Annotations: map[string]string{
			"test1": "test", "welcome1": "welcome",
		},
	}
	MergeMetaDataTo(meta2, meta1)
	assert.Contains(meta1.Labels, "test1")
	assert.Contains(meta1.Annotations, "test1")
	assert.NotContains(meta2.Labels, "test")
}

var wfDefault = `
metadata: 
  annotations: 
    testAnnotation: test
  labels: 
    testLabel: test
spec: 
  entrypoint: whalesay
  activeDeadlineSeconds: 7200
  arguments: 
    artifacts: 
      -
        name: message
        path: /tmp/message
    parameters: 
      - 
        name: message
        value: "hello world"
  onExit: whalesay-exit
  serviceAccountName: argo
  templates: 
    - 
      container: 
        args: 
          - "hello from the default exit handler"
        command: 
          - cowsay
        image: docker/whalesay
      name: whalesay-exit
  ttlSecondsAfterFinished: 86400
  ttlStrategy: 
    secondsAfterCompletion: 60
  volumes: 
    - 
      name: test
      secret: 
        secretName: test
`
var wft = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-submittable
  namespace: default
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: hello world
  templates:
    - name: whalesay-template
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
`

var wf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: 
  generateName: hello-world-
spec: 
  entrypoint: whalesay
  templates: 
    - 
      container: 
        args: 
          - "hello world"
        command: 
          - cowsay
        image: "docker/whalesay:latest"
      name: whalesay
`

var resultSpec = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: 
  generateName: hello-world-
spec: 
  activeDeadlineSeconds: 7200
  arguments: 
    artifacts: 
      - 
        name: message
        path: /tmp/message
    parameters: 
      - 
        name: message
        value: "hello world"
  entrypoint: whalesay
  onExit: whalesay-exit
  serviceAccountName: argo
  templates: 
    - 
      container: 
        args: 
          - "hello world"
        command: 
          - cowsay
        image: "docker/whalesay:latest"
      name: whalesay
    - 
      container: 
        args: 
          - "{{inputs.parameters.message}}"
        command: 
          - cowsay
        image: docker/whalesay
      inputs: 
        parameters: 
          - 
            name: message
      name: whalesay-template
    - 
      container: 
        args: 
          - "hello from the default exit handler"
        command: 
          - cowsay
        image: docker/whalesay
      name: whalesay-exit
  ttlSecondsAfterFinished: 86400
  ttlStrategy: 
    secondsAfterCompletion: 60
  volumes: 
    - 
      name: test
      secret: 
        secretName: test

`

func TestMergeWfSpecs(t *testing.T) {
	assert := assert.New(t)
	wfDefault := unmarshalWF(wfDefault)
	wf1 := unmarshalWF(wf)
	//wf1 := unmarshalWF(wf1)
	wft := unmarshalWFT(wft)
	result := unmarshalWF(resultSpec)

	targetWf, err := MergeWfSpecs(&wf1.Spec, wft.GetWorkflowSpec(), &wfDefault.Spec)
	assert.NoError(err)
	assert.Equal(result.Spec, targetWf.Spec)
	assert.Equal(1, len(wf1.Spec.Templates))
	assert.Equal("whalesay", wf1.Spec.Entrypoint)
	//assert.Equal(60, wf1.Spec.TTLStrategy.SecondsAfterCompletion)
	//assert.Equal("whalesay-exit", wf1.Spec.OnExit)

}
