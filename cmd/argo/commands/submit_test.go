package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"os"
	"testing"
)

const wf1 string =`
{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "Workflow",
  "metadata": {
    "name": "hello-world-"
  },
  "spec": {
    "entrypoint": "whalesay",
    "templates": [
      {
        "name": "whalesay",
        "container": {
          "image": "docker/whalesay:latest",
          "command": [
            "cowsay"
          ],
          "args": [
            "hello world"
          ]
        }
      }
    ]
  }
}
`

func TestSubmitFromResource(t *testing.T) {
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	wfClient.On("SubmitFrom", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	output := CaptureOutput(func(){submitWorkflowFromResource("workflowtemplate/test",&wfv1.SubmitOpts{},&cliSubmitOpts{})})
	fmt.Println(output)
	assert.Contains(t, output, "Created:")
}

func TestSubmitWorkflows(t *testing.T) {
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	wfClient.On("CreateWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wfv1.Workflow{}, nil)
	wfClient.On("CreateWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments){
		mock.Call.Return( args.Get(1).(wfv1.Workflow), nil)
	})
	//wfClient.On("CreateWorkflow", mock.MatchedBy(func(req workflow.WorkflowCreateRequest) (*wfv1.Workflow, error){
	//
	//}))
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	var wf wfv1.Workflow
	err:=json.Unmarshal([]byte(wf1), &wf)
	fmt.Println(err)
	workflows := []wfv1.Workflow{wf}
	output := CaptureOutput(func(){submitWorkflows(workflows,&wfv1.SubmitOpts{},&cliSubmitOpts{})})
	fmt.Println(output)
	assert.Contains(t, output, "Created:")
}


func CaptureOutput(f func()) string {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	return string(out)
}