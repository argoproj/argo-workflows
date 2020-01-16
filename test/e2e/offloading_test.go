package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo/test/e2e/fixtures"
)

type OffloadingSuite struct {
	fixtures.E2ESuite
}

func (s *OffloadingSuite) TestOffloading() {
	assert.Equal(s.T(), 0, s.Persistence.OffloadedCount())

	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(10 * time.Second)

	assert.NotEqual(s.T(), 0, s.Persistence.OffloadedCount())

	s.Given().
		WorkflowName("basic").
		When().
		DeleteWorkflow()

	time.Sleep(3*time.Second)

	assert.Equal(s.T(), 0, s.Persistence.OffloadedCount())
}

func TestOffloadingSuite(t *testing.T) {
	suite.Run(t, new(OffloadingSuite))
}
