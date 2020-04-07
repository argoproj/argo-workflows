package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type ClusterWorkflowTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *ClusterWorkflowTemplateSuite) TestSubmitClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@smoke/cluster-workflow-template-whalesay-template.yaml").
		WorkflowName("my-wf").
		When().
		CreateClusterWorkflowTemplates().
		RunCli([]string{"submit", "--from", "clusterworkflowtemplate/cluster-workflow-template-whalesay-template", "--name", "my-wf"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		}).
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.NodeSucceeded)
		})
}

func (s *ClusterWorkflowTemplateSuite) TestNestedClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@testdata/cluster-workflow-template-nested-template.yaml").
		When().CreateClusterWorkflowTemplates()

	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-nested-
  labels:
    argo-e2e: true
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    steps:
      - - name: call-whalesay-template
          templateRef:
		    name: cluster-workflow-template-nested-template
		    template: whalesay-template
		    clusterscope: true
          arguments:
            parameters:
            - name: message
              value: hello from nested
`).When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.NodeSucceeded)
		})

}

func TestClusterWorkflowTemplateSuite(t *testing.T) {
	suite.Run(t, new(ClusterWorkflowTemplateSuite))
}
