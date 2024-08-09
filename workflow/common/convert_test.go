package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestConvertCronWorkflowToWorkflow(t *testing.T) {
	cronWfString := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
spec:
  schedule: "* * * * *"
  workflowMetadata:
    labels:
      label1: value1
    annotations:
      annotation2: value2
    finalizers:
      - finalizer1
  workflowSpec:
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["hello world"]
`
	expectedWf := `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    annotation2: value2
    workflows.argoproj.io/scheduled-time: "2021-02-19T10:29:05-08:00"
  creationTimestamp: null
  finalizers:
  - finalizer1
  generateName: hello-world-
  labels:
    label1: value1
    workflows.argoproj.io/cron-workflow: hello-world
  ownerReferences:
  - apiVersion: argoproj.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: CronWorkflow
    name: hello-world
    uid: ""
spec:
  arguments: {}
  entrypoint: whalesay
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  finishedAt: null
  startedAt: null
`

	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal([]byte(cronWfString), &cronWf)
	wf := ConvertCronWorkflowToWorkflow(&cronWf)
	wf.GetAnnotations()[AnnotationKeyCronWfScheduledTime] = "2021-02-19T10:29:05-08:00"
	wfString, err := yaml.Marshal(wf)
	assert.NoError(t, err)
	assert.Equal(t, expectedWf, string(wfString))

	cronWfInstanceIdString := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
  labels:
    workflows.argoproj.io/controller-instanceid: test-controller
spec:
  schedule: "* * * * *"
  workflowMetadata:
    labels:
      label1: value1
    annotations:
      annotation2: value2
  workflowSpec:
    entrypoint: whalesay
    templateRef:
      - name: whalesay
        container:
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["hello world"]
`

	err = yaml.Unmarshal([]byte(cronWfInstanceIdString), &cronWf)
	assert.NoError(t, err)
	wf = ConvertCronWorkflowToWorkflow(&cronWf)
	if assert.Contains(t, wf.GetLabels(), LabelKeyControllerInstanceID) {
		assert.Equal(t, "test-controller", wf.GetLabels()[LabelKeyControllerInstanceID])
	}

	err = yaml.Unmarshal([]byte(cronWfInstanceIdString), &cronWf)
	assert.NoError(t, err)
	scheduledTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05-07:00")
	assert.NoError(t, err)
	wf = ConvertCronWorkflowToWorkflowWithProperties(&cronWf, "test-name", scheduledTime)
	assert.Equal(t, "test-name", wf.Name)
	assert.Len(t, wf.GetAnnotations(), 2)
	assert.NotEmpty(t, wf.GetAnnotations()[AnnotationKeyCronWfScheduledTime])
}

const workflowTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
  labels:
    argo-e2e: true
spec:
  workflowMetadata:
    labels:
      label1: value1
    annotations:
      annotation1: value1
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
        image: argoproj/argosay:v2
        args: [echo, "{{inputs.parameters.message}}"]
`

func TestConvertWorkflowTemplateToWorkflow(t *testing.T) {
	var wfTmpl v1alpha1.WorkflowTemplate
	v1alpha1.MustUnmarshal([]byte(workflowTmpl), &wfTmpl)
	t.Run("ConvertWorkflowFromWFT", func(t *testing.T) {
		wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, false)
		assert.NotNil(t, wf)
		assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/workflow-template"])
		assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
		assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
		assert.False(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
	})
	t.Run("ConvertWorkflowFromWFTWithNilWorkflowMetadata", func(t *testing.T) {
		wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, false)
		assert.NotNil(t, wf)
		assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/workflow-template"])
		assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
		assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
		assert.False(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
	})
	t.Run("ConvertWorkflowFromWFTWithNilWorkflowMetadataLabels", func(t *testing.T) {
		wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, false)
		assert.NotNil(t, wf)
		assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/workflow-template"])
		assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
		assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
		assert.False(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
	})
}

func TestConvertClusterWorkflowTemplateToWorkflow(t *testing.T) {
	var wfTmpl v1alpha1.WorkflowTemplate
	v1alpha1.MustUnmarshal([]byte(workflowTmpl), &wfTmpl)
	wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, true)
	assert.NotNil(t, wf)
	assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/cluster-workflow-template"])
	assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
	assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
	assert.True(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
}
