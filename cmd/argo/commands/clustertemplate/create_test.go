package clustertemplate

import (
	"io/ioutil"
	"os"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const cwfts = `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-whalesay-template
spec:
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
---
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-whalesay-template
spec:
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

func TestUnmarshalCWFT(t *testing.T) {

	clusterwfts, err := unmarshalClusterWorkflowTemplates([]byte(cwfts), false)
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(clusterwfts))
	}
}

func TestNewCreateCommand(t *testing.T) {
	err := ioutil.WriteFile("cwft.yaml", []byte(cwfts), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	wftClient := mocks.ClusterWorkflowTemplateServiceClient{}
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err = yaml.Unmarshal([]byte(cwfts), &cwftmpl)
	assert.NoError(t, err)

	wftClient.On("CreateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(&cwftmpl, nil)
	client.On("NewClusterWorkflowTemplateServiceClient").Return(&wftClient)
	common.APIClient = &client
	createCommand := NewCreateCommand()
	createCommand.SetArgs([]string{"cwft.yaml"})
	execFunc := func() {
		err := createCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	os.Remove("cwft.yaml")
	assert.Contains(t, output, "cluster-workflow-template-whalesay-template")
	assert.Contains(t, output, "Created")
}
