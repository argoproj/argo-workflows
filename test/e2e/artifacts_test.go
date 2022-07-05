//go:build executor
// +build executor

package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/minio/minio-go/v7"

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
	s.Given().
		Workflow("@testdata/output-on-input-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestArtifactPassing() {
	s.Given().
		Workflow("@smoke/artifact-passing.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestDefaultParameterOutputs() {
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
			ExpectArtifact("-", "main-logs", func(t *testing.T, object *minio.Object, err error) {
				assert.NoError(t, err)
			})
	})
	s.Run("ActiveDeadlineSeconds", func() {
		s.Given().
			Workflow("@expectedfailures/timeouts-step.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeFailed).
			Then().
			ExpectArtifact("-", "main-logs", func(t *testing.T, object *minio.Object, err error) {
				assert.NoError(t, err)
			})
	})
}

func (s *ArtifactsSuite) TestContainersetLogs() {
	s.Run("Basic", func() {
		s.Given().
			Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: containerset-logs-
spec:
  entrypoint: main
  templates:
    - name: main
      containerSet:
        containers:
          - name: a
            image: argoproj/argosay:v2
          - name: b
            image: argoproj/argosay:v2
`).
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				n := status.Nodes[m.Name]
				expectedOutputs := map[string]string{"a-logs": fmt.Sprintf("%s/%s/a.log", m.Name, m.Name), "b-logs": fmt.Sprintf("%s/%s/b.log", m.Name, m.Name)}
				if assert.NotNil(t, n) {
					assert.Equal(t, len(expectedOutputs), len(n.Outputs.Artifacts))
					for _, artifact := range n.Outputs.Artifacts {
						expectedS3Key, found := expectedOutputs[artifact.Name]
						assert.True(t, found)
						assert.Equal(t, expectedS3Key, artifact.S3.Key)
					}
				}
			})
	})
}

func TestArtifactsSuite(t *testing.T) {
	suite.Run(t, new(ArtifactsSuite))
}
