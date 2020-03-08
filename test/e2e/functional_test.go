package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
  labels:
    argo-e2e: true
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
        template: whalesay

  - name: boom
    dag:
      tasks:
      - name: B-1
        template: whalesplosion

  - name: whalesay
    container:
      image: cowsay:v1
      imagePullPolicy: IfNotPresent

  - name: whalesplosion
    container:
      image: cowsay:v1
      imagePullPolicy: IfNotPresent
      command: ["sh", "-c", "sleep 5 ; exit 1"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
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
	// TODO: Test fails due to using a service account with insufficient permissions, skipping for now
	// pods is forbidden: User "system:serviceaccount:argo:default" cannot list resource "pods" in API group "" in the namespace "argo"
	s.T().SkipNow()
	s.Given().
		Workflow("@expectedfailures/pod-termination-failure.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(120 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			assert.Len(t, status.Nodes, 4)
			nodeStatus := status.Nodes.FindByDisplayName("sleep")
			assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
			assert.Equal(t, "pod termination", nodeStatus.Message)
		})
}

func (s *FunctionalSuite) TestEventOnNodeFail() {
	// Test whether an WorkflowFailed event (with appropriate message) is emitted in case of node failure
	s.Given().
		Workflow("@expectedfailures/failed-step-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
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
	// Test whether an WorkflowSuccess event is emitted in case of successfully completed workflow
	s.Given().
		Workflow("@functional/success-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(60 * time.Second).
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
	//  Test whether an WorkflowFailed event (with appropriate message) is emitted in case of error in creating the PVC
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

func (s *FunctionalSuite) TestLoopEmptyParam() {
	s.Given().
		Workflow("@functional/loops-empty-param.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.Len(t, status.Nodes, 5)
			nodeStatus := status.Nodes.FindByDisplayName("sleep")
			assert.Equal(t, wfv1.NodeSkipped, nodeStatus.Phase)
			assert.Equal(t, "Skipped, empty params", nodeStatus.Message)
		})
}

func (s *FunctionalSuite) TestParameterAggregation() {
	s.Given().
		Workflow("@functional/param-aggregation.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(60 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("print(0:1)")
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
		})
}

func (s *FunctionalSuite) TestStopBehavior() {
	s.Given().
		Workflow("@functional/stop-terminate.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflowToStart(5*time.Second).
		RunCli([]string{"stop", "stop-terminate"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Contains(t, output, "workflow stop-terminate stopped")
		}).
		WaitForWorkflow(20 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A.onExit")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			}
			nodeStatus = status.Nodes.FindByDisplayName("stop-terminate.onExit")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			}
		})
}

func (s *FunctionalSuite) TestTerminateBehavior() {
	s.Given().
		Workflow("@functional/stop-terminate.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflowToStart(5*time.Second).
		RunCli([]string{"terminate", "stop-terminate"}, func(t *testing.T, output string, err error) {
			assert.NoError(t, err)
			assert.Contains(t, output, "workflow stop-terminate terminated")
		}).
		WaitForWorkflow(20 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A.onExit")
			assert.Nil(t, nodeStatus)
			nodeStatus = status.Nodes.FindByDisplayName("stop-terminate.onExit")
			assert.Nil(t, nodeStatus)
		})
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
