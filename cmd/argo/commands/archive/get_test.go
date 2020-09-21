package archive

import (
	"context"
	"testing"

	"github.com/argoproj/argo/pkg/apiclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflowarchive/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var wfWithStatus = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-06-24T22:53:35Z"
  generation: 6
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: hello-world
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
  - status: "True"
    type: Completed
  finishedAt: "2020-06-24T22:53:41Z"
  nodes:
    hello-world-2xg9p:
      displayName: hello-world-2xg9p
      finishedAt: "2020-06-24T22:53:39Z"
      id: hello-world-2xg9p
      name: hello-world-2xg9p
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-06-24T22:53:35Z"
      templateName: whalesay
      templateScope: local/hello-world-2xg9p
      type: Pod
  phase: Succeeded
  resourcesDuration:
    cpu: 3
    memory: 0
  startedAt: "2020-06-24T22:53:35Z"
`

func TestNewGetCommand(t *testing.T) {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	archiveClient := mocks.ArchivedWorkflowServiceClient{}
	archiveClient.On("GetArchivedWorkflow", mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewArchivedWorkflowServiceClient").Return(&archiveClient, nil)
	getCommand := NewGetCommand()
	getCommand.SetArgs([]string{"hello-World"})
	output := test.ExecuteCommand(t, getCommand)

	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "Succeeded")
}
