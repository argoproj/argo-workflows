package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
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
  creationTimestamp: null
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
  - arguments: {}
    container:
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
	err := yaml.Unmarshal([]byte(cronWfString), &cronWf)
	assert.NoError(t, err)
	wf := ConvertCronWorkflowToWorkflow(&cronWf)
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
		assert.Equal(t, wf.GetLabels()[LabelKeyControllerInstanceID], "test-controller")
	}

	err = yaml.Unmarshal([]byte(cronWfInstanceIdString), &cronWf)
	assert.NoError(t, err)
	wf = ConvertCronWorkflowToWorkflowWithName(&cronWf, "test-name")
	assert.Equal(t, "test-name", wf.Name)
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
        image: argoproj/argosay:v1
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
        imagePullPolicy: IfNotPresent
`

func TestConvertWorkflowTemplateToWorkflow(t *testing.T) {
	var wfTmpl v1alpha1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(workflowTmpl), &wfTmpl)
	if err != nil {
		panic(err)
	}
	t.Run("ConvertWorkflowFromWFT", func(t *testing.T) {
		wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, wfTmpl.Spec.WorkflowMetadata, false)
		assert.NotNil(t, wf)
		assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/workflow-template"])
		assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
		assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
		assert.False(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
		assert.Contains(t, wf.Labels, "label1")
		assert.Contains(t, wf.Annotations, "annotation1")
	})
	t.Run("ConvertWorkflowFromWFTWithNilWorkflowMetadata", func(t *testing.T) {

		wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, nil, false)
		assert.NotNil(t, wf)
		assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/workflow-template"])
		assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
		assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
		assert.False(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
	})
	t.Run("ConvertWorkflowFromWFTWithNilWorkflowMetadataLabels", func(t *testing.T) {
		wfMetadata := &metav1.ObjectMeta{
			Labels:      nil,
			Annotations: nil,
		}
		wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, wfMetadata, false)
		assert.NotNil(t, wf)
		assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/workflow-template"])
		assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
		assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
		assert.False(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
	})

}

func TestConvertClusterWorkflowTemplateToWorkflow(t *testing.T) {
	var wfTmpl v1alpha1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(workflowTmpl), &wfTmpl)
	if err != nil {
		panic(err)
	}
	wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, wfTmpl.Spec.WorkflowMetadata, true)
	assert.NotNil(t, wf)
	assert.Equal(t, "workflow-template-whalesay-template", wf.Labels["workflows.argoproj.io/cluster-workflow-template"])
	assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
	assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
	assert.True(t, wf.Spec.WorkflowTemplateRef.ClusterScope)
	assert.Contains(t, wf.Labels, "label1")
	assert.Contains(t, wf.Annotations, "annotation1")
}
