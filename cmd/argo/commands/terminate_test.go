package commands

import (
"context"
"testing"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/mock"
"sigs.k8s.io/yaml"

"github.com/argoproj/argo/pkg/apiclient/mocks"
wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewTerminateCommand(t *testing.T) {
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(workflow), &wf)
	assert.NoError(t, err)
	wfClient.On("TerminateWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	terminateCommand := NewTerminateCommand()
	terminateCommand.SetArgs([]string{"hello-world-2xg9p"})
	execFunc := func() {
		err := terminateCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, output, "terminated")
	assert.Contains(t, output, "hello-world")
}
