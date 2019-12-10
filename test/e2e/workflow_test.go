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
	suite.Suite
	testNamespace string
}

func (suite *WorkflowSuite) SetupSuite() {
	if *kubeConfig == "" {
		suite.T().Skip("Skipping test. Kubeconfig not provided")
		return
	}
	suite.testNamespace = createNamespaceForTest()
}

func (suite *WorkflowSuite) TearDownSuite() {
	if err := deleteTestNamespace(suite.testNamespace); err != nil {
		panic(err)
	}
	fmt.Printf("Deleted namespace %s\n", suite.testNamespace)
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

	deleteOptions := metav1.DeleteOptions{}
	err := wfClient.Delete(workflowName, &deleteOptions)
	if err != nil {
		log.Fatal(err)
	}
}

func TestArgoWorkflows(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
