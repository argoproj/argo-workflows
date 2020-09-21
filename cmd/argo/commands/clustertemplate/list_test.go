package clustertemplate

import (
	"context"
	"testing"

	"github.com/argoproj/argo/pkg/apiclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewListCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err := yaml.Unmarshal([]byte(cwfts), &cwftmpl)
	assert.NoError(t, err)
	cwftList := wfv1.ClusterWorkflowTemplateList{
		Items: wfv1.ClusterWorkflowTemplates{cwftmpl},
	}

	wftClient.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(&cwftList, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&wftClient)
	listCommand := NewListCommand()
	output := test.ExecuteCommand(t, listCommand)
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "workflow-template-whalesay-template")
}
