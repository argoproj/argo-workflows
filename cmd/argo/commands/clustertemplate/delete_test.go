package clustertemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clusterworkflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	wftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	wftClient.On("DeleteClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(&clusterworkflowtemplatepkg.ClusterWorkflowTemplateDeleteResponse{}, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&wftClient)
	common.APIClient = &client
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"cluster-workflow-template-whalesay-template"})
	execFunc := func() {
		err := deleteCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)

	assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
	assert.Contains(t, output, "deleted")
}
