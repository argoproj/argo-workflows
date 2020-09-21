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

func TestNewSuspend(t *testing.T) {
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
	suspendCommand := NewSuspendCommand()
	suspendCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		os.Setenv("ARGO_NAMESPACE", "default")
		err := suspendCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Contains(t, output, "CronWorkflow 'hello-world' suspended")
}
