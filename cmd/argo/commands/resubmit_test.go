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

func TestNewResubmitCommand(t *testing.T) {
	client := clientmocks.Client{}
	cmdcommon.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	assert.NoError(t, err)
	wfClient.On("ResubmitWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	resumeCommand := NewResubmitCommand()
	resumeCommand.SetArgs([]string{"hello-world"})
	output := test.ExecuteCommand(t, resumeCommand)
	assert.Contains(t, output, "Name:")
	assert.Contains(t, output, "Namespace:")
	assert.Contains(t, output, "ServiceAccount:")
	assert.Contains(t, output, "Status:")
	assert.Contains(t, output, "Created:")
}
