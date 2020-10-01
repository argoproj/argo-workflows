// +build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
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
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.NotEmpty(t, status.EstimatedDuration)
			assert.NotEmpty(t, status.Nodes[metadata.Name].EstimatedDuration)
		})
}

func (s *EstimatedDurationSuite) TestClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@testdata/basic-clusterworkflowtemplate.yaml").
		When().
		CreateClusterWorkflowTemplates().
		SubmitWorkflowsFromClusterWorkflowTemplates().
		WaitForWorkflow().
		SubmitWorkflowsFromClusterWorkflowTemplates().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.NotEmpty(t, status.EstimatedDuration)
			assert.NotEmpty(t, status.Nodes[metadata.Name].EstimatedDuration)
		})
}

func (s *EstimatedDurationSuite) TestCronWorkflow() {
	s.Given().
		CronWorkflow("@testdata/basic-cronworkflow.yaml").
		When().
		CreateCronWorkflow().
		SubmitWorkflowsFromCronWorkflows().
		WaitForWorkflow().
		SubmitWorkflowsFromCronWorkflows().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.NotEmpty(t, status.EstimatedDuration)
			assert.NotEmpty(t, status.Nodes[metadata.Name].EstimatedDuration)
		})
}

func TestEstimatedDurationSuite(t *testing.T) {
	suite.Run(t, new(EstimatedDurationSuite))
}
