//go:build functional

package e2e

import (
	"testing"
	"time"

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
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflow.yaml"}, fixtures.NoError).
		WorkflowName("malformed").
		When().
		// it is not possible to wait for this to finish, because it is malformed
		Wait(3 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "malformed", metadata.Name)
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedWorkflowTemplate() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflowtemplate.yaml"}, fixtures.NoError).
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
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-malformed-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.WorkflowError, status.Phase)
			assert.Contains(t, status.Message, "malformed workflow template")
		})
}

func (s *MalformedResourcesSuite) TestMalformedClusterWorkflowTemplate() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-clusterworkflowtemplate.yaml"}, fixtures.NoError).
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
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-clusterworkflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-malformed-cluster-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.WorkflowError, status.Phase)
			assert.Contains(t, status.Message, "malformed cluster workflow template")
		})
}

func TestMalformedResourcesSuite(t *testing.T) {
	suite.Run(t, new(MalformedResourcesSuite))
}
