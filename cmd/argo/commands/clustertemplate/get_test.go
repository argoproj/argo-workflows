package clustertemplate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewGetCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cwftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err := yaml.Unmarshal([]byte(cwfts), &cwftmpl)
	assert.NoError(t, err)

	cwftClient.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(&cwftmpl, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&cwftClient)
	getCommand := NewGetCommand()
	getCommand.SetArgs([]string{"cluster-workflow-template-whalesay-template"})
	output := test.ExecuteCommand(t, getCommand)
	assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
	assert.Contains(t, output, "Created")
}
