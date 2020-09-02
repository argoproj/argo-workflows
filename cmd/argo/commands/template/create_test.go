package template

import (
	"sigs.k8s.io/yaml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate/mocks"
	"github.com/argoproj/argo/cmd/argo/commands"
)

var wft =`
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
spec:
  entrypoint: whalesay-template
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestNewCreateCommand(t *testing.T) {
	client := clientmocks.Client{}
	wftClient := mocks.WorkflowTemplateServiceClient{}
	var wftmpl wfv1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(wft), &wftmpl)
	assert.NoError(t, err)

	wftClient.On("CreateWorkflowTemplate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wft, nil)
	client.On("NewWorkflowServiceClient").Return(&wftClient)
	commands.APIClient = &client
	createCommand := NewCreateCommand()
	execFunc := func() {
		err := createCommand.Execute()
		assert.NoError(t, err)
	}
	output := commands.CaptureOutput(execFunc)
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "Succeeded")
}
