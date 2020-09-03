package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfapi "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apiclient/workflow/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	wfDeleteRsp := wfapi.WorkflowDeleteResponse{}
	wfClient.On("DeleteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wfDeleteRsp, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	common.APIClient = &client

	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		err := deleteCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Contains(t, output, "deleted")
	assert.Contains(t, output, "hello-world")
}
