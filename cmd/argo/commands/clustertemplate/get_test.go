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

func TestNewGetCommand(t *testing.T) {
	client := clientmocks.Client{}
	cwftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err := yaml.Unmarshal([]byte(cwfts), &cwftmpl)
	assert.NoError(t, err)

	cwftClient.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(&cwftmpl, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&cwftClient)
	common.APIClient = &client
	getCommand := NewGetCommand()
	getCommand.SetArgs([]string{"cluster-workflow-template-whalesay-template"})
	execFunc := func() {
		err := getCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
	assert.Contains(t, output, "Created")
}
