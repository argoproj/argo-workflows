//go:build executor
// +build executor

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ArtifactsSuite struct {
	fixtures.E2ESuite
}

func (s *ArtifactsSuite) TestInputOnMount() {
	s.Given().
		Workflow("@testdata/input-on-mount-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestOutputOnMount() {
	s.Given().
		Workflow("@testdata/output-on-mount-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestOutputOnInput() {
	s.Need(fixtures.BaseLayerArtifacts) // I believe this would work on both K8S and Kubelet, but validation does not allow it
	s.Given().
		Workflow("@testdata/output-on-input-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestArtifactPassing() {
	s.Need(fixtures.BaseLayerArtifacts)
	s.Given().
		Workflow("@smoke/artifact-passing.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestDefaultParameterOutputs() {
	s.Need(fixtures.BaseLayerArtifacts)
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: default-params-
spec:
  entrypoint: start
  templates:
  - name: start
    steps:
      - - name: generate-1
          template: generate
      - - name: generate-2
          when: "True == False"
          template: generate
    outputs:
      parameters:
        - name: nested-out-parameter
          valueFrom:
            default: "Default value"
            parameter: "{{steps.generate-2.outputs.parameters.out-parameter}}"

  - name: generate
    container:
      image: argoproj/argosay:v2
      args: [echo, my-output-parameter, /tmp/my-output-parameter.txt]
    outputs:
      parameters:
      - name: out-parameter
        valueFrom:
          path: /tmp/my-output-parameter.txt
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.True(t, status.Nodes.Any(func(node wfv1.NodeStatus) bool {
				if node.Outputs != nil {
					for _, param := range node.Outputs.Parameters {
						if param.Value != nil && param.Value.String() == "Default value" {
							return true
						}
					}
				}
				return false
			}))
		})
}

func (s *ArtifactsSuite) TestSameInputOutputPathOptionalArtifact() {
	s.Need(fixtures.BaseLayerArtifacts)
	s.Given().
		Workflow("@testdata/same-input-output-path-optional.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestOutputResult() {
	s.Given().
		Workflow("@testdata/output-result-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			n := status.Nodes.FindByDisplayName("a")
			if assert.NotNil(t, n) {
				assert.NotNil(t, n.Outputs.ExitCode)
				assert.NotNil(t, n.Outputs.Result)
			}
		})
}

func (s *ArtifactsSuite) TestMainLog() {
	s.Run("Basic", func() {
		s.Given().
			Workflow("@testdata/basic-workflow.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				n := status.Nodes[m.Name]
				if assert.NotNil(t, n) {
					assert.Len(t, n.Outputs.Artifacts, 1)
				}
			})
	})
	s.Need(fixtures.None(fixtures.Kubelet))
	s.Run("ActiveDeadlineSeconds", func() {
		s.Given().
			Workflow("@expectedfailures/timeouts-step.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeFailed).
			Then().
			ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				n := status.Nodes[m.Name]
				if assert.NotNil(t, n.Outputs) {
					assert.Len(t, n.Outputs.Artifacts, 1)
				}
			})
	})
}

func TestArtifactsSuite(t *testing.T) {
	suite.Run(t, new(ArtifactsSuite))
}
