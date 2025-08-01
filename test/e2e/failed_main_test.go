//go:build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type FailedMainSuite struct {
	fixtures.E2ESuite
}

func (s *FailedMainSuite) TestFailedMain() {
	s.Given().
		Workflow("@functional/failed-main.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed)
}

func TestFailedMainSuite(t *testing.T) {
	suite.Run(t, new(FailedMainSuite))
}
