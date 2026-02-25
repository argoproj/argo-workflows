//go:build functional

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
)

type WorkflowTemplateSuite struct {
	fixtures.E2ESuite
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
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowTemplateSuite) TestSubmitWorkflowTemplateWithEnum() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-with-enum-values.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowTemplateSuite) TestSubmitWorkflowTemplateWorkflowMetadataSubstitution() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-sub-test.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow().
		Then().
		ExpectWorkflowNode(v1alpha1.SucceededPodNode, func(t *testing.T, n *v1alpha1.NodeStatus, p *apiv1.Pod) {
			for _, c := range p.Spec.Containers {
				if c.Name == "main" {
					assert.Len(t, c.Args, 3)
					assert.Equal(t, "echo", c.Args[0])
					assert.Equal(t, "myLabelArg", c.Args[1])
					assert.Equal(t, "thisLabelIsFromWorkflowDefaults", c.Args[2])
				}
			}
		})
}

func (s *WorkflowTemplateSuite) TestSubmitWorkflowTemplateResourceUnquotedExpressions() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-with-resource-expr.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowTemplateSuite) TestSubmitWorkflowTemplateWithParallelStepsRequiringPVC() {
	s.Given().
		WorkflowTemplate("@testdata/loops-steps-limited-parallelism-pvc.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).
		ExpectPVCDeleted()
}

func (s *WorkflowTemplateSuite) TestWorkflowTemplateInvalidOnExit() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-invalid-onexit.yaml").
		Workflow(`
metadata:
  generateName: workflow-template-invalid-onexit-
spec:
  workflowTemplateRef:
    name: workflow-template-invalid-onexit
`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowError, status.Phase)
			assert.Contains(t, status.Message, "error in exit template execution")
		}).
		ExpectPVCDeleted()
}

func (s *WorkflowTemplateSuite) TestWorkflowTemplateWithHook() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-templates/success-hook.yaml").
		Workflow(`
metadata:
  generateName: workflow-template-hook-
spec:
  workflowTemplateRef:
    name: hook
`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "hooks.succeed")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *WorkflowTemplateSuite) TestWorkflowTemplateInvalidEntryPoint() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-invalid-entrypoint.yaml").
		Workflow(`
metadata:
  generateName: workflow-template-invalid-entrypoint-
spec:
  workflowTemplateRef:
    name: workflow-template-invalid-entrypoint
`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowError, status.Phase)
			assert.Contains(t, status.Message, "error in entry template execution")
		}).
		ExpectPVCDeleted()
}

func TestWorkflowTemplateSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTemplateSuite))
}
