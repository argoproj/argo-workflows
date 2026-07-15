//go:build executor

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
)

type InvalidCommandSuite struct {
	fixtures.E2ESuite
}

func (s *InvalidCommandSuite) TestInvalidCommand() {
	s.Given().
		Workflow("@testdata/cannot-start-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Contains(t, status.Message, "executable file not found")
		})
}

func TestInvalidCommandSuite(t *testing.T) {
	suite.Run(t, new(InvalidCommandSuite))
}
