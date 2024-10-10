package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type CancelSuite struct {
	fixtures.E2ESuite
}

func (s *CancelSuite) TestNone() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-cancel-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
		args: ["sleep", "300s"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		CancelWorkflow().
		WaitForWorkflow(fixtures.ToBeCancelled).
		Then().
		ExpectWorkflow(func(t *testing.T, meta *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowCancelled, status.Phase)
			assert.Equal(t, wfv1.NodeCancelled, status.Nodes.FindByDisplayName("main").Phase)
		})
}
