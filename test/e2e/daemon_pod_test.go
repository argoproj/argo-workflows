//go:build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		Workflow(`
metadata:
  generateName: whalesay-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
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
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.False(t, status.FinishedAt.IsZero())
		})
}

func (s *DaemonPodSuite) TestDaemonFromWorkflowTemplate() {
	s.Given().
		WorkflowTemplate(`
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
		WorkflowTemplate(`
metadata:
  name: broken-pipeline
spec:
  entrypoint: main
  templates:
  - name: do-something
    container:
      image: argoproj/argosay:v2
  - name: main
    dag:
      tasks:
        - name: do-something
          template: do-something
        - name: run-tests-broken
          depends: "do-something"
          templateRef:
            name: run-tests-broken
            template: main
`).
		WorkflowTemplate(`
metadata:
  name: run-tests-broken
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
      - - name: postgres
          template: postgres
      - - name: run-tests-broken
          template: run-tests-broken
  - name: run-tests-broken
    container:
      image: argoproj/argosay:v2
  - name: postgres
    daemon: true
    container:
      image: argoproj/argosay:v2
      args: ["sleep", "100s"]
      name: database`).
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
			if assert.NotNil(t, node) {
				assert.Equal(t, v1alpha1.NodeSucceeded, node.Phase)
			}
		})
}

func TestDaemonPodSuite(t *testing.T) {
	suite.Run(t, new(DaemonPodSuite))
}
