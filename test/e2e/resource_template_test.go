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

func TestResourceTemplateSuite(t *testing.T) {
	suite.Run(t, new(ResourceTemplateSuite))
}
