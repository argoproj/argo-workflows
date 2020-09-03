package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	wftClient := mocks.WorkflowTemplateServiceClient{}
	wftClient.On("DeleteWorkflowTemplate", mock.Anything, mock.Anything).Return(&workflowtemplatepkg.WorkflowTemplateDeleteResponse{}, nil)
	client.On("NewWorkflowTemplateServiceClient").Return(&wftClient)
	common.APIClient = &client
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"workflow-template-whalesay-template"})
	execFunc := func() {
		err := deleteCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)

	assert.Contains(t, output, "workflow-template-whalesay-template")
	assert.Contains(t, output, "deleted")
}
