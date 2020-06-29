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

func TestNewResubmitCommand(t *testing.T) {
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	assert.NoError(t, err)
	wfClient.On("ResubmitWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything ).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	resumeCommand := NewResubmitCommand()
	resumeCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		err := resumeCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, output, "Name:")
	assert.Contains(t, output, "Namespace:")
	assert.Contains(t, output, "ServiceAccount:")
	assert.Contains(t, output, "Status:")
	assert.Contains(t, output, "Created:")
}
