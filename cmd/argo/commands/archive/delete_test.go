package archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	archiveworkflowpkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/pkg/apiclient/workflowarchive/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	archiveClient := mocks.ArchivedWorkflowServiceClient{}
	archiveClient.On("DeleteArchivedWorkflow", mock.Anything, mock.Anything).Return(&archiveworkflowpkg.ArchivedWorkflowDeletedResponse{}, nil)
	client.On("NewArchivedWorkflowServiceClient").Return(&archiveClient, nil)
	common.APIClient = &client
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		err := deleteCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)

	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "deleted")
}
