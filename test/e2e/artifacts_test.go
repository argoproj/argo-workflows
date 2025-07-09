//go:build executor

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

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

type expectedArtifact struct {
	key        string
	bucketName string
	value      string
}

func (s *ArtifactsSuite) TestGlobalArtifactPassing() {
	for _, tt := range []struct {
		workflowFile     string
		expectedArtifact expectedArtifact
	}{
		{
			workflowFile: "@testdata/global-artifact-passing.yaml",
			expectedArtifact: expectedArtifact{
				key:        "globalArtifact",
				bucketName: "my-bucket-3",
				value:      "01",
			},
		},
		{
			workflowFile: "@testdata/complex-global-artifact-passing.yaml",
			expectedArtifact: expectedArtifact{
				key:        "finalTestUpdate",
				bucketName: "my-bucket-3",
				value:      "Updated testUpdate",
			},
		},
	} {
		then := s.Given().
			Workflow(tt.workflowFile).
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*2).
			Then().
			ExpectWorkflow(func(t *testing.T, objectMeta *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				// Check the global artifact value and see if it equals the expected value.
				c, err := minio.New("localhost:9000", &minio.Options{
					Creds: credentials.NewStaticV4("admin", "password", ""),
				})

				if err != nil {
					t.Error(err)
				}

				object, err := c.GetObject(func() context.Context {
					ctx := context.Background()
					return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
				}(), tt.expectedArtifact.bucketName, tt.expectedArtifact.key, minio.GetObjectOptions{})
				if err != nil {
					t.Error(err)
				}

				buf := new(bytes.Buffer)
				_, err = buf.ReadFrom(object)
				if err != nil {
					t.Error(err)
				}
				value := buf.String()

				assert.Equal(t, tt.expectedArtifact.value, value)
			})

		then.
			When().
			RemoveFinalizers(false)
	}
}

type artifactState struct {
	artifactLocation s3Location

	deletedAtWFCompletion bool
	deletedAtWFDeletion   bool
}

type s3Location struct {
	bucketName string
	// specify one of these two:
	specifiedKey string              // exact key is known
	derivedKey   *artifactDerivedKey // exact key needs to be derived
}

type artifactDerivedKey struct {
	templateName string
	artifactName string
}

func (al *s3Location) getS3Key(wf *wfv1.Workflow) (string, error) {
	if al.specifiedKey == "" && al.derivedKey == nil {
		panic(fmt.Sprintf("invalid artifactLocation: %+v, must have specifiedKey or derivedKey set", al))
	}

	if al.specifiedKey != "" {
		return al.specifiedKey, nil
	}

	// get key by finding the node in the Workflow's NodeStatus and looking at its Artifacts

	// get node name using template
	n := wf.Status.Nodes.Find(func(nodeStatus wfv1.NodeStatus) bool { return nodeStatus.TemplateName == al.derivedKey.templateName })
	if n == nil {
		return "", fmt.Errorf("no node with template name=%q found in workflow %+v", al.derivedKey.templateName, wf)
	}
	for _, a := range n.Outputs.Artifacts {
		if a.Name == al.derivedKey.artifactName {
			if a.S3 == nil {
				return "", fmt.Errorf("didn't find expected S3 field in artifact %q: %+v", al.derivedKey.artifactName, a)
			}
			return a.S3.Key, nil
		}
	}

	return "", fmt.Errorf("artifact named %q not found", al.derivedKey.artifactName)
}

func (s *ArtifactsSuite) TestStoppedWorkflow() {
	s.T().Skip("This test is flaky and will be skipped as a result")

	for _, tt := range []struct {
		workflowFile string
	}{
		{workflowFile: "@testdata/artifactgc/artgc-dag-wf-stopped.yaml"},
		{workflowFile: "@testdata/artifactgc/artgc-dag-wf-stopped-pod-gc-on-pod-completion.yaml"},
	} {
		// Create the minio client for interacting with the bucket.
		c, err := minio.New("localhost:9000", &minio.Options{
			Creds: credentials.NewStaticV4("admin", "password", ""),
		})
		s.Require().NoError(err)

		// Ensure the artifacts aren't in the bucket.
		_, err = c.StatObject(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), "my-bucket-3", "on-deletion-wf-stopped-1", minio.StatObjectOptions{})
		if err == nil {
			err = c.RemoveObject(func() context.Context {
				ctx := context.Background()
				return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
			}(), "my-bucket-3", "on-deletion-wf-stopped-1", minio.RemoveObjectOptions{})
			s.Require().NoError(err)
		}
		_, err = c.StatObject(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), "my-bucket-3", "on-deletion-wf-stopped-2", minio.StatObjectOptions{})
		if err == nil {
			err = c.RemoveObject(func() context.Context {
				ctx := context.Background()
				return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
			}(), "my-bucket-3", "on-deletion-wf-stopped-2", minio.RemoveObjectOptions{})
			s.Require().NoError(err)
		}

		then := s.Given().
			Workflow(tt.workflowFile).
			When().
			Then()

		// Assert the artifacts don't exist.
		then.ExpectArtifactByKey("on-deletion-wf-stopped-1", "my-bucket-3", func(t *testing.T, object minio.ObjectInfo, err error) {
			require.Error(t, err)
		})
		then.ExpectArtifactByKey("on-deletion-wf-stopped-2", "my-bucket-3", func(t *testing.T, object minio.ObjectInfo, err error) {
			require.Error(t, err)
		})

		when := then.When().
			SubmitWorkflow().
			WaitForWorkflow(
				fixtures.WorkflowCompletionOkay(true),
				fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {

					condition := "for artifacts to exist"

					_, err1 := c.StatObject(func() context.Context {
						ctx := context.Background()
						return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
					}(), "my-bucket-3", "on-deletion-wf-stopped-1", minio.StatObjectOptions{})
					_, err2 := c.StatObject(func() context.Context {
						ctx := context.Background()
						return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
					}(), "my-bucket-3", "on-deletion-wf-stopped-2", minio.StatObjectOptions{})

					if err1 == nil && err2 == nil {
						return true, condition
					}

					return false, condition
				}))

		then = when.Then()

		// Assert artifact exists
		then.ExpectArtifactByKey("on-deletion-wf-stopped-1", "my-bucket-3", func(t *testing.T, object minio.ObjectInfo, err error) {
			require.NoError(t, err)
		})
		then.ExpectArtifactByKey("on-deletion-wf-stopped-2", "my-bucket-3", func(t *testing.T, object minio.ObjectInfo, err error) {
			require.NoError(t, err)
		})

		when = then.When()

		when.
			DeleteWorkflow().
			WaitForWorkflowDeletion()

		then = when.Then()

		// Assert the artifacts don't exist.
		then.ExpectArtifactByKey("on-deletion-wf-stopped-1", "my-bucket-3", func(t *testing.T, object minio.ObjectInfo, err error) {
			require.Error(t, err)
		})
		then.ExpectArtifactByKey("on-deletion-wf-stopped-2", "my-bucket-3", func(t *testing.T, object minio.ObjectInfo, err error) {
			require.Error(t, err)
		})

		when = then.When()

		// Remove the finalizers so the workflow gets deleted in case the test failed.
		when.RemoveFinalizers(false)
	}
}

func (s *ArtifactsSuite) TestDeleteWorkflow() {
	when := s.Given().
		Workflow("@testdata/artifactgc/artgc-dag-wf-self-delete.yaml").
		When().
		SubmitWorkflow()

	then := when.
		WaitForWorkflow(fixtures.ToBeCompleted).
		Then().
		ExpectWorkflow(func(t *testing.T, objectMeta *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Contains(t, objectMeta.Finalizers, common.FinalizerArtifactGC)
		})

	when = then.When()

	when.WaitForWorkflowDeletion()

	when.RemoveFinalizers(false)
}

func (s *ArtifactsSuite) TestArtifactGC() {

	s.Given().
		WorkflowTemplate("@testdata/artifactgc/artgc-template.yaml").
		WorkflowTemplate("@testdata/artifactgc/artgc-template-2.yaml").
		WorkflowTemplate("@testdata/artifactgc/artgc-template-ref-template.yaml").
		WorkflowTemplate("@testdata/artifactgc/artgc-template-no-gc.yaml").
		When().
		CreateWorkflowTemplates()

	for _, tt := range []struct {
		workflowFile                 string
		hasGC                        bool
		workflowShouldSucceed        bool
		expectedArtifacts            []artifactState
		expectedGCPodsOnWFCompletion int
	}{
		{
			workflowFile:                 "@testdata/artifactgc/artgc-multi-strategy-multi-anno.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 2,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "first-on-completion-1"}, true, false},
				{s3Location{bucketName: "my-bucket-3", specifiedKey: "first-on-completion-2"}, true, false},
				{s3Location{bucketName: "my-bucket-3", specifiedKey: "first-no-deletion"}, false, false},
				{s3Location{bucketName: "my-bucket-3", specifiedKey: "second-on-deletion"}, false, true},
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "second-on-completion"}, true, false},
			},
		},
		// entire Workflow based on a WorkflowTemplate
		{
			workflowFile:                 "@testdata/artifactgc/artgc-from-template.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 1,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-completion"}, true, false},
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-deletion"}, false, true},
			},
		},
		// entire Workflow based on a WorkflowTemplate
		{
			workflowFile:                 "@testdata/artifactgc/artgc-from-template-2.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 1,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-completion"}, true, false},
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-deletion"}, false, true},
			},
		},
		// Step in Workflow references a WorkflowTemplate's template
		{
			workflowFile:                 "@testdata/artifactgc/artgc-step-wf-tmpl.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 1,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-completion"}, true, false},
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-deletion"}, false, true},
			},
		},
		// Step in Workflow references a WorkflowTemplate's template
		{
			workflowFile:                 "@testdata/artifactgc/artgc-step-wf-tmpl-2.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 1,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-completion"}, true, false},
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-deletion"}, false, false},
			},
		},
		// entire Workflow based on a WorkflowTemplate which has a Step that references another WorkflowTemplate's template
		{
			workflowFile:                 "@testdata/artifactgc/artgc-from-ref-template.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 1,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-completion"}, true, false},
				{s3Location{bucketName: "my-bucket-2", specifiedKey: "on-deletion"}, false, true},
			},
		},
		// Step in Workflow references a WorkflowTemplate's template
		// Workflow defines ArtifactGC but all artifacts override with "Never" so Artifact GC should not be done
		{
			workflowFile:                 "@testdata/artifactgc/artgc-step-wf-tmpl-no-gc.yaml",
			hasGC:                        false,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 0,
			expectedArtifacts:            []artifactState{},
		},
		// Workflow fails to write an artifact that's been defined as an Output
		{
			workflowFile:                 "@testdata/artifactgc/artgc-non-optional-artifact-not-written.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        false, // artifact not being present causes Workflow to fail
			expectedGCPodsOnWFCompletion: 0,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket", derivedKey: &artifactDerivedKey{templateName: "artifact-written", artifactName: "present"}}, false, true},
				{s3Location{bucketName: "my-bucket", derivedKey: &artifactDerivedKey{templateName: "some-artifacts-not-written", artifactName: "present"}}, false, true},
			},
		},
		// Workflow doesn't write an artifact that's been defined as an Output, but it's an Optional artifact, so Workflow succeeds
		{
			workflowFile:                 "@testdata/artifactgc/artgc-optional-artifact-not-written.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        true,
			expectedGCPodsOnWFCompletion: 0,
			expectedArtifacts: []artifactState{
				{s3Location{bucketName: "my-bucket", derivedKey: &artifactDerivedKey{templateName: "artifact-written", artifactName: "present"}}, false, true},
				{s3Location{bucketName: "my-bucket", derivedKey: &artifactDerivedKey{templateName: "some-artifacts-not-written", artifactName: "present"}}, false, true},
			},
		},
		// Workflow defined output artifact but execution failed, no artifacts to be gced
		{
			workflowFile:                 "@testdata/artifactgc/artgc-artifact-not-written-failed.yaml",
			hasGC:                        true,
			workflowShouldSucceed:        false,
			expectedGCPodsOnWFCompletion: 0,
			expectedArtifacts:            []artifactState{},
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
				if tt.hasGC {
					assert.Contains(t, objectMeta.Finalizers, common.FinalizerArtifactGC)
				}
			})

		if tt.workflowShouldSucceed && when.WorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError
		}) {
			fmt.Println("can't reliably verify Artifact GC since workflow failed")
			when.RemoveFinalizers(false)
			continue
		}

		// wait for all pods to have started and been completed and recouped
		when.
			WaitForWorkflow(
				fixtures.WorkflowCompletionOkay(true),
				fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
					return (len(wf.Status.ArtifactGCStatus.PodsRecouped) >= tt.expectedGCPodsOnWFCompletion) || (tt.expectedGCPodsOnWFCompletion == 0),
						fmt.Sprintf("for all %d pods to have been recouped", tt.expectedGCPodsOnWFCompletion)
				}))

		then := when.Then()

		// verify that the artifacts that should have been deleted at completion time were
		for _, expectedArtifact := range tt.expectedArtifacts {
			artifactKey, err := expectedArtifact.artifactLocation.getS3Key(when.GetWorkflow())
			fmt.Printf("artifact key: %q\n", artifactKey)
			if err != nil {
				panic(err)
			}
			if expectedArtifact.deletedAtWFCompletion {
				fmt.Printf("verifying artifact %s is deleted at completion time\n", artifactKey)
				then.ExpectArtifactByKey(artifactKey, expectedArtifact.artifactLocation.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					require.Error(t, err)
				})
			} else {
				fmt.Printf("verifying artifact %s is not deleted at completion time\n", artifactKey)
				then.ExpectArtifactByKey(artifactKey, expectedArtifact.artifactLocation.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					require.NoError(t, err)
				})
			}
		}

		fmt.Println("deleting workflow; verifying that Artifact GC finalizer gets removed")

		when.
			DeleteWorkflow().
			WaitForWorkflowDeletion().
			Then().
			ExpectWorkflowDeleted()

		when = when.RemoveFinalizers(false) // just in case - if the above test failed we need to forcibly remove the finalizer for Artifact GC

		then = when.Then()

		for _, expectedArtifact := range tt.expectedArtifacts {
			artifactKey, err := expectedArtifact.artifactLocation.getS3Key(when.GetWorkflow())
			fmt.Printf("artifact key: %q\n", artifactKey)
			if err != nil {
				panic(err)
			}

			if expectedArtifact.deletedAtWFCompletion { // already checked this
				continue
			}
			if expectedArtifact.deletedAtWFDeletion {
				fmt.Printf("verifying artifact %s is deleted\n", artifactKey)
				then.ExpectArtifactByKey(artifactKey, expectedArtifact.artifactLocation.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					require.Error(t, err)
				})
			} else {
				fmt.Printf("verifying artifact %s is not deleted\n", artifactKey)
				then.ExpectArtifactByKey(artifactKey, expectedArtifact.artifactLocation.bucketName, func(t *testing.T, object minio.ObjectInfo, err error) {
					require.NoError(t, err)
				})
			}
		}
	}
}

var insufficientRoleWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: art-gc-simple-
spec:
  entrypoint: main
  artifactGC:
    forceFinalizerRemoval: true
  templates:
  - name: main
    container:
      image: argoproj/argosay:v2
      command:
        - sh
        - -c
      args:
        - |
          echo "can throw this away" > /tmp/temporary-artifact-on-completion.txt
    outputs:
      artifacts:
        - name: temporary-artifact-on-completion
          path: /tmp/temporary-artifact-on-completion.txt
          s3:
            key: temporary-artifact-on-completion.txt
          artifactGC:
            strategy: OnWorkflowCompletion
            serviceAccountName: artgc-role-test-sa
`

// create a ServiceAccount which won't be tied to the artifactgc role and attempt to use that service account in the GC Pod
// Want to verify that this causes the ArtifactGCError Condition in the Workflow
func (s *ArtifactsSuite) TestInsufficientRole() {
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	_, err := s.KubeClient.CoreV1().ServiceAccounts(fixtures.Namespace).Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "artgc-role-test-sa"}}, metav1.CreateOptions{})
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		_ = s.KubeClient.CoreV1().ServiceAccounts(fixtures.Namespace).Delete(ctx, "artgc-role-test-sa", metav1.DeleteOptions{})
	})

	// We can test this failure case in 2 ways
	// 1. Workflow sets ForceFinalizerRemoval to false, so finalizer is still present after failure
	// 2. Workflow sets ForceFinalizerRemoval to true, so finalizer isn't present after failure
	tests := []struct { // I suppose this could just be a slice of bool, but making it a struct in case we want to expand it
		forceFinalizerRemoval bool
	}{
		{
			forceFinalizerRemoval: true,
		},
		{
			forceFinalizerRemoval: false,
		},
	}

	for _, tt := range tests {
		// unmarshal and marshal the yaml so we can modify the Workflow spec
		var workflow wfv1.Workflow
		err = yaml.Unmarshal([]byte(insufficientRoleWorkflow), &workflow)
		if err != nil {
			s.Fail(err.Error())
		}

		workflow.Spec.ArtifactGC.ForceFinalizerRemoval = tt.forceFinalizerRemoval
		modifiedWorkflow, err := yaml.Marshal(&workflow)
		if err != nil {
			s.Fail(err.Error())
		}

		// Submit the Workflow
		when := s.Given().Workflow(string(modifiedWorkflow)).
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeCompleted)

		// if the Workflow fails for some reason outside of our control, we can't complete this test
		if when.WorkflowCondition(func(wf *wfv1.Workflow) bool {
			return wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError
		}) {
			fmt.Println("can't reliably verify Artifact GC (Insufficient Role test) since workflow failed")
			when.RemoveFinalizers(false)
			return
		}

		// Once Workflow completes, check its result
		when.WaitForWorkflow(
			fixtures.WorkflowCompletionOkay(true),
			fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
				return wf.Status.ArtifactGCStatus != nil &&
					len(wf.Status.ArtifactGCStatus.PodsRecouped) == 1, "for pod to have been recouped"
			})).
			Then().
			ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				failCondition := false
				for _, c := range status.Conditions {
					if c.Type == wfv1.ConditionTypeArtifactGCError {
						failCondition = true
					}
				}
				assert.True(t, failCondition)
			}).
			ExpectWorkflow(func(t *testing.T, meta *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
				if tt.forceFinalizerRemoval {
					s.NotContains(meta.Finalizers, common.FinalizerArtifactGC)
				} else {
					s.Contains(meta.Finalizers, common.FinalizerArtifactGC)
				}
			}).
			When().
			RemoveFinalizers(true)
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
			require.NotNil(t, n)
			assert.NotNil(t, n.Outputs.ExitCode)
			assert.NotNil(t, n.Outputs.Result)
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
				require.NoError(t, err)
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
				require.NoError(t, err)
			})
	})
}

func (s *ArtifactsSuite) TestResourceLog() {
	s.Run("Basic", func() {
		s.Given().
			Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: resource-tmpl-wf-
spec:
  entrypoint: main
  templates:
    - name: main
      resource:
        action: create
        successCondition: status.phase == Succeeded
        setOwnerReference: true
        manifest: |
          apiVersion: argoproj.io/v1alpha1
          kind: Workflow
          metadata:
            generateName: hello-world-
            labels:
              workflows.argoproj.io/test: "true"
          spec:
            entrypoint: whalesay
            templates:
              - name: whalesay
                container:
                  image: argoproj/argosay:v2
                  command: [sh, -c]
                  args: [echo, ":) Hello Argo!"]
`).
			When().
			SubmitWorkflow().
			WaitForWorkflow(fixtures.ToBeSucceeded).
			Then().
			ExpectArtifact("-", "main-logs", "my-bucket", func(t *testing.T, object minio.ObjectInfo, err error) {
				require.NoError(t, err)
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
				require.NotNil(t, n)
				assert.Equal(t, expectedOutputs, n.Outputs)
			})
	})
}

func (s *ArtifactsSuite) TestGitArtifactDepthClone() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: git-depth-
spec:
  entrypoint: git-depth
  templates:
  - name: git-depth
    inputs:
      artifacts:
      - name: git-repo
        path: /tmp/git
        git:
          repo: https://github.com/argoproj-labs/go-git.git
          revision: master
          depth: 1
    container:
      image: argoproj/argosay:v2
      command: [sh, -c]
      args: ["ls -l"]
      workingDir: /tmp/git
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *ArtifactsSuite) TestArtifactEphemeralVolume() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-volume-claim-
spec:
  entrypoint: artifact-volume-claim
  volumeClaimTemplates:
    - metadata:
        name: vol
      spec:
        accessModes: [ "ReadWriteOnce" ]
        resources:
          requests:
            storage: 1Mi
  templates:
  - name: artifact-volume-claim
    inputs:
      artifacts:
      - name: artifact-volume-claim
        path: /tmp/input/input.txt
        raw:
          data: abc
    container:
      image: argoproj/argosay:v2
      command: [sh, -c]
      args: ["ls -l"]
      workingDir: /tmp
      volumeMounts:
      - name: vol
        mountPath: /tmp
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestArtifactsSuite(t *testing.T) {
	suite.Run(t, new(ArtifactsSuite))
}
