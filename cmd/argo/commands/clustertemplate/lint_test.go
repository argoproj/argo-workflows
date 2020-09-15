package clustertemplate

import (
	"io/ioutil"
	"os"
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

func TestNewLintCommand(t *testing.T) {
	err := ioutil.WriteFile("cwft.yaml", []byte(cwfts), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	cwftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err = yaml.Unmarshal([]byte(cwfts), &cwftmpl)
	assert.NoError(t, err)

	cwftClient.On("LintClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(&cwftmpl, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&cwftClient)
	common.APIClient = &client
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"cwft.yaml"})
	execFunc := func() {
		err := lintCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	os.Remove("cwft.yaml")
	assert.Contains(t, output, "manifests validated")
}
