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

type ProgressSuite struct {
	fixtures.E2ESuite
}

func (s *ProgressSuite) TestProgress() {
	s.Given().
		Workflow("@testdata/progress-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.Equal(t, wfv1.Progress("50/100"), status.Progress)
			// DAG
			assert.Equal(t, wfv1.Progress("50/100"), status.Nodes[metadata.Name].Progress)
			// Pod
			podNode := status.Nodes.FindByDisplayName("progress")
			assert.Equal(t, wfv1.Progress("50/100"), podNode.Progress)
			assert.Equal(t, "my-message", podNode.Message)
		})
}

func TestProgressSuite(t *testing.T) {
	suite.Run(t, new(ProgressSuite))
}
