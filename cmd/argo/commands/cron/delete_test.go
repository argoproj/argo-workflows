package cron

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"

	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	cronWfClient.On("DeleteCronWorkflow", mock.Anything, mock.Anything).Return(&cronworkflow.CronWorkflowDeletedResponse{}, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)

	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"hello-world"})
	output := test.ExecuteCommand(t, deleteCommand)
	assert.Empty(t, output)
}
