package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type SmokeSuite struct {
	fixtures.E2ESuite
}

func (s *SmokeSuite) TestBasicWorkflow() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, wf *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, wf.Phase)
			assert.NotEmpty(t, wf.Nodes)
		})
}

func (s *SmokeSuite) TestArtifactPassing() {
	s.Given().
		Workflow("@smoke/artifact-passing.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *SmokeSuite) TestWorkflowTemplateBasic() {
	s.Given().
		WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
		Workflow("@smoke/hello-world-workflow-tmpl.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(60 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func TestSmokeSuite(t *testing.T) {
	suite.Run(t, new(SmokeSuite))
}
