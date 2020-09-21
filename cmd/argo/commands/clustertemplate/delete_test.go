package clustertemplate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	clusterworkflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
)

func TestNewDeleteCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	wftClient.On("DeleteClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(&clusterworkflowtemplatepkg.ClusterWorkflowTemplateDeleteResponse{}, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&wftClient)
	deleteCommand := NewDeleteCommand()
	deleteCommand.SetArgs([]string{"cluster-workflow-template-whalesay-template"})
	output := test.ExecuteCommand(t, deleteCommand)

	assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
	assert.Contains(t, output, "deleted")
}
