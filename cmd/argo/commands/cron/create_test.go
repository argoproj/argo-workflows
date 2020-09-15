package cron

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var cronwf = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
spec:
  schedule: "* * * * *"
  timezone: "America/Los_Angeles"   # Default to local machine timezone
  startingDeadlineSeconds: 0
  concurrencyPolicy: "Replace"      # Default to "Allow"
  successfulJobsHistoryLimit: 4     # Default 3
  failedJobsHistoryLimit: 4         # Default 1
  suspend: false                    # Set to "true" to suspend scheduling
  workflowSpec:
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["🕓 hello world"]
`

func TestNewCreateCommand(t *testing.T) {
	err := ioutil.WriteFile("cronwf.yaml", []byte(cronwf), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	var wftmpl wfv1.CronWorkflow
	err = yaml.Unmarshal([]byte(cronwf), &wftmpl)
	assert.NoError(t, err)

	cronWfClient.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(&wftmpl, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)
	common.APIClient = &client
	createCommand := NewCreateCommand()
	createCommand.SetArgs([]string{"cronwf.yaml"})
	execFunc := func() {
		err := createCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	os.Remove("cronwf.yaml")
	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "Created")
	assert.Contains(t, output, "* * * * *")
}
