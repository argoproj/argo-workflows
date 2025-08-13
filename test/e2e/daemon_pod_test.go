//go:build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type DaemonPodSuite struct {
	fixtures.E2ESuite
}

func (s *DaemonPodSuite) TestWorkflowCompletesIfContainsDaemonPod() {
	s.Given().
		Workflow("@functional/whalesay.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.False(t, status.FinishedAt.IsZero())
		})
}

func (s *DaemonPodSuite) TestDaemonFromWorkflowTemplate() {
	s.Given().
		WorkflowTemplate("@functional/daemon.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *DaemonPodSuite) TestDaemonFromClusterWorkflowTemplate() {
	s.Given().
		ClusterWorkflowTemplate(`
metadata:
  name: daemon
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
        - name: redis
          template: redis-tmpl
        - name: whale
          dependencies: [redis]
          template: whale-tmpl
  - name: redis-tmpl
    daemon: true
    container:
      image: argoproj/argosay:v2
      args: ["sleep", "100s"]
  - name: whale-tmpl
    container:
      image: argoproj/argosay:v2
`).
		When().
		CreateClusterWorkflowTemplates().
		SubmitWorkflowsFromClusterWorkflowTemplates().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *DaemonPodSuite) TestDaemonTemplateRef() {
	s.Given().
		WorkflowTemplate("@functional/broken-pipeline.yaml").
		WorkflowTemplate("@functional/run-tests-broken.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *DaemonPodSuite) TestMarkDaemonedPodSucceeded() {
	s.Given().
		Workflow("@testdata/daemoned-pod-completed.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			node := status.Nodes.FindByDisplayName("daemoned")
			require.NotNil(t, node)
			assert.Equal(t, v1alpha1.NodeSucceeded, node.Phase)
		})
}

func (s *DaemonPodSuite) TestDaemonPodRetry() {
	s.Given().
		Workflow("@functional/daemon-retry.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			failedNode := status.Nodes.FindByDisplayName("daemoned(0)")
			succeededNode := status.Nodes.FindByDisplayName("daemoned(1)")
			require.NotNil(t, failedNode)
			require.NotNil(t, succeededNode)
			assert.Equal(t, v1alpha1.NodeFailed, failedNode.Phase)
			assert.Equal(t, v1alpha1.NodeSucceeded, succeededNode.Phase)
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		})
}

func TestDaemonPodSuite(t *testing.T) {
	suite.Run(t, new(DaemonPodSuite))
}
