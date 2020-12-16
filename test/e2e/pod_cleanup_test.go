// +build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/workflow/common"
)

type PodCleanupSuite struct {
	fixtures.E2ESuite
}

const enoughTimeForPodCleanup = 5 * time.Second

func (s *PodCleanupSuite) TestNone() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-
  labels:
    argo-e2e: true
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Wait(enoughTimeForPodCleanup).
		Then().
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) && assert.NotNil(t, p) {
				assert.Equal(t, "true", p.Labels[common.LabelKeyCompleted])
			}
		})
}

func (s *PodCleanupSuite) TestOnPodCompletion() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-success-
  labels:
    argo-e2e: true
spec:
  podGC: 
    strategy: OnPodCompletion
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
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Wait(enoughTimeForPodCleanup).
		Then().
		ExpectWorkflowNode(wfv1.FailedPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.Nil(t, p, "failed pod is deleted")
			}
		}).
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.Nil(t, p, "successful pod is deleted")
			}
		})
}

func (s *PodCleanupSuite) TestOnPodSuccess() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-pod-success-
  labels:
    argo-e2e: true
spec:
  podGC: 
    strategy: OnPodSuccess
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
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Wait(enoughTimeForPodCleanup).
		Then().
		ExpectWorkflowNode(wfv1.FailedPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.NotNil(t, p, "failed pod is NOT deleted")
			}
		}).
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.Nil(t, p, "successful pod is deleted")
			}
		})
}

func (s *PodCleanupSuite) TestOnWorkflowCompletion() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-completion-
  labels:
    argo-e2e: true
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
		WaitForWorkflow().
		Wait(enoughTimeForPodCleanup).
		Then().
		ExpectWorkflowNode(wfv1.FailedPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.Nil(t, p, "failed pod is deleted")
			}
		})
}

func (s *PodCleanupSuite) TestOnWorkflowSuccess() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-pod-cleanup-on-workflow-success-
  labels:
    argo-e2e: true
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
		WaitForWorkflow().
		Wait(enoughTimeForPodCleanup).
		Then().
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *corev1.Pod) {
			if assert.NotNil(t, n) {
				assert.Nil(t, p, "successful pod is deleted")
			}
		})
}

func TestPodCleanupSuite(t *testing.T) {
	suite.Run(t, new(PodCleanupSuite))
}
