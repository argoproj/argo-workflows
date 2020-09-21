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

func TestNewStopCommand(t *testing.T) {
	client := clientmocks.Client{}
	cmdcommon.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(workflow), &wf)
	assert.NoError(t, err)
	wfClient.On("StopWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	stopCommand := NewStopCommand()
	stopCommand.SetArgs([]string{"hello-world-test"})
	output := test.ExecuteCommand(t, stopCommand)
	assert.Contains(t, output, "stopped")
	assert.Contains(t, output, "hello-world-test")
}
