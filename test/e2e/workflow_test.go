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

func (s *WorkflowSuite) TestRunWorkflowBasic() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    generateName: my-test-
spec:
    entrypoint: run-workflow
    templates:
    - name: run-workflow
      container:
        image: argosay:v1
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		Expect(func(t *testing.T, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
		})
}

func (s *WorkflowSuite) TestContinueOnFail() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: continue-on-fail-
spec:
  entrypoint: workflow-ignore
  parallelism: 1
  templates:
  - name: workflow-ignore
    steps:
    - - name: A
        template: argosay
      - name: B
        template: boom
        continueOn:
          failed: true
    - - name: C
        dependencies: [A, B]
        template: argosay

  - name: boom
    dag:
      tasks:
      - name: B-1
        template: whalesplosion

  - name: argosay
    container:
      image: argosay:v1

  - name: whalesplosion
    container:
      image: argosay:v1
      command: ["argosay", "--sleep", "5s", "--exit-code", "1"]
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

func TestWorkflowSuite(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
