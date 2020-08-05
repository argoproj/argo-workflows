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

func (s *MalformedResourcesSuite) TestMalformedWorkflow() {
	s.Given().
		Exec("kubectl", []string{"apply", "-f", "testdata/malformed/malformed-workflow.yaml"}, fixtures.NoError).
		When().
		WaitForWorkflow(15 * time.Second).
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
		WaitForWorkflow(1 * time.Minute).
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
		WaitForWorkflow(1 * time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "wellformed", metadata.Name)
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func TestMalformedResourcesSuite(t *testing.T) {
	suite.Run(t, new(MalformedResourcesSuite))
}
