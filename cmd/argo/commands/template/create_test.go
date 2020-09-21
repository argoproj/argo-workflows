package template

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
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var wft = `
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

	wftClient.On("CreateWorkflowTemplate", mock.Anything, mock.Anything).Return(&wftmpl, nil)
	client.On("NewWorkflowTemplateServiceClient").Return(&wftClient)
	createCommand := NewCreateCommand()
	createCommand.SetArgs([]string{"wft.yaml"})
	output := test.ExecuteCommand(t, createCommand)
	os.Remove("wft.yaml")
	assert.Contains(t, output, "workflow-template-whalesay-template")
	assert.Contains(t, output, "Created")
}
