//go:build functional
// +build functional

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type RetryTestSuite struct {
	fixtures.E2ESuite
}

func (s *RetryTestSuite) TestRetryLimit() {
	s.Given().
		Workflow(`
metadata:
  name: test-retry-limit
spec:
  entrypoint: main
  templates:
    - name: main
      retryStrategy:
        limit: 0
        backoff:
          duration: 2s
          factor: 2
          maxDuration: 5m
      container:
        name: main
        image: 'argoproj/argosay:v2'
        args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowPhase("Failed"), status.Phase)
			assert.Equal(t, "No more retries left", status.Message)
			assert.Equal(t, v1alpha1.Progress("0/1"), status.Progress)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-retry-limit"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.Equal(t, v1alpha1.NodeTypeRetry, status.Type)
			assert.Nil(t, status.NodeFlag)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "test-retry-limit(0)"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeFailed, status.Phase)
			assert.Equal(t, true, status.NodeFlag.Retried)
		})
}

func (s *RetryTestSuite) TestRetryBackoff() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-backoff-strategy-
spec:
  entrypoint: main
  templates:
    - name: main
      retryStrategy:
        limit: '10'
        backoff:
          duration: 10s
          maxDuration: 1m
      container:
          name: main
          image: 'argoproj/argosay:v2'
          args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Second * 90).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowPhase("Failed"), status.Phase)
			assert.LessOrEqual(t, len(status.Nodes), 10)
		})
	s.Given().
		Workflow(`
metadata:
  generateName: test-backoff-strategy-
spec:
  entrypoint: main
  templates:
    - name: main
      retryStrategy:
        limit: 10
        backoff:
          duration: 10s
          maxDuration: 1m
      container:
          name: main
          image: 'argoproj/argosay:v2'
          args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Second * 90).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowPhase("Failed"), status.Phase)
			assert.LessOrEqual(t, len(status.Nodes), 10)
		})
}

func (s *RetryTestSuite) TestManualRetry() {
	failWorkflowWhen := s.Given().
		Workflow(`
metadata:
  name: fail-workflow
  labels:
    workflows.argoproj.io/workflow: "fail-workflow"
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        name: main
        image: 'argoproj/argosay:v2'
        command: [sh, -c]
        args:
          - |
            echo "test artifact" > /tmp/artifact.txt
            exit 1
      outputs:
        artifacts:
          - name: artifact
            path: /tmp/artifact.txt
`).When()

	retryVerifyNoErrorWorkflowWhen := s.Given().
		Workflow(`
metadata:
  name: retry-verify-no-error
spec:
  entrypoint: main
  serviceAccountName: argo
  templates:
  - name: main
    steps:
      - - name: workflow-retry
          template: workflow-retry
        - name: wait-for-workflow-running
          template: wait-for-workflow-running
      - - name: verify-no-retry-error
          template: verify-no-retry-error

  - name: workflow-retry
    container:
      image: argoproj/argocli:latest
      args: 
        - retry
        - -l
        - workflows.argoproj.io/workflow=fail-workflow
        - --namespace=argo
        - --loglevel=debug

  - name: wait-for-workflow-running
    inputs:
      artifacts:
        - name: argocli
          path: /tmp/argo.gz
          http:
            url: https://github.com/argoproj/argo-workflows/releases/download/v3.5.4/argo-linux-amd64.gz
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          gunzip -c /tmp/argo.gz > /bin/argo
          chmod +x /bin/argo
          x=0
          while [ $x -le 30 ]
          do
            argo get fail-workflow -o yaml > /tmp/workflow.txt
            grep "phase: Running" /tmp/workflow.txt > /tmp/result.txt 2>&1
            if [ -s "/tmp/result.txt" ]
            then
              cat /tmp/result.txt
              echo "Workflow running"
              break
            fi
            x=$(( $x + 1 ))
            sleep 1
          done

  - name: verify-no-retry-error
    inputs:
      artifacts:
        - name: argocli
          path: /tmp/argo.gz
          http:
            url: https://github.com/argoproj/argo-workflows/releases/download/v3.5.4/argo-linux-amd64.gz
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          gunzip -c /tmp/argo.gz > /bin/argo
          chmod +x /bin/argo
          x=0
          while [ $x -le 30 ]
          do
            argo get fail-workflow -o yaml > /tmp/workflow.txt
            grep "phase: Failed" /tmp/workflow.txt > /tmp/failed.txt 2>&1
            if [ -s /tmp/failed.txt ]
            then
              echo "Successfully retried failing workflow."
              exit 0
            fi
            x=$(( $x + 1 ))
            sleep 1
          done
          echo "Timed out waiting for failed workflow."
          grep "phase: " /tmp/workflow.txt > /tmp/phase.txt 2>&1
          cat /tmp/phase.txt
          exit 1
`).When()

	failWorkflowWhen.
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed)

	retryVerifyNoErrorWorkflowWhen.
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetryTestSuite))
}
