package cron

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewLintCommand(t *testing.T) {
	err := ioutil.WriteFile("cronwf.yaml", []byte(cronwf), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	common.CreateNewAPIClientFunc = func() (context.Context, apiclient.Client) {
		return context.TODO(), &client
	}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	var cronWfObj wfv1.CronWorkflow
	err = yaml.Unmarshal([]byte(cronwf), &cronWfObj)
	assert.NoError(t, err)

	cronWfClient.On("LintCronWorkflow", mock.Anything, mock.Anything).Return(&cronWfObj, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"cronwf.yaml"})
	output := test.ExecuteCommand(t, lintCommand)
	os.Remove("cronwf.yaml")
	assert.Contains(t, output, "manifests validated")
}
