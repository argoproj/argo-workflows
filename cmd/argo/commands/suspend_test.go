package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewSuspendCommand(t *testing.T) {
	client := clientmocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(workflow), &wf)
	assert.NoError(t, err)
	wfClient.On("SuspendWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	APIClient = &client
	suspendCommand := NewSuspendCommand()
	suspendCommand.SetArgs([]string{"hello-world-2xg9p"})
	execFunc := func() {
		err := suspendCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, output, "suspended")
	assert.Contains(t, output, "hello-world-2xg9p")
}
