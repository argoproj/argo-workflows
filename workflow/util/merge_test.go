package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	patchWf := wfv1.MustUnmarshalWorkflow(origWF)
	targetWf := wfv1.MustUnmarshalWorkflow(patchWF)

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
	mergeMetaDataTo(meta2, meta1)
	assert.Contains(meta1.Labels, "test1")
	assert.Contains(meta1.Annotations, "test1")
	assert.NotContains(meta2.Labels, "test")
}

var wfDefault = `
metadata: 
  annotations: 
    testAnnotation: default
  labels: 
    testLabel: default
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
  workflowMetaData:
    annotations: 
      testAnnotation: wft
    labels:
      testLabel: wft
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
  workflowMetadata:
    annotations:
      testAnnotation: wft
    labels: 
      testLabel: wft 
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
  ttlStrategy: 
    secondsAfterCompletion: 60
  volumes: 
    - 
      name: test
      secret: 
        secretName: test

`

func TestJoinWfSpecs(t *testing.T) {
	assert := assert.New(t)
	wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
	wf1 := wfv1.MustUnmarshalWorkflow(wf)
	// wf1 := wfv1.MustUnmarshalWorkflow(wf1)
	wft := wfv1.MustUnmarshalWorkflowTemplate(wft)
	result := wfv1.MustUnmarshalWorkflow(resultSpec)

	targetWf, err := JoinWorkflowSpec(&wf1.Spec, wft.GetWorkflowSpec(), &wfDefault.Spec)
	assert.NoError(err)
	assert.Equal(result.Spec, targetWf.Spec)
	assert.Equal(3, len(targetWf.Spec.Templates))
	assert.Equal("whalesay", targetWf.Spec.Entrypoint)
}

func TestJoinWorkflowMetaData(t *testing.T) {
	assert := assert.New(t)
	t.Run("WfDefaultMetaData", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
		wf1 := wfv1.MustUnmarshalWorkflow(wf)
		JoinWorkflowMetaData(&wf1.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf1.Labels, "testLabel")
		assert.Equal("default", wf1.Labels["testLabel"])
		assert.Contains(wf1.Annotations, "testAnnotation")
		assert.Equal("default", wf1.Annotations["testAnnotation"])
	})
	t.Run("WFTMetadata", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
		wf2 := wfv1.MustUnmarshalWorkflow(wf)
		JoinWorkflowMetaData(&wf2.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf2.Labels, "testLabel")
		assert.Equal("default", wf2.Labels["testLabel"])
		assert.Contains(wf2.Annotations, "testAnnotation")
		assert.Equal("default", wf2.Annotations["testAnnotation"])
	})
	t.Run("WfMetadata", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
		wf2 := wfv1.MustUnmarshalWorkflow(wf)
		wf2.Labels = map[string]string{"testLabel": "wf"}
		wf2.Annotations = map[string]string{"testAnnotation": "wf"}
		JoinWorkflowMetaData(&wf2.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf2.Labels, "testLabel")
		assert.Equal("wf", wf2.Labels["testLabel"])
		assert.Contains(wf2.Annotations, "testAnnotation")
		assert.Equal("wf", wf2.Annotations["testAnnotation"])
	})
}

var baseNilHookWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
`

var baseHookWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  hooks:
    foo:
      template: a
      expression: workflow.status == "Pending"
`

var patchNilHookWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
`

var patchHookWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  hooks:
    foo:
      template: c
      expression: workflow.status == "Pending"
    bar:
      template: b
      expression: workflow.status == "Pending"
`

func TestMergeHooks(t *testing.T) {
	t.Run("NilBaseAndNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow(patchNilHookWF)
		targetHookWf := wfv1.MustUnmarshalWorkflow(baseNilHookWF)

		err := MergeTo(patchHookWf, targetHookWf)
		assert.NoError(t, err)
		assert.Nil(t, targetHookWf.Spec.Hooks)
	})

	t.Run("NilBaseAndNotNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow(patchHookWF)
		targetHookWf := wfv1.MustUnmarshalWorkflow(baseNilHookWF)

		err := MergeTo(patchHookWf, targetHookWf)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(targetHookWf.Spec.Hooks))
		assert.Equal(t, "c", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})

	// Ensure hook bar ends up in result, but foo is unchanged
	t.Run("NotNilBaseAndPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow(patchHookWF)
		targetHookWf := wfv1.MustUnmarshalWorkflow(baseHookWF)

		err := MergeTo(patchHookWf, targetHookWf)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(targetHookWf.Spec.Hooks))
		assert.Equal(t, "a", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})
}
