package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type SmokeSuite struct {
	fixtures.E2ESuite
}

func (s *SmokeSuite) TestBasic() {

	// TODO
	s.T().SkipNow()
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    name: basic
spec:
    entrypoint: run-workflow
    templates:
    - name: run-workflow
      container:
        image: docker/whalesay:latest
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(10*time.Second).
		Then().
		Expect(func(t *testing.T, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
		})
}

func (s *SmokeSuite) TestArtifactPassing() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: artifact-passing
spec:
  entrypoint: artifact-example
  templates:
  - name: artifact-example
    steps:
    - - name: generate-artifact
        template: generate-message
    - - name: consume-artifact
        template: print-message
        arguments:
          artifacts:
          - name: message
            from: "{{steps.generate-artifact.outputs.artifacts.hello-art}}"

  - name: generate-message
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      - name: hello-art
        path: /tmp/hello_world.txt

  - name: print-message
    inputs:
      artifacts:
      - name: message
        path: /tmp/message
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cat /tmp/message"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
		Then().
		Expect(func(t *testing.T, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
		})
}

func (s *SmokeSuite) TestContinueOnFail() {
	// TODO
	s.T().SkipNow()
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: continue-on-fail
spec:
  entrypoint: workflow-ignore
  parallelism: 2
  templates:
  - name: workflow-ignore
    steps:
    - - name: A
        template: whalesay
      - name: B
        template: boom
        continueOn:
          failed: true
    - - name: C
        dependencies: [A, B]
        template: whalesay

  - name: boom
    dag:
      tasks:
      - name: B-1
        template: whalesplosion

  - name: whalesay
    container:
      image: docker/whalesay:latest

  - name: whalesplosion
    container:
      image: docker/whalesay:latest
      command: ["sh", "-c", "sleep 5 ; exit 1"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(30*time.Second).
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

func TestSmokeSuite(t *testing.T) {
	suite.Run(t, new(SmokeSuite))
}
