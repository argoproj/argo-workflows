//go:build functional

package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
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
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowPhase("Failed"), status.Phase)
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
			assert.True(t, status.NodeFlag.Retried)
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
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowPhase("Failed"), status.Phase)
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
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowPhase("Failed"), status.Phase)
			assert.LessOrEqual(t, len(status.Nodes), 10)
		})
}

func (s *RetryTestSuite) TestWorkflowTemplateWithRetryStrategyInContainerSet() {
	s.Given().
		WorkflowTemplate("@testdata/workflow-template-with-containerset.yaml").
		Workflow(`
metadata:
  name: workflow-template-containerset
spec:
  workflowTemplateRef:
    name: containerset-with-retrystrategy
`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowFailed, status.Phase)
		}).
		// Success, no need retry
		ExpectContainerLogs("c1", func(t *testing.T, logs string) {
			count := strings.Count(logs, "capturing logs")
			assert.Equal(t, 1, count)
			assert.Contains(t, logs, "hi")
		}).
		// Command err. No retry logic is entered.
		ExpectContainerLogs("c2", func(t *testing.T, logs string) {
			count := strings.Count(logs, "capturing logs")
			assert.Equal(t, 0, count)
			assert.Contains(t, logs, "executable file not found in $PATH")
		}).
		// Retry when err.
		ExpectContainerLogs("c3", func(t *testing.T, logs string) {
			count := strings.Count(logs, "capturing logs")
			assert.Equal(t, 2, count)
			countFailureInfo := strings.Count(logs, "intentional failure")
			assert.Equal(t, 2, countFailureInfo)
		})
}

func (s *RetryTestSuite) TestRetryNodeAntiAffinity() {
	s.Given().
		Workflow(`
metadata:
  name: test-nodeantiaffinity-strategy
spec:
  entrypoint: main
  templates:
    - name: main
      retryStrategy:
        limit: '1'
        retryPolicy: "Always"
        affinity:
          nodeAntiAffinity: {}
      container:
          name: main
          image: 'argoproj/argosay:v2'
          args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveFailedPod).
		Wait(5 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			if status.Phase == v1alpha1.WorkflowFailed {
				nodeStatus := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(0)")
				nodeStatusRetry := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(1)")
				assert.NotEqual(t, nodeStatus.HostNodeName, nodeStatusRetry.HostNodeName)
			}
			if status.Phase == v1alpha1.WorkflowRunning {
				nodeStatus := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(0)")
				nodeStatusRetry := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(1)")
				assert.Contains(t, nodeStatusRetry.Message, "didn't match Pod's node affinity/selector")
				assert.NotEqual(t, nodeStatus.HostNodeName, nodeStatusRetry.HostNodeName)
			}
		})
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetryTestSuite))
}
