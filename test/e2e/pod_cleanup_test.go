//go:build functional
// +build functional

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
		Workflow(`
metadata:
  generateName: test-pod-cleanup-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnPodCompletion() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-completion-
spec:
  podGC:
    strategy: OnPodCompletion
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-completion-
spec:
  podGC:
    strategy: OnPodCompletion
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
}

func (s *PodCleanupSuite) TestOnPodCompletionLabelSelected() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-completion-label-selected-
spec:
  podGC:
    strategy: OnPodCompletion
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
      metadata:
        labels:
          evicted: true
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-completion-label-selected-
spec:
  podGC:
    strategy: OnPodCompletion
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodCompleted)
	})
}

func (s *PodCleanupSuite) TestOnPodSuccess() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-success-
spec:
  podGC:
    strategy: OnPodSuccess
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodCompleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-success-
spec:
  podGC:
    strategy: OnPodSuccess
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
}

func (s *PodCleanupSuite) TestOnPodSuccessLabelNotMatch() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-success-label-not-match-
spec:
  podGC:
    strategy: OnPodSuccess
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnPodSuccessLabelMatch() {
	s.Run("FailedPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-success-label-match-
spec:
  podGC:
    strategy: OnPodSuccess
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodCompleted)
	})
	s.Run("SucceededPod", func() {
		s.Given().
			Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-success-label-match-
spec:
  podGC:
    strategy: OnPodSuccess
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
      metadata:
        labels:
          evicted: true
`).
			When().
			SubmitWorkflow().
			WaitForPod(fixtures.PodDeleted)
	})
}

func (s *PodCleanupSuite) TestOnWorkflowCompletion() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-completion-
spec:
  podGC:
    strategy: OnWorkflowCompletion
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowCompletionLabelNotMatch() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-completion-label-not-match-
spec:
  podGC:
    strategy: OnWorkflowCompletion
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnWorkflowCompletionLabelMatch() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-completion-label-match-
spec:
  podGC:
    strategy: OnWorkflowCompletion
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
      metadata:
        labels:
          evicted: true
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowSuccess() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-success-
spec:
  podGC:
    strategy: OnWorkflowSuccess
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowSuccessLabelNotMatch() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-success-label-not-match-
spec:
  podGC:
    strategy: OnWorkflowSuccess
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted)
}

func (s *PodCleanupSuite) TestOnWorkflowSuccessLabelMatch() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-success-label-match-
spec:
  podGC:
    strategy: OnWorkflowSuccess
    labelSelector:
      matchLabels:
        evicted: true
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
      metadata:
        labels:
          evicted: true
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodDeleted)
}

func (s *PodCleanupSuite) TestOnWorkflowTemplate() {
	s.Given().
		WorkflowTemplate(`
metadata:
  name: test-pod-cleanup
spec:
  podGC:
    strategy: OnWorkflowCompletion
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflowsFromWorkflowTemplates().
		WaitForPod(fixtures.PodDeleted)
}

func TestPodCleanupSuite(t *testing.T) {
	suite.Run(t, new(PodCleanupSuite))
}
