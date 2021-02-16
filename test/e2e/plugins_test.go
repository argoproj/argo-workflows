// +build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type PluginsSuite struct {
	fixtures.E2ESuite
}

func (s *PluginsSuite) TestHTTPPlugin() {
	s.Need(fixtures.BaseLayerArtifacts)
	s.Given().
		Workflow(`@testdata/plugins/http-plugin-workflow.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, "to be succeeded").
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			rootNode := status.Nodes[m.Name]
			if assert.Len(t, rootNode.Outputs.Parameters, 1) {
				parameter := rootNode.Outputs.Parameters[0]
				assert.Equal(t, "version", parameter.Name)
				assert.Contains(t, parameter.Value.String(), "latest")
			}
		})
}

func (s *PluginsSuite) TestJSONRPCPlugin() {
	s.Need(fixtures.BaseLayerArtifacts)
	s.Given().
		Workflow(`@testdata/plugins/hello-plugin-workflow.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, "to be succeeded").
		Then().
		ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			rootNode := status.Nodes[m.Name]
			assert.Equal(t, "hi", rootNode.Message)
		})
}

func TestPluginsSuite(t *testing.T) {
	suite.Run(t, new(PluginsSuite))
}
