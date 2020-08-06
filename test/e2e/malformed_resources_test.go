// +build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/workflow/common"
)

type MalformedResourcesSuite struct {
	fixtures.E2ESuite
}

func (s *MalformedResourcesSuite) AfterTest(suiteName, testName string) {

	// delete any malformed items first, as they'll break later clean-up
	_, err := fixtures.Exec("kubectl", "-n", fixtures.Namespace, "delete", "workflows,workflowtemplates,clusterworkflowtemplates,cronworkflows", "-l", "argo-e2e=malformed")
	s.CheckError(err)

	s.E2ESuite.AfterTest(suiteName, testName)
}

func (s *MalformedResourcesSuite) TestMalformedWorkflow() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflow.yaml"}, fixtures.NoError).
		WorkflowName("malformed").
		When().
		// it is not possible to wait for this to finish, because it is malformed
		Wait(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "malformed", metadata.Name)
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedCronWorkflow() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-cronworkflow.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-cronworkflow.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow(1*time.Minute + 15*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Labels[common.LabelKeyCronWorkflow])
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedWorkflowTemplate() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedWorkflowTemplateRef() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-malformed-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.NodeError, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedClusterWorkflowTemplate() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-clusterworkflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-clusterworkflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-cluster-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *MalformedResourcesSuite) TestMalformedClusterWorkflowTemplateRef() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-clusterworkflowtemplate.yaml"}, fixtures.NoError).
		Exec("kubectl", []string{"apply", "-f", "testdata/wellformed/wellformed-workflow-with-malformed-cluster-workflow-template-ref.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.NodeError, status.Phase)
		})
}

func TestMalformedResourcesSuite(t *testing.T) {
	suite.Run(t, new(MalformedResourcesSuite))
}
