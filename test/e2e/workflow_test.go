package e2e

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands"
)

type WorkflowSuite struct {
	E2ESuite
}

func (suite *WorkflowSuite) TestRunWorkflowBasic() {
	workflowName := "my-test"
	tmpfile, closer := createTempFile(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    name: ` + workflowName + `
spec:
    entrypoint: run-workflow
    templates:
    - name: run-workflow
      container:
        image: alpine:3.6
        command: [date]
`)
	defer closer()

	commands.SubmitWorkflows([]string{tmpfile}, nil, nil)

	wfClient := commands.InitWorkflowClient()

	for {
		wf, err := wfClient.Get(workflowName, metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}
		allCompleted := true
		for k, v := range wf.Status.Nodes {
			if !v.Completed() {
				fmt.Printf("Status of %s: %v\n", k, v.Phase)
				allCompleted = false
			}
		}

		if allCompleted {
			fmt.Printf("Workflow %s completed successfully", workflowName)
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	err := wfClient.Delete(workflowName, &metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
}

func (suite *WorkflowSuite) TestContinueOnFail() {
	commands.SubmitWorkflows([]string{"functional/continue-fail.yaml"}, nil,nil)
}

func TestArgoWorkflows(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
