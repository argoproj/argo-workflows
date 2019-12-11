package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type WorkflowSuite struct {
	E2ESuite
}

func (suite *WorkflowSuite) TestRunWorkflowBasic() {
	suite.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    name: my-test
spec:
    entrypoint: run-workflow
    templates:
    - name: run-workflow
      container:
        image: alpine:3.6
        command: ["sleep", "1"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		DeleteWorkflow()
}

func (suite *WorkflowSuite) TestContinueOnFail() {
	suite.Given().
		Workflow("@functional/continue-on-fail.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		DeleteWorkflow()
}

func TestArgoWorkflows(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
