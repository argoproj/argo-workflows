package e2e

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	workflowYaml := `
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
`

	content := []byte(workflowYaml)
	tmpfile, err := ioutil.TempFile("", "argo_test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	commands.SubmitWorkflows([]string{tmpfile.Name()}, nil, nil)

	wfClient := commands.InitWorkflowClient()

	for {
		wf, err := wfClient.Get(workflowName, metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}
		var allCompleted bool
		allCompleted = true
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
	wfClient.Delete(workflowName, &deleteOptions)
}

func TestArgoWorkflows(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
