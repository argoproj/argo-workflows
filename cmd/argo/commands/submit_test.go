package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const workflow string = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-test
  namespace: default
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func TestSubmitFromResource(t *testing.T) {
	client := clientmocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	wfClient.On("SubmitWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	cmdcommon.APIClient = &client
	output := test.CaptureOutput(func() {
		submitWorkflowFromResource(context.TODO(), &wfClient, "default", "workflowtemplate/test", &wfv1.SubmitOpts{}, &cliSubmitOpts{})
	})
	assert.Contains(t, output, "Created:")
}

func TestSubmitWorkflows(t *testing.T) {
	client := clientmocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	wfClient.On("CreateWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	cmdcommon.APIClient = &client

	err := yaml.Unmarshal([]byte(workflow), &wf)
	assert.NoError(t, err)
	workflows := []wfv1.Workflow{wf}
	output := test.CaptureOutput(func() {
		submitWorkflows(context.TODO(), &wfClient, "default", workflows, &wfv1.SubmitOpts{}, &cliSubmitOpts{})
	})
	fmt.Println(output)
	assert.Contains(t, output, "Created:")
}
