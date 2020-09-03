package template

import (
	"io/ioutil"
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

func TestNewLintCommand(t *testing.T) {
	err := ioutil.WriteFile("wf.yaml", []byte(wft), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	wftClient := mocks.WorkflowTemplateServiceClient{}
	var wftmpl wfv1.WorkflowTemplate
	err = yaml.Unmarshal([]byte(wft), &wftmpl)
	assert.NoError(t, err)

	wftClient.On("LintWorkflowTemplate", mock.Anything, mock.Anything).Return(&wftmpl, nil)
	client.On("NewWorkflowTemplateServiceClient").Return(&wftClient)
	common.APIClient = &client
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"wf.yaml"})
	execFunc := func() {
		err := lintCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Contains(t, output, "manifests validated")
}
