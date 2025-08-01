//go:build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ClusterWorkflowTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *ClusterWorkflowTemplateSuite) TestNestedClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@testdata/cluster-workflow-template-nested-template.yaml").
		When().Given().
		ClusterWorkflowTemplate("@smoke/cluster-workflow-template-whalesay-template.yaml").
		When().CreateClusterWorkflowTemplates().
		Given().
		Workflow("@functional/cwft-wf.yaml").When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestClusterWorkflowTemplateSuite(t *testing.T) {
	suite.Run(t, new(ClusterWorkflowTemplateSuite))
}
