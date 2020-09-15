package archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	"github.com/argoproj/argo/pkg/apiclient/workflowarchive/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewListCommand(t *testing.T) {
	var wfList wfv1.WorkflowList
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	assert.NoError(t, err)

	wfList.Items = wfv1.Workflows{wf}

	client := clientmocks.Client{}
	archiveClient := mocks.ArchivedWorkflowServiceClient{}
	archiveClient.On("ListArchivedWorkflows", mock.Anything, mock.Anything).Return(&wfList, nil)
	client.On("NewArchivedWorkflowServiceClient").Return(&archiveClient, nil)
	common.APIClient = &client
	listCommand := NewListCommand()
	execFunc := func() {
		err := listCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "Succeeded")
}
