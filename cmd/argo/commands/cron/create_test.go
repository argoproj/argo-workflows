package cron

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var cronwf = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world
  namespace: default
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
	defer os.Remove("cronwf.yaml")
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	var cronWf wfv1.CronWorkflow
	err = yaml.Unmarshal([]byte(cronwf), &cronWf)
	assert.NoError(t, err)
	cronWfClient.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(&cronWf, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)
	createCommand := NewCreateCommand()
	createCommand.SetArgs([]string{"cronwf.yaml"})
	output := test.ExecuteCommand(t, createCommand)
	fmt.Println(output)
	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "Created")
	assert.Contains(t, output, "* * * * *")
}
