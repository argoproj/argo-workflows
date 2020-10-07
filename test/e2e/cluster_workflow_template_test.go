// +build e2e

package e2e

import (
	"testing"

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
		RunCli([]string{"submit", "--from", "clusterworkflowtemplate/cluster-workflow-template-whalesay-template", "--name", "my-wf", "-l", "argo-e2e=true"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.NodeSucceeded)
		})
}

func (s *ClusterWorkflowTemplateSuite) TestNestedClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate("@testdata/cluster-workflow-template-nested-template.yaml").
		When().Given().
		ClusterWorkflowTemplate("@smoke/cluster-workflow-template-whalesay-template.yaml").
		When().CreateClusterWorkflowTemplates().
		Given().
		WorkflowName("cwft-wf").
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: cwft-wf
  namespace: argo
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
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: hello from nested
`).When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})

}

func TestClusterWorkflowTemplateSuite(t *testing.T) {
	suite.Run(t, new(ClusterWorkflowTemplateSuite))
}
