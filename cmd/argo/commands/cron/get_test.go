package cron

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var invalidCwf = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  creationTimestamp: "2020-05-19T16:47:25Z"
  generation: 98
  name: wonderful-tiger
  namespace: argo
  resourceVersion: "465179"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/cronworkflows/wonderful-tiger
  uid: c4ea2e84-ec58-4638-bf1d-5d543e7cc86a
spec:
  schedule: '* * * * *'
  workflowSpec:
    arguments: {}
    entrypoint: argosay
    templates:
    - arguments: {}
      container:
        args:
        - echo
        - hello argo!
        command:
        - /argosay
        image: argoproj/argosay:v2
        name: main
        resources: {}
      inputs: {}
      metadata: {}
      name: argosay!3
      outputs: {}
status:
  conditions:
  - message: 'Failed to submit Workflow: spec.templates[0].name: ''argosay!3'' is
      invalid: name must consist of alpha-numeric characters or ''-'', and must start
      with an alpha-numeric character (e.g. My-name1-2, 123-NAME)'
    status: "True"
    type: SubmissionError
  lastScheduledTime: "2020-05-19T17:56:00Z"
`

var expectedOut = `
Conditions:                    
✖ SubmissionError              Failed to submit Workflow: spec.templates[0].name: 'argosay!3' is invalid: name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character (e.g. My-name1-2, 123-NAME)`

func TestPrintCronWorkflow(t *testing.T) {
	var cronWf v1alpha1.CronWorkflow
	err := yaml.Unmarshal([]byte(invalidCwf), &cronWf)
	if assert.NoError(t, err) {
		out := getCronWorkflowGet(&cronWf)
		assert.Contains(t, out, expectedOut)
	}
}

func TestNextRuntime(t *testing.T) {
	var cronWf v1alpha1.CronWorkflow
	err := yaml.Unmarshal([]byte(invalidCwf), &cronWf)
	if assert.NoError(t, err) {
		next, err := GetNextRuntime(&cronWf)
		if assert.NoError(t, err) {
			assert.LessOrEqual(t, next.Unix(), time.Now().Add(1*time.Minute).Unix())
			assert.Greater(t, next.Unix(), time.Now().Unix())
		}
	}
}

func TestNewGetCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	var cronWfObj v1alpha1.CronWorkflow
	err := yaml.Unmarshal([]byte(cronwf), &cronWfObj)
	assert.NoError(t, err)
	cronWfClient.On("GetCronWorkflow", mock.Anything, mock.Anything).Return(&cronWfObj, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)
	getCommand := NewGetCommand()
	getCommand.SetArgs([]string{"hello-world"})
	output := test.ExecuteCommand(t, getCommand)
	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "* * * * *")
}
