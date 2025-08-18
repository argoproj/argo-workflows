//go:build executor

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ResourceTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithWorkflow() {
	s.Given().
		Workflow("@executor/k8s-resource-tmpl-with-wf.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithPod() {
	s.Given().
		Workflow("@executor/k8s-resource-tmpl-with-pod.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithArtifact() {
	s.Given().
		Workflow("@executor/k8s-resource-tmpl-with-artifact.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithOutputs() {
	s.Given().
		Workflow("@testdata/resource-templates/outputs.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			outputs := status.Nodes[md.Name].Outputs
			require.NotNil(t, outputs)
			parameters := outputs.Parameters
			require.Len(t, parameters, 2)
			assert.Equal(t, "my-pod", parameters[0].Value.String(), "metadata.name is capture for json")
			assert.Equal(t, "my-pod", parameters[1].Value.String(), "metadata.name is capture for jq")
			for _, value := range status.TaskResultsCompletionStatus {
				assert.True(t, value)
			}
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateAutomountServiceAccountTokenDisabled() {
	s.Given().
		Workflow("@executor/k8s-resource-tmpl-with-automountservicetoken-disabled.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateFailed() {
	s.Given().
		Workflow("@testdata/resource-templates/failed.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
		})
}

func TestResourceTemplateSuite(t *testing.T) {
	suite.Run(t, new(ResourceTemplateSuite))
}
