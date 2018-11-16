package e2e

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WaitSuite struct {
	suite.Suite
	testNamespace string
}

func (suite *WaitSuite) SetupSuite() {
	if *kubeConfig == "" {
		suite.T().Skip("Skipping test. Kubeconfig not provided")
	}
	suite.testNamespace = createNamespaceForTest()
}

func (suite *WaitSuite) TearDownSuite() {
	if err := deleteTestNamespace(suite.testNamespace); err != nil {
		panic(err)
	}
	fmt.Printf("Deleted namespace %s\n", suite.testNamespace)
}

func (suite *WaitSuite) TestWait() {
	t := suite.T()

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
        command: [sleep, 5]
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
	commands.WaitWorkflows([]string{workflowName}, false, false)

	wf, err := wfClient.Get(workflowName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, wf.Status.FinishedAt.IsZero())

	deleteOptions := metav1.DeleteOptions{}
	wfClient.Delete(workflowName, &deleteOptions)
}

func TestWaitCmd(t *testing.T) {
	suite.Run(t, new(WaitSuite))
}
