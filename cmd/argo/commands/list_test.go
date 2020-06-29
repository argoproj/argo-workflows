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

func TestNewListCommand(t *testing.T) {
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	var wfList wfv1.WorkflowList
	var wf, wf1 wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(workflow), &wf1)
	assert.NoError(t, err)
	wfList.Items = wfv1.Workflows{wf, wf1}
	wfClient.On("ListWorkflows", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wfList, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	listCommand := NewListCommand()
	execFunc := func() {
		err := listCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "Succeeded")
}
