package cron

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	cronWfClient.On("DeleteCronWorkflow", mock.Anything, mock.Anything).Return(&cronworkflow.CronWorkflowDeletedResponse{}, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)
	common.APIClient = &client
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		err := deleteCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Empty(t, output)
}
