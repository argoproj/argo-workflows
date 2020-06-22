package common

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"

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
	templatedCronWfString := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
spec:
  schedule: "* * * * *"
  template:
    metadata:
      labels:
        label1: value1
      annotations:
        annotation2: value2
    spec:
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
	wf, err := ConvertCronWorkflowToWorkflow(&cronWf)
	assert.NoError(t, err)
	wfString, err := yaml.Marshal(wf)
	assert.NoError(t, err)
	assert.Equal(t, expectedWf, string(wfString))

	cronWf = v1alpha1.CronWorkflow{}
	err = yaml.Unmarshal([]byte(templatedCronWfString), &cronWf)
	assert.NoError(t, err)
	wf, err = ConvertCronWorkflowToWorkflow(&cronWf)
	assert.NoError(t, err)
	wfString, err = yaml.Marshal(wf)
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

	templatedCronWfInstanceIdString := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
  labels:
    workflows.argoproj.io/controller-instanceid: test-controller
spec:
  schedule: "* * * * *"
  template:
    metadata:
      labels:
        label1: value1
      annotations:
        annotation2: value2
    spec:
      entrypoint: whalesay
      templates:
        - name: whalesay
          container:
            image: docker/whalesay:latest
            command: [cowsay]
            args: ["hello world"]
`

	cronWf = v1alpha1.CronWorkflow{}
	err = yaml.Unmarshal([]byte(cronWfInstanceIdString), &cronWf)
	assert.NoError(t, err)
	wf, err = ConvertCronWorkflowToWorkflow(&cronWf)
	assert.NoError(t, err)
	if assert.Contains(t, wf.GetLabels(), LabelKeyControllerInstanceID) {
		assert.Equal(t, wf.GetLabels()[LabelKeyControllerInstanceID], "test-controller")
	}

	cronWf = v1alpha1.CronWorkflow{}
	err = yaml.Unmarshal([]byte(templatedCronWfInstanceIdString), &cronWf)
	assert.NoError(t, err)
	wf, err = ConvertCronWorkflowToWorkflow(&cronWf)
	assert.NoError(t, err)
	if assert.Contains(t, wf.GetLabels(), LabelKeyControllerInstanceID) {
		assert.Equal(t, wf.GetLabels()[LabelKeyControllerInstanceID], "test-controller")
	}
}

func TestCannotUseBothTemplatedAndNot(t *testing.T) {
	invalid := `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
spec:
  schedule: "* * * * *"
  workflowSpec:
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["hello world"]
  template:
    metadata:
      labels:
        label1: value1
      annotations:
        annotation2: value2
    spec:
      entrypoint: whalesay
      templates:
        - name: whalesay
          container:
            image: docker/whalesay:latest
            command: [cowsay]
            args: ["hello world"]
`
	var cronWf v1alpha1.CronWorkflow
	err := yaml.Unmarshal([]byte(invalid), &cronWf)
	assert.NoError(t, err)
	_, err = ConvertCronWorkflowToWorkflow(&cronWf)
	assert.EqualError(t, err, "cannot use both CronWorkflow.spec.workflowSpec and CronWorkflow.spec.template to specify a Workflow to run. Please use only CronWorkflow.spec.template instead")
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
	wf := NewWorkflowFromWorkflowTemplate(wfTmpl.Name, false)
	assert.NotNil(t, wf)
	assert.NotNil(t, wf.Spec.WorkflowTemplateRef)
	assert.Equal(t, wfTmpl.Name, wf.Spec.WorkflowTemplateRef.Name)
	assert.Equal(t, false, wf.Spec.WorkflowTemplateRef.ClusterScope)
}
