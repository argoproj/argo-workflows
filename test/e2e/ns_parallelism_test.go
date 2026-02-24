//go:build functional

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
)

type NamespaceParallelismSuite struct {
	fixtures.E2ESuite
}

const wf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  labels:
    workflows.argoproj.io/archive-strategy: "false"
  annotations:
    workflows.argoproj.io/description: |
      This is a simple hello world example.
spec:
  entrypoint: hello-world
  templates:
  - name: hello-world
    container:
      image: "argoproj/argosay:v2"
      command: [sleep]
      args: ["60"]
`

func (s *NamespaceParallelismSuite) TestNamespaceParallelism() {

	s.Given().
		Workflow(wf).
		When().
		AddNamespaceLimit("1").
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart)

	time.Sleep(time.Second * 5)
	wf := s.Given().
		Workflow(wf).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBePending).GetWorkflow()
	t := s.T()
	assert.Equal(t, "Workflow processing has been postponed because too many workflows are already running", wf.Status.Message)
}

func TestNamespaceParallelismSuite(t *testing.T) {
	suite.Run(t, new(NamespaceParallelismSuite))
}
