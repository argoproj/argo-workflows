package template

import (
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewListCommand(t *testing.T) {
	client := clientmocks.Client{}
	wftClient := mocks.WorkflowTemplateServiceClient{}
	var wftmpl wfv1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(wft), &wftmpl)
	assert.NoError(t, err)
	wftList := wfv1.WorkflowTemplateList{
		Items: wfv1.WorkflowTemplates{wftmpl},
	}

	wftClient.On("ListWorkflowTemplates", mock.Anything, mock.Anything).Return(&wftList, nil)
	client.On("NewWorkflowTemplateServiceClient").Return(&wftClient)
	common.APIClient = &client
	listCommand := NewListCommand()
	execFunc := func() {
		err := listCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "workflow-template-whalesay-template")
}
