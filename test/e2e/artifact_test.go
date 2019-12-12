package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type ArtifactSuite struct {
	fixtures.E2ESuite
}

func (suite ArtifactSuite) Test() {
	suite.Given().
		Workflow("@functional/artifact-input-output-samedir.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		Expect(func(t *testing.T, wf *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.NodeSucceeded, wf.Phase)
		})
}

func TestArtifactSuite(t *testing.T) {
	suite.Run(t, new(ArtifactSuite))
}
