//go:build functional
// +build functional

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type HooksSuite struct {
	fixtures.E2ESuite
}

func (s *HooksSuite) TestWorkflowLevelHooks() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: lifecycle-hook-
spec:
  entrypoint: main
  hooks:
    exit:
      template: http
    running:
      expression: workflow.status == "Running"
      template: http
  templates:
    - name: main
      steps:
      - - name: step1
          template: http

    - name: http
      http:
        url: "http://httpstat.us"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowSucceeded)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "hook")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {

		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooks() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: lifecycle-hook-tmpl-level-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: step-1
            hooks:
              exit:
                expression: steps["step-1"].status == "Running"
                template: http
              success:
                expression: steps["step-1"].status == "Succeeded"
                template: http
            template: http
    - name: http
      http:
        url: "http://httpstat.us"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowSucceeded)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "hook")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {

		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func TestHooksSuite(t *testing.T) {
	suite.Run(t, new(HooksSuite))
}
