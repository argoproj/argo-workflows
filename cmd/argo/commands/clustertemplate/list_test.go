package clustertemplate

import (
	"testing"

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
	wftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err := yaml.Unmarshal([]byte(cwfts), &cwftmpl)
	assert.NoError(t, err)
	cwftList := wfv1.ClusterWorkflowTemplateList{
		Items: wfv1.ClusterWorkflowTemplates{cwftmpl},
	}

	wftClient.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(&cwftList, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&wftClient)
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
