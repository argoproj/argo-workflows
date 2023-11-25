//go:build functional
// +build functional

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type WorkflowSuite struct {
	fixtures.E2ESuite
}

func (s *WorkflowSuite) TestContainerTemplateAutomountServiceAccountTokenDisabled() {
	s.Given().Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: get-resources-via-container-template-
  namespace: argo
spec:
  serviceAccountName: argo
  automountServiceAccountToken: false
  executor:
    serviceAccountName: argo 
  entrypoint: main
  templates:
    - name: main
      container:
        name: main
        image: bitnami/kubectl
        command:
          - sh
        args:
          - -c
          - |
           kubectl get cm 
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*11).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowSuite) TestScriptTemplateAutomountServiceAccountTokenDisabled() {
	s.Given().Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: get-resources-via-script-template-
  namespace: argo
spec:
  serviceAccountName: argo
  automountServiceAccountToken: false
  executor:
    serviceAccountName: argo
  entrypoint: main
  templates:
    - name: main
      script:
        name: main
        image: bitnami/kubectl
        command:
          - sh
        source:
          kubectl get cm 
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*11).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func TestWorkflowSuite(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
