//go:build functional

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
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
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
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay; exit 1"]
        
    - name: hook
      container:
        image: argoproj/argosay:v2
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
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
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay"]
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
	})
	// TODO: Temporarily comment out this assertion since it's flaky:
	// 	  The running hook is occasionally not triggered. Possibly because the step finishes too quickly
	//	  while the controller did not get a chance to trigger this hook.
	//.ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
	//	return strings.Contains(status.Name, "step-2.hooks.running")
	//}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
	//	assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
	//})
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
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay; exit 1"]
    - name: hook
      container:
        image: argoproj/argosay:v2
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay"]
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
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay"]
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
		// TODO: Temporarily comment out this assertion since it's flaky:
		// 	  The running hook is occasionally not triggered. Possibly because the step finishes too quickly
		//	  while the controller did not get a chance to trigger this hook.
		//assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
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
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay; exit 1"]
    - name: hook
      container:
        image: argoproj/argosay:v2
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay"]
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

func (s *HooksSuite) TestTemplateLevelHooksDagHasDependencyVersion() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: lifecycle-hook-tmpl-level-
spec:
  templates:
    - name: main
      dag:
        tasks:
          - name: A
            template: fail
            hooks:
              running:
                template: hook
                expression: tasks.A.status == "Running"
              success:
                template: hook
                expression: tasks.A.status == "Succeeded"
          - name: B
            template: success
            dependencies:
              - A
            hooks:
              running:
                template: hook
                expression: tasks.B.status == "Running"
              success:
                template: hook
                expression: tasks.B.status == "Succeeded"
    - name: success
      container:
        name: ''
        image: argoproj/argosay:v2
        command:
          - /bin/sh
          - '-c'
        args:
          - /bin/sleep 1; /argosay; exit 0
    - name: fail
      container:
        name: ''
        image: argoproj/argosay:v2
        command:
          - /bin/sh
          - '-c'
        args:
          - /bin/sleep 1; /argosay; exit 1
    - name: hook
      container:
        name: ''
        image: argoproj/argosay:v2
        command:
          - /bin/sh
          - '-c'
        args:
          - /bin/sleep 1; /argosay
  entrypoint: main
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
			// Make sure unnecessary hooks are not triggered
			assert.Equal(t, status.Progress, v1alpha1.Progress("1/2"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "A.hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "B")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeOmitted, status.Phase)
		})
}

func (s *HooksSuite) TestWorkflowLevelHooksWaitForTriggeredHook() {
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
      template: argosay-sleep-2seconds
    # This hook never triggered by following test.
    # To guarantee workflow does not wait forever for untriggered hooks.
    failed:
      expression: workflow.status == "Failed"
      template: argosay-sleep-2seconds
  templates:
    - name: main
      steps:
      - - name: step1
          template: argosay

    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 1; /argosay"]
    - name: argosay-sleep-2seconds
      container:
        image: argoproj/argosay:v2
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 2; /argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("2/2"))
			assert.Equal(t, 1, int(status.Progress.N()/status.Progress.M()))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, ".hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *HooksSuite) TestTemplateLevelHooksWaitForTriggeredHook() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: example-steps
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: job
            template: argosay
            hooks:
              running:
                expression: steps['job'].status == "Running"
                template: argosay-sleep-2seconds
              failed:
                expression: steps['job'].status == "Failed"
                template: argosay-sleep-2seconds

    - name: argosay
      container:
        image: argoproj/argosay:v2
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 5; /argosay"]
    - name: argosay-sleep-2seconds
      container:
        image: argoproj/argosay:v2
        command: ["/bin/sh", "-c"]
        args: ["/bin/sleep 2; /argosay"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("2/2"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "job.hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

// Ref: https://github.com/argoproj/argo-workflows/issues/11117
func (s *HooksSuite) TestTemplateLevelHooksWaitForTriggeredHookAndRespectSynchronization() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: example-steps-simple-mutex
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: job
            template: exit0
            hooks:
              running:
                expression: steps['job'].status == "Running"
                template: sleep
              succeed:
                expression: steps['job'].status == "Succeeded"
                template: sleep
    - name: sleep
      synchronization:
        mutexes:
          - name: job
      script:
        image: alpine:latest
        command: [/bin/sh]
        source: |
          sleep 4
    - name: exit0
      script:
        image: alpine:latest
        command: [/bin/sh]
        source: |
          sleep 2
          exit 0
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("3/3"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "job.hooks.running")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "job.hooks.succeed")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *HooksSuite) TestWorkflowLevelHooksWithRetry() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-workflow-level-hooks-with-retry
spec:
  templates:
    - name: argosay
      container:
        image: argoproj/argosay:v2
        command:
          - /bin/sh
          - '-c'
        args:
          - /bin/sleep 1; exit 1
      retryStrategy:
        limit: 1
    - name: hook
      container:
        image: argoproj/argosay:v2
        command:
          - /bin/sh
          - '-c'
        args:
          - /argosay
  entrypoint: argosay
  hooks:
    failed:
      template: hook
      expression: workflow.status == "Failed"
    running:
      template: hook
      expression: workflow.status == "Running"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
			assert.Equal(t, status.Progress, v1alpha1.Progress("2/4"))
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry.hooks.running"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
			assert.True(t, status.NodeFlag.Hooked)
			assert.False(t, status.NodeFlag.Retried)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry.hooks.failed"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
			assert.True(t, status.NodeFlag.Hooked)
			assert.False(t, status.NodeFlag.Retried)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.Equal(t, v1alpha1.NodeTypeRetry, status.Type)
			assert.Nil(t, status.NodeFlag)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry(0)"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.False(t, status.NodeFlag.Hooked)
			assert.True(t, status.NodeFlag.Retried)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-workflow-level-hooks-with-retry(1)"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.False(t, status.NodeFlag.Hooked)
			assert.True(t, status.NodeFlag.Retried)
		})
}

func (s *HooksSuite) TestTemplateLevelHooksWithRetry() {
	var children []string
	(s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retries-with-hooks-and-artifact
  labels:
    workflows.argoproj.io/test: "true"
  annotations:
    workflows.argoproj.io/description: |
      when retries and hooks are both included, the workflow cannot resolve the artifact 
    workflows.argoproj.io/version: '>= 3.5.0'
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: build
            template: output-artifact
            hooks:
              started:
                expression: steps["build"].status == "Running"
                template: started
              success:
                expression: steps["build"].status == "Succeeded"
                template: success
              failed:
                expression: steps["build"].status == "Failed" || steps["build"].status == "Error"
                template: failed
        - - name: print
            template: print-artifact
            arguments:
              artifacts:
                - name: message
                  from: "{{steps.build.outputs.artifacts.result}}"
    
    - name: output-artifact
      script:
        image: python:alpine3.6
        command: [ python ]
        source: |
          import time
          import random
          import sys
          time.sleep(1) # lifecycle hook for running won't trigger unless it runs for more than "a few seconds"
          with open("result.txt", "w") as f:
            f.write("Welcome")
          if {{retries}} == 2:
          	sys.exit(0)
          sys.exit(1)
      retryStrategy: 
        limit: 2
      outputs:
        artifacts:
          - name: result
            path: /result.txt

    - name: started
      container:
        image: python:alpine3.6
        command: [sh, -c]
        args: ["echo STARTED!"]

    - name: success
      container:
        image: python:alpine3.6
        command: [sh, -c]
        args: ["echo SUCCEEDED!"]

    - name: failed
      container:
        image: python:alpine3.6
        command: [sh, -c]
        args: ["echo FAILED or ERROR!"]

    - name: print-artifact
      inputs:
        artifacts:
          - name: message
            path: /tmp/message
      container:
        image: python:alpine3.6
        command: [sh, -c]
        args: ["cat /tmp/message"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.True(t, status.Fulfilled())
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
			for _, node := range status.Nodes {
				if node.Type == v1alpha1.NodeTypeRetry {
					assert.Equal(t, v1alpha1.NodeSucceeded, node.Phase)
					children = node.Children
				}
			}
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[0].build(0)"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Contains(t, children, status.ID)
			assert.False(t, status.NodeFlag.Hooked)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[0].build.hooks.started"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Contains(t, children, status.ID)
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[0].build.hooks.success"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Contains(t, children, status.ID)
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "retries-with-hooks-and-artifact[1].print"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *HooksSuite) TestExitHandlerWithWorkflowLevelDeadline() {
	var onExitNodeName string
	(s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handler-with-workflow-level-deadline
spec:
  entrypoint: main
  activeDeadlineSeconds: 1
  hooks:
    exit:
      template: exit-handler
  templates:
    - name: main
      steps:
      - - name: sleep
          template: sleep
    - name: exit-handler
      steps:
      - - name: sleep
          template: sleep
    - name: sleep
      container:
        image: argoproj/argosay:v2
        args: ["sleep", "5"]
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		WaitForWorkflow(fixtures.Condition(func(wf *v1alpha1.Workflow) (bool, string) {
			onExitNodeName = common.GenerateOnExitNodeName(wf.ObjectMeta.Name)
			onExitNode := wf.Status.Nodes.FindByDisplayName(onExitNodeName)
			return onExitNode.Completed(), "exit handler completed"
		})).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.DisplayName == onExitNodeName
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}))
}

func (s *HooksSuite) TestHttpExitHandlerWithWorkflowLevelDeadline() {
	var onExitNodeName string
	(s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-exit-handler-with-workflow-level-deadline
spec:
  entrypoint: main
  activeDeadlineSeconds: 1
  hooks:
    exit:
      template: exit-handler
  templates:
    - name: main
      steps:
      - - name: sleep
          template: sleep
    - name: sleep
      container:
        image: argoproj/argosay:v2
        args: ["sleep", "5"]
    - name: exit-handler
      steps:
      - - name: http
          template: http
    - name: http
      http:
        url: http://httpbin:9100/get
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		WaitForWorkflow(fixtures.Condition(func(wf *v1alpha1.Workflow) (bool, string) {
			onExitNodeName = common.GenerateOnExitNodeName(wf.ObjectMeta.Name)
			onExitNode := wf.Status.Nodes.FindByDisplayName(onExitNodeName)
			return onExitNode.Completed(), "exit handler completed"
		})).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.DisplayName == onExitNodeName
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.True(t, status.NodeFlag.Hooked)
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}))
}

func (s *HooksSuite) TestHooksWithArtifactsInSteps() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: test-hook-
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: hello1   
        template: parameterization
        hooks:
          success:
            template: success-hook
            expression: steps["hello1"].status == "Succeeded"
            arguments:
              artifacts:
              - name: file_path
                from: "{{steps.hello1.outputs.artifacts.result}}"

  - name: parameterization
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import os
        with open("foo.txt", "w") as f:
            f.write("Hello")
        os.rename('foo.txt', '/tmp/foo.txt')
    outputs:
      artifacts:
      - name: result
        path: /tmp/foo.txt

  - name: success-hook
    inputs:
      artifacts:
      - name: file_path
        path: /tmp/file_path
    script:
      image: python:alpine3.6
      command: ["sh"]
      source: |
        echo "File Path: {{inputs.artifacts.file_path.path}}"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, ".hooks.succeed")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func TestHooksSuite(t *testing.T) {
	suite.Run(t, new(HooksSuite))
}
