// +build e2emc

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type MultiClusterSuite struct {
	fixtures.E2ESuite
}

func (s *MultiClusterSuite) SetupSuite() {
	s.E2ESuite.SetupSuite()
	s.Given().
		Exec("k3d", []string{"image", "import", "--cluster=other", "argoproj/argoexec:latest", "argoproj/argosay:v2"}, fixtures.NoError)
}

func (s *MultiClusterSuite) TestNamespaceUnmanaged() {
	s.Given().
		Workflow(`
metadata:
  generateName: namespace-unmanaged-
  labels:
    argo-e2e: true
spec:
  entrypoint: main
  templates:
    - name: main
      namespace: unmanaged
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.Equal(t, "access denied for namespace \"argo\" to un-managed namespace \"unmanaged\"", status.Message)
		})
}

func (s *MultiClusterSuite) TestNamespaceDenied() {
	s.Given().
		Workflow(`
metadata:
  generateName: namespace-denied-
  labels:
    argo-e2e: true
spec:
  entrypoint: main
  templates:
    - name: main
      namespace: default
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.Equal(t, "access denied for namespace \"argo\" to cluster-namespace \"default/default\"", status.Message)
		})
}

func (s *MultiClusterSuite) TestClusterDenied() {
	s.Given().
		Workflow(`
metadata:
  generateName: cluster-denied-
  labels:
    argo-e2e: true
spec:
  entrypoint: main
  templates:
    - name: main
      clusterName: denied
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.Equal(t, "access denied for namespace \"argo\" to cluster-namespace \"denied/argo\"", status.Message)
		})
}

func (s *MultiClusterSuite) TestClusterNotFound() {
	s.Given().
		Workflow(`
metadata:
  generateName: multi-cluster-not-found-
  labels:
    argo-e2e: true
spec:
  entrypoint: main
  templates:
    - name: main
      clusterName: not-found
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.Equal(t, "no cluster named \"not-found\" has been configured", status.Message)
		})
}

func (s *MultiClusterSuite) TestOtherClusters() {
	defer func() {
		s.Given().Exec("kubectl", []string{"--context=k3d-other", "-nargo", "logs", "multi-cluster", "-c", "main"}, fixtures.NoError)
		s.Given().Exec("kubectl", []string{"--context=k3d-other", "-nargo", "logs", "multi-cluster", "-c", "wait"}, fixtures.NoError)
	}()
	s.Given().
		Workflow(`
metadata:
  name: multi-cluster
  labels:
    argo-e2e: true
spec:
  artifactRepositoryRef:
    key: empty
  entrypoint: main
  templates:
    - name: main
      clusterName: other
      namespace: argo
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			thisNode := status.Nodes.FindByDisplayName("multi-cluster")
			if assert.NotNil(t, thisNode) {
				assert.NotEmpty(t, thisNode.ClusterName)
				assert.Empty(t, thisNode.Namespace)
			}
		})
}

func TestMultiClusterSuite(t *testing.T) {
	suite.Run(t, new(MultiClusterSuite))
}
