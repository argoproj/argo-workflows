//go:build functional
// +build functional

package e2e

import (
	"context"
	"io"
	"strings"
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

func (s *RetryTestSuite) TestWorkflowTemplateWithRetryStrategyInContainerSet() {
	var name string
	var ns string
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
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, status.Phase, wfv1.WorkflowFailed)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return status.Name == "workflow-template-containerset"
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			name = pod.GetName()
			ns = pod.GetNamespace()
		})
	// Success, no need retry
	s.Run("ContainerLogs", func() {
		ctx := context.Background()
		podLogOptions := &apiv1.PodLogOptions{Container: "c1"}
		stream, err := s.KubeClient.CoreV1().Pods(ns).GetLogs(name, podLogOptions).Stream(ctx)
		assert.Nil(s.T(), err)
		defer stream.Close()
		logBytes, err := io.ReadAll(stream)
		assert.Nil(s.T(), err)
		output := string(logBytes)
		count := strings.Count(output, "capturing logs")
		assert.Equal(s.T(), 1, count)
		assert.Contains(s.T(), output, "hi")
	})
	// Command err. No retry logic is entered.
	s.Run("ContainerLogs", func() {
		ctx := context.Background()
		podLogOptions := &apiv1.PodLogOptions{Container: "c2"}
		stream, err := s.KubeClient.CoreV1().Pods(ns).GetLogs(name, podLogOptions).Stream(ctx)
		assert.Nil(s.T(), err)
		defer stream.Close()
		logBytes, err := io.ReadAll(stream)
		assert.Nil(s.T(), err)
		output := string(logBytes)
		count := strings.Count(output, "capturing logs")
		assert.Equal(s.T(), 0, count)
		assert.Contains(s.T(), output, "executable file not found in $PATH")
	})
	// Retry when err.
	s.Run("ContainerLogs", func() {
		ctx := context.Background()
		podLogOptions := &apiv1.PodLogOptions{Container: "c3"}
		stream, err := s.KubeClient.CoreV1().Pods(ns).GetLogs(name, podLogOptions).Stream(ctx)
		assert.Nil(s.T(), err)
		defer stream.Close()
		logBytes, err := io.ReadAll(stream)
		assert.Nil(s.T(), err)
		output := string(logBytes)
		count := strings.Count(output, "capturing logs")
		assert.Equal(s.T(), 2, count)
		countFailureInfo := strings.Count(output, "intentional failure")
		assert.Equal(s.T(), 2, countFailureInfo)
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
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			if status.Phase == wfv1.WorkflowFailed {
				nodeStatus := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(0)")
				nodeStatusRetry := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(1)")
				assert.NotEqual(t, nodeStatus.HostNodeName, nodeStatusRetry.HostNodeName)
			}
			if status.Phase == wfv1.WorkflowRunning {
				nodeStatus := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(0)")
				nodeStatusRetry := status.Nodes.FindByDisplayName("test-nodeantiaffinity-strategy(1)")
				assert.Contains(t, nodeStatusRetry.Message, "1 node(s) didn't match Pod's node affinity/selector")
				assert.NotEqual(t, nodeStatus.HostNodeName, nodeStatusRetry.HostNodeName)
			}
		})
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetryTestSuite))
}
