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

type ExecutorPluginsSuite struct {
	fixtures.E2ESuite
}

func (s *ExecutorPluginsSuite) TestTemplateExecutor() {
	s.Given().
		Workflow("@testdata/plugins/executor/template-executor-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, s *wfv1.WorkflowStatus) {
			n := s.Nodes[md.Name]
			assert.Contains(t, n.Message, "Hello")
		})
}

func TestExecutorPluginsSuite(t *testing.T) {
	suite.Run(t, new(ExecutorPluginsSuite))
}
