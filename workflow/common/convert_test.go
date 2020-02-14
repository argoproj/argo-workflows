package common

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
	"testing"
)

const cronWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic
  labels:
    argo-e2e-cron: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: python:alpine3.6
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]
`

func TestConvertCronWorkflowToWorkflow(t *testing.T) {

	var cronWf v1alpha1.CronWorkflow
	err := yaml.Unmarshal([]byte(cronWorkflow), &cronWf)
	if err != nil {
		panic(err)
	}
	wf := ConvertCronWorkflowToWorkflow(&cronWf)
	assert.NotNil(t, wf)
	assert.Equal(t, wf.Labels[LabelKeyCronWorkflow], cronWf.Name)
	assert.Equal(t, wf.GenerateName, cronWf.Name+"-")

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
        image: cowsay:v1
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
	wf := ConvertWorkflowTemplateToWorkflow(&wfTmpl)
	assert.NotNil(t, wf)
	assert.Equal(t, wf.Labels[LabelKeyWorkflowTemplate], wfTmpl.Name)
	assert.Equal(t, wf.GenerateName, wfTmpl.Name+"-")
}
