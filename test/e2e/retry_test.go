//go:build functional

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
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
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
		s.Require().NoError(err)
		defer stream.Close()
		logBytes, err := io.ReadAll(stream)
		s.Require().NoError(err)
		output := string(logBytes)
		count := strings.Count(output, "capturing logs")
		s.Equal(1, count)
		s.Contains(output, "hi")
	})
	// Command err. No retry logic is entered.
	s.Run("ContainerLogs", func() {
		ctx := context.Background()
		podLogOptions := &apiv1.PodLogOptions{Container: "c2"}
		stream, err := s.KubeClient.CoreV1().Pods(ns).GetLogs(name, podLogOptions).Stream(ctx)
		s.Require().NoError(err)
		defer stream.Close()
		logBytes, err := io.ReadAll(stream)
		s.Require().NoError(err)
		output := string(logBytes)
		count := strings.Count(output, "capturing logs")
		s.Equal(0, count)
		s.Contains(output, "executable file not found in $PATH")
	})
	// Retry when err.
	s.Run("ContainerLogs", func() {
		ctx := context.Background()
		podLogOptions := &apiv1.PodLogOptions{Container: "c3"}
		stream, err := s.KubeClient.CoreV1().Pods(ns).GetLogs(name, podLogOptions).Stream(ctx)
		s.Require().NoError(err)
		defer stream.Close()
		logBytes, err := io.ReadAll(stream)
		s.Require().NoError(err)
		output := string(logBytes)
		count := strings.Count(output, "capturing logs")
		s.Equal(2, count)
		countFailureInfo := strings.Count(output, "intentional failure")
		s.Equal(2, countFailureInfo)
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
				assert.Contains(t, nodeStatusRetry.Message, "didn't match Pod's node affinity/selector")
				assert.NotEqual(t, nodeStatus.HostNodeName, nodeStatusRetry.HostNodeName)
			}
		})
}

func (s *RetryTestSuite) TestRetryDaemonContainer() {
	s.Given().
		Workflow(`
metadata:
  name: test-stepsdaemonretry-strategy
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: server
        template: server
    - - name: client
        template: client
        arguments:
          parameters:
          - name: server-ip
            value: "{{steps.server.ip}}"
        withSequence:
          count: "3"
  - name: server
    retryStrategy:
      limit: "10"
    daemon: true
    container:
      image: nginx:1.13
      readinessProbe:
      httpGet:
        path: /
        port: 80
      initialDelaySeconds: 2
      timeoutSeconds: 1
  - name: client
    inputs:
      parameters:
      - name: server-ip
    synchronization:
      mutex:
      name: client-{{workflow.uid}}
    container:
      image: appropriate/curl:latest
      command: ["/bin/sh", "-c"]
      args: ["echo curl --silent -G http://{{inputs.parameters.server-ip}}:80/ && curl --silent -G http://{{inputs.parameters.server-ip}}:80/ && sleep 10"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow((fixtures.Condition)(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
				return node.GetTemplateName() == "client" && node.Phase == wfv1.NodeSucceeded
			}), "waiting for at least one client to succeed"
		})).DeleteNodePod("test-stepsdaemonretry-strategy[0].server(0)").
		Wait(10 * time.Second).
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			node := status.Nodes.FindByName("test-stepsdaemonretry-strategy[0].server(1)")
			assert.NotNil(t, node)
		})
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetryTestSuite))
}
