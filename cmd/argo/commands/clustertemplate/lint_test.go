package clustertemplate

import (
	"context"
	"io/ioutil"
	"os"
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

func TestNewLintCommand(t *testing.T) {
	err := ioutil.WriteFile("cwft.yaml", []byte(cwfts), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cwftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err = yaml.Unmarshal([]byte(cwfts), &cwftmpl)
	assert.NoError(t, err)

	cwftClient.On("LintClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(&cwftmpl, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&cwftClient)
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"cwft.yaml"})
	output := test.ExecuteCommand(t, lintCommand)
	os.Remove("cwft.yaml")
	assert.Contains(t, output, "manifests validated")
}
