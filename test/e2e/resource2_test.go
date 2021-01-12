// +build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type Resource2Suite struct {
	fixtures.E2ESuite
}

func (s *Resource2Suite) TestResource2() {
	s.Given().
		Workflow(`
metadata:
  generateName: my-ns-
  labels:
    argo-e2e: true
spec:
  entrypoint: main
  templates:
  - name: main
    resource2:
      apiVersion: v1
      kind: ConfigMap
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func TestResource2Suite(t *testing.T) {
	suite.Run(t, new(Resource2Suite))
}
