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
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-resource-tmpl-with-wf-
spec:
  entrypoint: main
  templates:
    - name: main
      resource:
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: argoproj.io/v1alpha1
          kind: Workflow
          metadata:
            generateName: k8s-wf-resource-
          spec:
            entrypoint: main
            templates:
              - name: main
                container:
                  image: argoproj/argosay:v2
                  command: ["/argosay"]
`).
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
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-resource-tmpl-with-pod-
spec:
  serviceAccount: argo
  entrypoint: main
  templates:
    - name: main
      serviceAccountName: argo
      resource:
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: v1
          kind: Pod
          metadata:
            generateName: k8s-pod-resource-
          spec:
            serviceAccountName: argo
            containers:
            - name: argosay-container
              image: argoproj/argosay:v2
              command: ["/argosay"]
            restartPolicy: Never
`).
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
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-resource-tmpl-with-artifact-
spec:
  entrypoint: main
  templates:
    - name: main
      inputs:
        artifacts:
        - name: manifest
          path: /tmp/manifestfrom-path.yaml
          raw:
            data: |
              apiVersion: v1
              kind: Pod
              metadata:
                generateName: k8s-pod-resource-
              spec:
                serviceAccountName: argo
                containers:
                - name: argosay-container
                  image: argoproj/argosay:v2
                  command: ["/argosay"]
                restartPolicy: Never
      resource:
        action: create
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifestFrom:
          artifact:
            name: manifest
`).
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
