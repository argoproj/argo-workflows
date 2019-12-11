package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type WorkflowSuite struct {
	fixtures.E2ESuite
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
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: continue-on-fail
spec:
  entrypoint: workflow-ignore
  parallelism: 1
  templates:
  - name: workflow-ignore
    steps:
    - - name: A
        template: whalesay
      - name: B
        template: boom
        continueOn:
          failed: true
    - - name: D
        depedencies: [A, B]
        template: whalesay

  - name: boom
    dag:
      tasks:
      - name: B-1
        template: whalesplosion

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]

  - name: whalesplosion
    container:
      image: docker/whalesay:latest
      command: [boom]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		Expect(func(t *testing.T, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.Len(t, status.Nodes, 7)
			nodeStatus := status.Nodes.FindByDisplayName("B")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
				assert.Len(t, nodeStatus.Children, 1)
				assert.Len(t, nodeStatus.OutboundNodes, 1)
			}
		})
}

func TestArgoWorkflows(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
