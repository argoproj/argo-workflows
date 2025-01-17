//go:build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type MalformedResourcesSuite struct {
	fixtures.E2ESuite
}

func (s *MalformedResourcesSuite) TestMalformedWorkflow() {
	s.Given().Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflow.yaml"}, fixtures.ErrorOutput("unknown field \"spec.arguments.parameters.someParam\""))
}

func (s *MalformedResourcesSuite) TestMalformedWorkflowTemplate() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflowtemplate.yaml"}, fixtures.ErrorOutput("unknown field \"spec.arguments.parameters.someParam\"")).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedWorkflowTemplateRef() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflowtemplate.yaml"}, fixtures.ErrorOutput("unknown field \"spec.arguments.parameters.someParam\"")).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-malformed-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.WorkflowError, status.Phase)
			assert.Contains(t, status.Message, "\"malformed\" not found")
		})
}

func (s *MalformedResourcesSuite) TestMalformedClusterWorkflowTemplate() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-clusterworkflowtemplate.yaml"}, fixtures.ErrorOutput("unknown field \"spec.arguments.parameters.someParam\"")).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-clusterworkflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-cluster-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedClusterWorkflowTemplateRef() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-clusterworkflowtemplate.yaml"}, fixtures.ErrorOutput("unknown field \"spec.arguments.parameters.someParam\"")).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-malformed-cluster-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.WorkflowError, status.Phase)
			assert.Contains(t, status.Message, "\"malformed\" not found")
		})
}

func TestMalformedResourcesSuite(t *testing.T) {
	suite.Run(t, new(MalformedResourcesSuite))
}
