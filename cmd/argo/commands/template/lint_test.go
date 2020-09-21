package template

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/argoproj/argo/pkg/apiclient"

	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewLintCommand(t *testing.T) {
	err := ioutil.WriteFile("wft.yaml", []byte(wft), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wftClient := mocks.WorkflowTemplateServiceClient{}
	var wftmpl wfv1.WorkflowTemplate
	err = yaml.Unmarshal([]byte(wft), &wftmpl)
	assert.NoError(t, err)

	wftClient.On("LintWorkflowTemplate", mock.Anything, mock.Anything).Return(&wftmpl, nil)
	client.On("NewWorkflowTemplateServiceClient").Return(&wftClient)
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"wft.yaml"})
	output := test.ExecuteCommand(t, lintCommand)
	os.Remove("wft.yaml")
	assert.Contains(t, output, "manifests validated")
}
