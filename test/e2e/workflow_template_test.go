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

type WorkflowTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *WorkflowTemplateSuite) TestSubmitWorkflowTemplate() {
	s.Given().
		WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
		When().
		CreateWorkflowTemplates().
		RunCli([]string{"submit", "--from", "workflowtemplate/workflow-template-whalesay-template", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowSucceeded)
		})
}

func (s *WorkflowTemplateSuite) TestNestedWorkflowTemplate() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-nested-template.yaml").
		WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
		When().
		CreateWorkflowTemplates().
		Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-nested-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    steps:
      - - name: call-whalesay-template
          templateRef:
            name: workflow-template-nested-template
            template: whalesay-template
          arguments:
            parameters:
            - name: message
              value: "hello from nested"
`).When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowSucceeded)
		})
}

func (s *WorkflowTemplateSuite) TestSubmitWorkflowTemplateWithEnum() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-with-enum-values.yaml").
		When().
		CreateWorkflowTemplates().
		RunCli([]string{"submit", "--from", "workflowtemplate/workflow-template-with-enum-values", "-l", "workflows.argoproj.io/test=true"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowSucceeded)
		})
}

func TestWorkflowTemplateSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTemplateSuite))
}
