package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/util/argo"

	apiv1 "k8s.io/api/core/v1"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

func (s *FunctionalSuite) TestContinueOnFail() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: continue-on-fail
spec:
  entrypoint: workflow-ignore
  parallelism: 2
  templates:
  - name: workflow-ignore
    steps:
    - - name: A
        template: whalesay
      - name: B
        template: boom
        continueOn:
          failed: true
    - - name: C
        dependencies: [A, B]
        template: whalesay

  - name: boom
    dag:
      tasks:
      - name: B-1
        template: whalesplosion

  - name: whalesay
    container:
      image: docker/whalesay:latest

  - name: whalesplosion
    container:
      image: docker/whalesay:latest
      command: ["sh", "-c", "sleep 5 ; exit 1"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		Expect(func(t *testing.T, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.Len(t, status.Nodes, 7)
			nodeStatus := status.Nodes.FindByDisplayName("B")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
				assert.Len(t, nodeStatus.Children, 1)
				assert.Len(t, nodeStatus.OutboundNodes, 1)
			}
		})
}

func (s *FunctionalSuite) TestFastFailOnPodTermination() {
	s.Given().
		Workflow("@expectedfailures/pod-termination-failure.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(120 * time.Second).
		Then().
		Expect(func(t *testing.T, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			assert.Len(t, status.Nodes, 4)
			nodeStatus := status.Nodes.FindByDisplayName("sleep")
			assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
			assert.Equal(t, "pod termination", nodeStatus.Message)
		})
}

func (s *FunctionalSuite) TestEventOnNodeFail() {
	s.Given().
		Workflow("@expectedfailures/failed-step-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(120 * time.Second).
		Then().
		ExpectAuditEvents(func(t *testing.T, events *apiv1.EventList) {
			found := false
			for _, e := range events.Items {
				isAboutFailedStep := strings.HasPrefix(e.InvolvedObject.Name, "failed-step-event-")
				isFailureEvent := e.Reason == argo.EventReasonWorkflowFailed
				if isAboutFailedStep && isFailureEvent {
					found = true
					assert.Equal(t, "failed with exit code 1", e.Message)
				}
			}
			assert.True(t, found, "event not found")
		})
}

func (s *FunctionalSuite) TestEventOnWorkflowSuccess() {
	s.Given().
		Workflow("@functional/success-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(120 * time.Second).
		Then().
		ExpectAuditEvents(func(t *testing.T, events *apiv1.EventList) {
			found := false
			for _, e := range events.Items {
				isAboutSuccess := strings.HasPrefix(e.InvolvedObject.Name, "success-event-")
				isSuccessEvent := e.Reason == argo.EventReasonWorkflowSucceded
				if isAboutSuccess && isSuccessEvent {
					found = true
					assert.Equal(t, "Workflow completed", e.Message)
				}
			}
			assert.True(t, found, "event not found")
		})
}

func (s *FunctionalSuite) TestEventOnPVCFail() {
	s.Given().
		Workflow("@expectedfailures/volumes-pvc-fail-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(120 * time.Second).
		Then().
		ExpectAuditEvents(func(t *testing.T, events *apiv1.EventList) {
			found := false
			for _, e := range events.Items {
				isAboutSuccess := strings.HasPrefix(e.InvolvedObject.Name, "volumes-pvc-fail-event-")
				isFailureEvent := e.Reason == argo.EventReasonWorkflowFailed
				if isAboutSuccess && isFailureEvent {
					found = true
					assert.True(t, strings.Contains(e.Message, "pvc create error"), "event should contain \"pvc create error\"")
				}
			}
			assert.True(t, found, "event not found")
		})
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
