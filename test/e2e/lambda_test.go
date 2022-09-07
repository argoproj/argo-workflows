//go:build executor
// +build executor

package e2e

import (
	"github.com/stretchr/testify/suite"
	"testing"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type LambdaSuite struct {
	fixtures.E2ESuite
}

func (s *LambdaSuite) TestImage() {
	s.Given().
		Workflow("@testdata/lambda-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestLambdaSuite(t *testing.T) {
	suite.Run(t, new(LambdaSuite))
}
