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

type ProgressSuite struct {
	fixtures.E2ESuite
}

func (s *ProgressSuite) TestDefaultProgress() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.Progress("1/1"), status.Progress)
			assert.Equal(t, wfv1.Progress("1/1"), status.Nodes[metadata.Name].Progress)
		})
}

func (s *ProgressSuite) TestLoggedProgress() {
	assertProgress := func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus, expectedPhase wfv1.WorkflowPhase, expectedProgress wfv1.Progress) {
		assert.Equal(t, expectedPhase, status.Phase)
		assert.Equal(t, expectedProgress, status.Progress)
		// DAG
		assert.Equal(t, expectedProgress, status.Nodes[metadata.Name].Progress)
		// Pod
		podNode := status.Nodes.FindByDisplayName("progress")
		assert.Equal(t, expectedProgress, podNode.Progress)
	}

	s.Given().
		Workflow("@testdata/progress-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Wait(5 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assertProgress(t, metadata, status, wfv1.WorkflowRunning, "0/100")
		}).
		When().
		Wait(65 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assertProgress(t, metadata, status, wfv1.WorkflowRunning, "50/100")
		}).
		When().
		WaitForWorkflow(10 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assertProgress(t, metadata, status, wfv1.WorkflowSucceeded, "100/100")
		})
}

func TestProgressSuite(t *testing.T) {
	suite.Run(t, new(ProgressSuite))
}
