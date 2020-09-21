package cron

import (
	"context"
	"os"
	"testing"

	"github.com/argoproj/argo/pkg/apiclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var cronWfWithStatus = `
metadata:
  creationTimestamp: "2020-09-14T23:11:06Z"
  generation: 1
  managedFields:
  - apiVersion: argoproj.io/v1alpha1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        .: {}
        f:concurrencyPolicy: {}
        f:failedJobsHistoryLimit: {}
        f:schedule: {}
        f:startingDeadlineSeconds: {}
        f:successfulJobsHistoryLimit: {}
        f:timezone: {}
        f:workflowSpec:
          .: {}
          f:arguments: {}
          f:entrypoint: {}
          f:templates: {}
      f:status: {}
    manager: argo
    operation: Update
    time: "2020-09-14T23:11:06Z"
  name: hello-world
  namespace: argo
  resourceVersion: "441082"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/cronworkflows/hello-world
  uid: 000616a6-a7d8-4407-a423-a036b28a8c14
spec:
  concurrencyPolicy: Replace
  failedJobsHistoryLimit: 4
  schedule: '* * * * *'
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  timezone: America/Los_Angeles
  workflowSpec:
    arguments: {}
    entrypoint: whalesay
    templates:
    - arguments: {}
      container:
        args:
        - "\U0001F553 hello world"
        command:
        - cowsay
        image: docker/whalesay:latest
        name: ""
        resources: {}
      inputs: {}
      metadata: {}
      name: whalesay
      outputs: {}
status: {}
`

func TestNewResumeCommand(t *testing.T) {
	client := clientmocks.Client{}
	cmdcommon.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cronwfClient := mocks.CronWorkflowServiceClient{}
	var cronWf wfv1.CronWorkflow
	err := yaml.Unmarshal([]byte(cronWfWithStatus), &cronWf)
	assert.NoError(t, err)
	cronwfClient.On("GetCronWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&cronWf, nil)
	cronwfClient.On("UpdateCronWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&cronWf, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronwfClient)
	resumeCommand := NewResumeCommand()
	resumeCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		os.Setenv("ARGO_NAMESPACE", "default")
		err := resumeCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Contains(t, output, "CronWorkflow 'hello-world' resumed")
}
