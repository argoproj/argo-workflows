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
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(10 * time.Second).
		Then().
		Expect(func(t *testing.T, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
		})
}

func (s *SmokeSuite) TestArtifactPassing() {
	s.Given().
		Workflow("@smoke/artifact-passing.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(20 * time.Second).
		Then().
		Expect(func(t *testing.T, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
		})
}

func TestSmokeSuite(t *testing.T) {
	suite.Run(t, new(SmokeSuite))
}
