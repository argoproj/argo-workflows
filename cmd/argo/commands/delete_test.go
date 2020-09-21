package commands

import (
	"context"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfapi "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apiclient/workflow/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wfClient := mocks.WorkflowServiceClient{}
	wfDeleteRsp := wfapi.WorkflowDeleteResponse{}
	wfClient.On("DeleteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wfDeleteRsp, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)

	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"hello-world"})
	output := test.ExecuteCommand(t, deleteCommand)
	assert.Contains(t, output, "deleted")
	assert.Contains(t, output, "hello-world")
}
