package cron

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewListCommand(t *testing.T) {
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	var cronWfObj wfv1.CronWorkflow
	err := yaml.Unmarshal([]byte(cronwf), &cronWfObj)
	assert.NoError(t, err)
	cronWfList := wfv1.CronWorkflowList{
		Items: []wfv1.CronWorkflow{cronWfObj},
	}

	cronWfClient.On("ListCronWorkflows", mock.Anything, mock.Anything).Return(&cronWfList, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)

	listCommand := NewListCommand()

	output := test.ExecuteCommand(t, listCommand)
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "hello-world")
}
