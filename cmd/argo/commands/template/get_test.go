package template

import (
	"context"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewGetCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wftClient := mocks.WorkflowTemplateServiceClient{}
	var wftmpl wfv1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(wft), &wftmpl)
	assert.NoError(t, err)

	wftClient.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(&wftmpl, nil)
	client.On("NewWorkflowTemplateServiceClient").Return(&wftClient)
	getCommand := NewGetCommand()
	getCommand.SetArgs([]string{"workflow-template-whalesay-template"})
	output := test.ExecuteCommand(t, getCommand)
	assert.Contains(t, output, "workflow-template-whalesay-template")
	assert.Contains(t, output, "Created")
}
