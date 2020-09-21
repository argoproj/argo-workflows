package archive

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	archiveworkflowpkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/pkg/apiclient/workflowarchive/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	archiveClient := mocks.ArchivedWorkflowServiceClient{}
	archiveClient.On("DeleteArchivedWorkflow", mock.Anything, mock.Anything).Return(&archiveworkflowpkg.ArchivedWorkflowDeletedResponse{}, nil)
	client.On("NewArchivedWorkflowServiceClient").Return(&archiveClient, nil)
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"hello-world"})
	output := test.ExecuteCommand(t, deleteCommand)

	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "deleted")
}
