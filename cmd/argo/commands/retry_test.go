package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var wfWithError = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-06-24T22:53:35Z"
  generateName: hello-world-
  generation: 6
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: hello-world-2xg9p
  namespace: default
  resourceVersion: "1110858"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-2xg9p
  uid: 8c25e2e7-6b35-4a49-a667-87b4cd1afa3c
spec:
  arguments: {}
  entrypoint: whalesay
  templates:
  - arguments: {}
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  conditions:
  - status: "False"
    type: Failed
  finishedAt: "2020-06-24T22:53:41Z"
  nodes:
    hello-world-2xg9p:
      displayName: hello-world-2xg9p
      finishedAt: "2020-06-24T22:53:39Z"
      id: hello-world-2xg9p
      name: hello-world-2xg9p
      outputs:
        exitCode: "0"
      phase: Failed
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-06-24T22:53:35Z"
      templateName: whalesay
      templateScope: local/hello-world-2xg9p
      type: Pod
  phase: Failed
  resourcesDuration:
    cpu: 3
    memory: 0
  startedAt: "2020-06-24T22:53:35Z"
`

func TestNewRetryCommand(t *testing.T) {
	client := clientmocks.Client{}
	cmdcommon.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithError), &wf)
	assert.NoError(t, err)
	wfClient.On("RetryWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	retryCommand := NewRetryCommand()
	retryCommand.SetArgs([]string{"hello-world]"})
	output := test.ExecuteCommand(t, retryCommand)
	assert.Contains(t, output, "Name:")
	assert.Contains(t, output, "hello-world")
}
