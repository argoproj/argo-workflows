// +build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ScaleSubresourceSuite struct {
	fixtures.E2ESuite
}

func (s *ScaleSubresourceSuite) TestWorkflowCompletesIfContainsDaemonPod() {
	s.Given().
		Workflow("@smoke/basic-generate-name.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			name := metadata.GetName()
			assert.Equal(t, name, status.Selector)
		})
}

func TestScaleSubresourceSuite(t *testing.T) {
	suite.Run(t, new(ScaleSubresourceSuite))
}
