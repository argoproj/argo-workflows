package e2e

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands"
)

type WaitSuite struct {
	E2ESuite
	testNamespace string
}

func (suite *WaitSuite) TestWait() {
	t := suite.T()

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
        command: [sleep, 5]
`)
	defer closer()

	commands.SubmitWorkflows([]string{tmpfile}, nil, nil)

	wfClient := commands.InitWorkflowClient()
	commands.WaitWorkflows([]string{workflowName}, false, false)

	wf, err := wfClient.Get(workflowName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, wf.Status.FinishedAt.IsZero())

	deleteOptions := metav1.DeleteOptions{}
	err = wfClient.Delete(workflowName, &deleteOptions)
	if err != nil {
		log.Fatal(err)
	}
}

func TestWaitCmd(t *testing.T) {
	suite.Run(t, new(WaitSuite))
}
