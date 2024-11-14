//go:build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type ClusterWorkflowTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *ClusterWorkflowTemplateSuite) TestClusterWorkflowFromWorkflowTemplateHasLabel() {
	s.Given().
		ClusterWorkflowTemplate("@smoke/cluster-workflow-template-whalesay-template.yaml").
		When().
		CreateClusterWorkflowTemplates().
		Given().
		Workflow(`
metadata:
  generateName: workflow-from-cluster-template-label-
spec:
  workflowTemplateRef:
    name: cluster-workflow-template-whalesay-template
    clusterScope: true
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, "cluster-workflow-template-whalesay-template", metadata.Labels[common.LabelKeyClusterWorkflowTemplate])
		})
}

func (s *ClusterWorkflowTemplateSuite) TestNestedClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@testdata/cluster-workflow-template-nested-template.yaml").
		When().Given().
		ClusterWorkflowTemplate("@smoke/cluster-workflow-template-whalesay-template.yaml").
		When().CreateClusterWorkflowTemplates().
		Given().
		Workflow(`
metadata:
  generateName: cwft-wf-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    steps:
    - - name: call-whalesay-template
        templateRef:
          name: cluster-workflow-template-nested-template 
          template: whalesay-template
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: hello from nested
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestClusterWorkflowTemplateSuite(t *testing.T) {
	suite.Run(t, new(ClusterWorkflowTemplateSuite))
}
