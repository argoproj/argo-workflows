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

func TestNewResumeCommand(t *testing.T) {
	client := clientmocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	assert.NoError(t, err)
	wfClient.On("ResumeWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	APIClient = &client
	resumeCommand := NewResumeCommand()
	resumeCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		err := resumeCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, output, wf.Name+" resumed")
}
