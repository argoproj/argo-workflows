package commands

import (
	"context"
	"github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"os"
	"testing"
)

func TestLintCommand(t *testing.T) {
	var wf wfv1.Workflow
	//err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	err := ioutil.WriteFile("wf.yaml", []byte(wfWithStatus),0644)
	assert.NoError(t, err)
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	wfClient.On("LintWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"wf.yaml"})
	execFunc := func() {
		err := lintCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, "wf.yaml is valid\nWorkflow manifests validated\n", output)
	err = os.Remove("wf.yaml")
	assert.NoError(t, err)
}
