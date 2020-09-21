package commands

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflow/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestLintCommand(t *testing.T) {
	var wf wfv1.Workflow
	//err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	err := ioutil.WriteFile("wf.yaml", []byte(wfWithStatus), 0644)
	defer os.Remove("wf.yaml")
	assert.NoError(t, err)
	client := clientmocks.Client{}
	cmdcommon.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wfClient := mocks.WorkflowServiceClient{}
	wfClient.On("LintWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"wf.yaml"})
	output := test.ExecuteCommand(t, lintCommand)
	assert.Contains(t, output, "wf.yaml is valid\nWorkflow manifests validated\n")
}
