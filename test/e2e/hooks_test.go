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

func (s *HooksSuite) TestWorkflowLevelHooksSuccessVersion() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: lifecycle-hook-
spec:
  entrypoint: main
  hooks:
    running:
      expression: workflow.status == "Running"
      template: argosay
    succeed:
      expression: workflow.status == "Succeeded"
      template: argosay

  templates:
    - name: main
      steps:
      - - name: step1
          template: argosay

    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowSucceeded)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestWorkflowLevelHooksFailVersion() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: lifecycle-hook-
spec:
  entrypoint: main
  hooks:
    running:
      expression: workflow.status == "Running"
      template: hook
    failed:
      expression: workflow.status == "Failed"
      template: hook

  templates:
    - name: main
      steps:
      - - name: step1
          template: argosay

    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/argosay", "sleep 5", "exit 1"]
        
    - name: hook
      container:
        image: argoproj/argosay:v2
        command: ["/argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowFailed)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, ".hooks.failed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksStepSuccessVersion() {
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
              running:
                expression: steps["step-1"].status == "Running"
                template: argosay
              succeed:
                expression: steps["step-1"].status == "Succeeded"
                template: argosay
            template: argosay
        - - name: step-2
            hooks:
              running:
                expression: steps["step-2"].status == "Running"
                template: argosay
              succeed:
                expression: steps["step-2"].status == "Succeeded"
                template: argosay
            template: argosay
    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-2.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-2.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksStepFailVersion() {
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
              running:
                expression: steps["step-1"].status == "Running"
                template: hook
              failed:
                expression: steps["step-1"].status == "Failed"
                template: hook
            template: argosay
    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/argosay", "sleep 5", "exit 1"]
    - name: hook
      container:
        image: argoproj/argosay:v2
        command: ["/argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.failed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksDagSuccessVersion() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: lifecycle-hook-tmpl-level-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: step-1
            hooks:
              running:
                expression: tasks["step-1"].status == "Running"
                template: argosay
              succeed:
                expression: tasks["step-1"].status == "Succeeded"
                template: argosay
            template: argosay
          - name: step-2
            hooks:
              running:
                expression: tasks["step-2"].status == "Running"
                template: argosay
              succeed:
                expression: tasks["step-2"].status == "Succeeded"
                template: argosay
            template: argosay
            dependencies: [step-1]
    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-2.hooks.succeed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-2.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func (s *HooksSuite) TestTemplateLevelHooksDagFailVersion() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: lifecycle-hook-tmpl-level-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: step-1
            hooks:
              running:
                expression: tasks["step-1"].status == "Running"
                template: hook
              failed:
                expression: tasks["step-1"].status == "Failed"
                template: hook
            template: argosay
    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/argosay", "sleep 5", "exit 1"]
    - name: hook
      container:
        image: argoproj/argosay:v2
        command: ["/argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.failed")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	}).ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
		return strings.Contains(status.Name, "step-1.hooks.running")
	}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
		assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	})
}

func TestHooksSuite(t *testing.T) {
	suite.Run(t, new(HooksSuite))
}