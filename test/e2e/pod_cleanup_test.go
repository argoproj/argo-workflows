//go:build functional
// +build functional

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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

func (s *PodCleanupSuite) TestInvalidPodGCLabelSelector() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-invalid-pod-gc-label-selector-
spec:
  podGC:
    strategy: OnPodCompletion
    labelSelector:
      matchExpressions:
        - {key: environment, operator: InvalidOperator, values: [dev]}
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: success
            template: success
    - name: success
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.True(t, strings.Contains(status.Message, "failed to parse label selector"))
		})
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
		s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7159")
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
		s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7159")
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
		s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7159")
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
	s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7159")
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
      steps:
        - - name: success
            template: success
          - name: failure
            template: failure
    - name: success
      container:
        image: argoproj/argosay:v2
    - name: failure
      container:
        image: argoproj/argosay:v2
        args: [exit, 1]
      metadata:
        labels:
          evicted: true
`).
		When().
		SubmitWorkflow().
		WaitForPod(fixtures.PodCompleted).
		Then().
		ExpectWorkflowNode(wfv1.FailedPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.NotNil(t, p, "failed pod is not deleted since it did not succeed")
			}
		}).
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.NotNil(t, p, "successful pod is not deleted since it did not match the label selector in podGC")
			}
		})
}

func (s *PodCleanupSuite) TestOnPodSuccessLabelMatch() {
	s.Run("FailedPod", func() {
		s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7159")
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
	s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7159")
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
	s.T().Skip("https://github.com/argoproj/argo-workflows/issues/7159")
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

func TestPodCleanupSuite(t *testing.T) {
	suite.Run(t, new(PodCleanupSuite))
}
