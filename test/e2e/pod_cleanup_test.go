//go:build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type PodCleanupSuite struct {
	fixtures.E2ESuite
}

func (s *PodCleanupSuite) TestNone() {
	s.Given().
		Workflow("@functional/test-pod-cleanup.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnPodCompletion() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-completion.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-completion.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
}

func (s *PodCleanupSuite) TestOnPodCompletionLabelSelected() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-completion-label-selected.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-completion-label-selected.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodCompleted)
	})
}

func (s *PodCleanupSuite) TestOnPodSuccess() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-success.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodCompleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-success.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
}

func (s *PodCleanupSuite) TestOnPodSuccessLabelNotMatch() {
	s.Given().
		Workflow("@functional/test-pod-cleanup-on-pod-success-label-not-match.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnPodSuccessLabelMatch() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-success-label-match.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodCompleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow("@functional/test-pod-cleanup-on-pod-success-label-match.yaml").
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
}

func (s *PodCleanupSuite) TestOnWorkflowCompletion() {
	s.Given().
		Workflow("@functional/test-pod-cleanup-on-workflow-completion.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowCompletionLabelNotMatch() {
	s.Given().
		Workflow("@functional/test-pod-cleanup-on-workflow-completion-label-not-match.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnWorkflowCompletionLabelMatch() {
	s.Given().
		Workflow("@functional/test-pod-cleanup-on-workflow-completion-label-match.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowSuccess() {
	s.Given().
		Workflow("@functional/test-pod-cleanup-on-workflow-success.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowSuccessLabelNotMatch() {
	s.Given().
		Workflow("@functional/test-pod-cleanup-on-workflow-success-label-not-match.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnWorkflowSuccessLabelMatch() {
	s.Given().
		Workflow("@functional/test-pod-cleanup-on-workflow-success-label-match.yaml").
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowTemplate() {
	s.Given().
		WorkflowTemplate("@functional/test-pod-cleanup.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForPod(fixtures.PodDeleted)
}

func TestPodCleanupSuite(t *testing.T) {
	suite.Run(t, new(PodCleanupSuite))
}
