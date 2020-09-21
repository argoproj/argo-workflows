package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wftClient := mocks.WorkflowTemplateServiceClient{}
	wftClient.On("DeleteWorkflowTemplate", mock.Anything, mock.Anything).Return(&workflowtemplatepkg.WorkflowTemplateDeleteResponse{}, nil)
	client.On("NewWorkflowTemplateServiceClient").Return(&wftClient)
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"workflow-template-whalesay-template"})
	output := test.ExecuteCommand(t, deleteCommand)

	assert.Contains(t, output, "workflow-template-whalesay-template")
	assert.Contains(t, output, "deleted")
}
