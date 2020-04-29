package common

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
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

	var cronWf wfv1.CronWorkflow
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
    templates:
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
}

const workflowTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
  labels:
    argo-e2e: true
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: hello world
  workflowMetadata:
    labels: 
      my-label: my-value
    annotations:
      my-annotation: my-value
  templates:
    - name: whalesay-template
      inputs:
        parameters:
          - name: message
      container:
        image: cowsay:v1
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
        imagePullPolicy: IfNotPresent
`

func TestConvertWorkflowTemplateToWorkflow(t *testing.T) {
	var wfTmpl wfv1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(workflowTmpl), &wfTmpl)
	if err != nil {
		panic(err)
	}
	wf := ConvertWorkflowTemplateToWorkflow(&wfTmpl)
	if assert.NotNil(t, wf) {
		assert.Equal(t, wf.Labels[LabelKeyWorkflowTemplate], wfTmpl.Name)
		assert.Equal(t, wf.GenerateName, wfTmpl.Name+"-")
		if assert.NotNil(t, wf) {
			if assert.NotNil(t, wf.Labels) {
				assert.Contains(t, wf.Labels, "my-label")
				assert.Contains(t, wf.Annotations, "my-annotation")
			}
		}
	}
}

func TestConvertClusterWorkflowTemplateToWorkflow(t *testing.T) {
	wfTmpl := &wfv1.ClusterWorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-cwft",
		},
		Spec: wfv1.WorkflowTemplateSpec{
			WorkflowMetadata: &metav1.ObjectMeta{
				Labels:      map[string]string{"my-label": "my-value"},
				Annotations: map[string]string{"my-annotation": "my-value"},
			},
		},
	}
	wf := ConvertClusterWorkflowTemplateToWorkflow(wfTmpl)
	if assert.NotNil(t, wf) {
		assert.Equal(t, wf.GenerateName, "my-cwft-")
		if assert.NotNil(t, wf.Labels) {
			assert.Contains(t, wf.Labels, "my-label")
			assert.Contains(t, wf.Annotations, "my-annotation")
		}
	}
}
