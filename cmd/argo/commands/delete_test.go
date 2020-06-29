package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/pkg/apiclient/mocks"
	wfapi "github.com/argoproj/argo/pkg/apiclient/workflow"
)


func TestNewDeleteCommand(t *testing.T) {
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	wfDeleteRsp := wfapi.WorkflowDeleteResponse{}
	wfClient.On("DeleteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything ).Return(&wfDeleteRsp, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		err := deleteCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, output, "deleted")
	assert.Contains(t, output, "hello-world")
}

