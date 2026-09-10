//go:build functional

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
)

type WorkflowActionSuite struct {
	fixtures.E2ESuite
}

const actionSleepWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: action-sleep-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
        args: [sleep, 300s]
`

const actionSuspendNodeWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: action-suspend-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: approve
            template: approve
    - name: approve
      suspend: {}
`

func actionSucceeded(a *wfv1.WorkflowAction) bool {
	return a.Status.Phase == wfv1.WorkflowActionSucceeded
}

func (s *WorkflowActionSuite) TestSuspendResumeViaAction() {
	s.Given().
		Workflow(actionSuspendNodeWf).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		CreateWorkflowAction(wfv1.WorkflowActionSpec{Action: wfv1.ActionTypeSuspend}).
		WaitForWorkflowAction(30*time.Second, actionSucceeded).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.Suspended, "workflow suspended"
		})).
		CreateWorkflowAction(wfv1.WorkflowActionSpec{Action: wfv1.ActionTypeResume}).
		WaitForWorkflowAction(30*time.Second, actionSucceeded).
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowActionSuite) TestTerminateViaAction() {
	s.Given().
		Workflow(actionSleepWf).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		CreateWorkflowAction(wfv1.WorkflowActionSpec{Action: wfv1.ActionTypeTerminate}).
		WaitForWorkflowAction(30*time.Second, actionSucceeded).
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.ShutdownStrategyTerminate, status.Shutdown)
		})
}

func (s *WorkflowActionSuite) TestStopViaAction() {
	s.Given().
		Workflow(actionSleepWf).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		CreateWorkflowAction(wfv1.WorkflowActionSpec{Action: wfv1.ActionTypeStop}).
		WaitForWorkflowAction(30*time.Second, actionSucceeded).
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.ShutdownStrategyStop, status.Shutdown)
		})
}

func (s *WorkflowActionSuite) TestActionOnMissingWorkflow() {
	s.Given().
		When().
		CreateWorkflowAction(wfv1.WorkflowActionSpec{
			WorkflowRef: wfv1.WorkflowActionRef{Name: "does-not-exist"},
			Action:      wfv1.ActionTypeTerminate,
		}).
		WaitForWorkflowAction(30*time.Second, func(a *wfv1.WorkflowAction) bool {
			return a.Status.Phase == wfv1.WorkflowActionFailed &&
				a.Status.Reason == wfv1.WorkflowActionReasonWorkflowNotFound
		})
}

func (s *WorkflowActionSuite) TestActionOnCompletedWorkflow() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: action-quick-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		CreateWorkflowAction(wfv1.WorkflowActionSpec{Action: wfv1.ActionTypeTerminate}).
		WaitForWorkflowAction(30*time.Second, func(a *wfv1.WorkflowAction) bool {
			return a.Status.Phase == wfv1.WorkflowActionFailed &&
				a.Status.Reason == wfv1.WorkflowActionReasonWorkflowCompleted
		})
}

func TestWorkflowActionSuite(t *testing.T) {
	suite.Run(t, new(WorkflowActionSuite))
}
