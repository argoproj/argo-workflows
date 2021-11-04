//go:build functional
// +build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type EstimatedDurationSuite struct {
	fixtures.E2ESuite
}

func (s *EstimatedDurationSuite) TestWorkflowTemplate() {
	s.Given().
		WorkflowTemplate("@testdata/basic-workflowtemplate.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.NotEmpty(t, status.EstimatedDuration)
			assert.NotEmpty(t, status.Nodes[metadata.Name].EstimatedDuration)
		})
}

func TestEstimatedDurationSuite(t *testing.T) {
	suite.Run(t, new(EstimatedDurationSuite))
}
