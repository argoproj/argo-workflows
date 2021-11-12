//go:build plugins
// +build plugins

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ControllerPluginsSuite struct {
	fixtures.E2ESuite
}

func (s *ControllerPluginsSuite) TestParameterSubstitutionPlugin() {
	s.Given().
		Workflow("@testdata/plugins/controller/params-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ControllerPluginsSuite) TestWorkflowLifecycleHook() {
	s.Given().
		Workflow("@testdata/plugins/controller/workflow-lifecycle-hook-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, _ *wfv1.WorkflowStatus) {
			assert.Equal(t, "good morning", md.Annotations["hello"])
		})
}

func (s *ControllerPluginsSuite) TestNodeLifecycleHook() {
	s.Given().
		Workflow("@testdata/plugins/controller/controller-plugin-template-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, s *wfv1.WorkflowStatus) {
			n := s.Nodes[md.Name]
			assert.Contains(t, n.Message, "Hello")
		})
}

func TestControllerPluginsSuite(t *testing.T) {
	suite.Run(t, new(ControllerPluginsSuite))
}
