// +build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ClusterWorkflowTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *ClusterWorkflowTemplateSuite) TestSubmitClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@smoke/cluster-workflow-template-whalesay-template.yaml").
		When().
		CreateClusterWorkflowTemplates().
		RunCli([]string{"submit", "--from", "clusterworkflowtemplate/cluster-workflow-template-whalesay-template", "--name", "my-wf", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		}).
		WaitForWorkflow(fixtures.ToBeSucceeded)
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
