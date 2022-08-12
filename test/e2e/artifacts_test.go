//go:build executor
// +build executor

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/common"

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

type artifactState struct {
	key                   string
	bucketName            string
	deletedAtWFCompletion bool
	deletedAtWFDeletion   bool
}

func (s *ArtifactsSuite) TestArtifactGC() {

	s.Given().
		WorkflowTemplate("@testdata/artifactgc/artgc-template.yaml").
		When().
		CreateWorkflowTemplates()

	for _, tt := range []struct {
		workflowFile      string
		expectedArtifacts []artifactState
	}{
		{
			workflowFile: "@testdata/artifactgc/artgc-multi-strategy-multi-anno.yaml",
			expectedArtifacts: []artifactState{
				artifactState{"first-on-completion-1", "my-bucket-2", true, false},
				artifactState{"first-on-completion-2", "my-bucket-3", true, false},
				artifactState{"first-no-deletion", "my-bucket-3", false, false},
				artifactState{"second-on-deletion", "my-bucket-3", false, true},
				artifactState{"second-on-completion", "my-bucket-2", true, false},
			},
		},
		{
			workflowFile: "@testdata/artifactgc/artgc-from-template.yaml",
			expectedArtifacts: []artifactState{
				artifactState{"on-completion", "my-bucket-2", true, false},
				artifactState{"on-deletion", "my-bucket-2", false, true},
			},
		},
		{
			workflowFile: "@testdata/artifactgc/artgc-step-wf-tmpl.yaml",
			expectedArtifacts: []artifactState{
				artifactState{"on-completion", "my-bucket-2", true, false},
				artifactState{"on-deletion", "my-bucket-2", false, true},
			},
		},
	} {
		// for each test make sure that:
		// 1. the finalizer gets added
		// 2. the artifacts are deleted at the right time
		// 3. the finalizer gets removed after all artifacts are deleted
		// (note that in order to verify that the finalizer has been added once the Workflow's been submitted,
		// we need it to still be there after being submitted, so each of the following tests includes at least one
		// 'OnWorkflowDeletion' strategy)

		when := s.Given().
			Workflow(tt.workflowFile).
			When().
			SubmitWorkflow()
		when.
			WaitForWorkflow(fixtures.ToBeCompleted).
			Then().
			ExpectWorkflow(func(t *testing.T, objectMeta *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				assert.Contains(t, objectMeta.Finalizers, common.FinalizerArtifactGC)
			})

		then := when.Then()

		time.Sleep(5 * time.Second) // todo: do we want to do this or not? better if we could wait for Artifact GC strategy processed...

		for _, expectedArtifact := range tt.expectedArtifacts {
			if expectedArtifact.deletedAtWFCompletion {
				fmt.Printf("verifying artifact %s is deleted at completion time\n", expectedArtifact.key)
				then.ExpectArtifactByKey(expectedArtifact.key, expectedArtifact.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					assert.NotNil(t, err)
				})
			} else {
				fmt.Printf("verifying artifact %s is not deleted at completion time\n", expectedArtifact.key)
				then.ExpectArtifactByKey(expectedArtifact.key, expectedArtifact.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					assert.Nil(t, err)
				})
			}
		}

		fmt.Println("deleting workflow; verifying that Artifact GC finalizer gets removed")

		when.
			DeleteWorkflow().
			WaitForWorkflowDeletion()

		when = when.RemoveFinalizers(false) // just in case - if the above test failed we need to forcibly remove the finalizer for Artifact GC

		then = when.Then()

		for _, expectedArtifact := range tt.expectedArtifacts {
			if expectedArtifact.deletedAtWFDeletion || expectedArtifact.deletedAtWFCompletion {
				fmt.Printf("verifying artifact %s is deleted\n", expectedArtifact.key)
				then.ExpectArtifactByKey(expectedArtifact.key, expectedArtifact.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					assert.NotNil(t, err)
				})
			} else {
				fmt.Printf("verifying artifact %s is not deleted\n", expectedArtifact.key)
				then.ExpectArtifactByKey(expectedArtifact.key, expectedArtifact.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					assert.Nil(t, err)
				})
			}
		}
	}
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
			ExpectArtifact("-", "main-logs", "my-bucket", func(t *testing.T, object minio.ObjectInfo, err error) {
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
			ExpectArtifact("-", "main-logs", "my-bucket", func(t *testing.T, object minio.ObjectInfo, err error) {
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
				expectedOutputs := &wfv1.Outputs{
					Artifacts: wfv1.Artifacts{
						{
							Name: "a-logs",
							ArtifactLocation: wfv1.ArtifactLocation{
								S3: &wfv1.S3Artifact{
									Key: fmt.Sprintf("%s/%s/a.log", m.Name, m.Name),
								},
							},
						},
						{
							Name: "b-logs",
							ArtifactLocation: wfv1.ArtifactLocation{
								S3: &wfv1.S3Artifact{
									Key: fmt.Sprintf("%s/%s/b.log", m.Name, m.Name),
								},
							},
						},
					},
				}
				if assert.NotNil(t, n) {
					assert.Equal(t, n.Outputs, expectedOutputs)
				}
			})
	})
}

func TestArtifactsSuite(t *testing.T) {
	suite.Run(t, new(ArtifactsSuite))
}
