package cron

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/cmd/argo/commands/test"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow/mocks"
	clientmocks "github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNewLintCommand(t *testing.T) {
	err := ioutil.WriteFile("cronwf.yaml", []byte(cronwf), 0644)
	assert.NoError(t, err)
	client := clientmocks.Client{}
	cronWfClient := mocks.CronWorkflowServiceClient{}
	var cronWfObj wfv1.CronWorkflow
	err = yaml.Unmarshal([]byte(cronwf), &cronWfObj)
	assert.NoError(t, err)

	cronWfClient.On("LintCronWorkflow", mock.Anything, mock.Anything).Return(&cronWfObj, nil)
	client.On("NewCronWorkflowServiceClient").Return(&cronWfClient)
	common.APIClient = &client
	lintCommand := NewLintCommand()
	lintCommand.SetArgs([]string{"cronwf.yaml"})
	execFunc := func() {
		err := lintCommand.Execute()
		assert.NoError(t, err)
	}
	output := test.CaptureOutput(execFunc)
	os.Remove("cronwf.yaml")
	assert.Contains(t, output, "manifests validated")
}
