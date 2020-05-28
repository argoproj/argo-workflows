package e2e

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/util/argo"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

func (s *FunctionalSuite) TestArchiveStrategies() {
	s.Given().
		Workflow(`@testdata/archive-strategies.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
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
      image: argoproj/argosay:v2
      imagePullPolicy: IfNotPresent

  - name: whalesplosion
    container:
      image: argoproj/argosay:v2
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

func (s *FunctionalSuite) TestContinueOnFailDag() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: continue-on-failed-dag
  labels:
    argo-e2e: true
spec:
  entrypoint: workflow-ignore
  parallelism: 2
  templates:
    - name: workflow-ignore
      dag:
        failFast: false
        tasks:
          - name: A
            template: whalesay
          - name: B
            template: boom
            continueOn:
              failed: true
            dependencies:
              - A
          - name: C
            template: whalesay
            dependencies:
              - A
          - name: D
            template: whalesay
            dependencies:
              - B
              - C

    - name: boom
      dag:
        tasks:
          - name: B-1
            template: whalesplosion

    - name: whalesay
      container:
        imagePullPolicy: IfNotPresent
        image: argoproj/argosay:v2

    - name: whalesplosion
      container:
        imagePullPolicy: IfNotPresent
        image: argoproj/argosay:v2
        command: ["sh", "-c", "sleep 10; exit 1"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.Len(t, status.Nodes, 6)

			bStatus := status.Nodes.FindByDisplayName("B")
			if assert.NotNil(t, bStatus) {
				assert.Equal(t, wfv1.NodeFailed, bStatus.Phase)
			}

			dStatus := status.Nodes.FindByDisplayName("D")
			if assert.NotNil(t, dStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, dStatus.Phase)
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
			assert.Equal(t, wfv1.NodeError, status.Phase)
			assert.Len(t, status.Nodes, 4)
			nodeStatus := status.Nodes.FindByDisplayName("sleep")
			assert.Equal(t, wfv1.NodeError, nodeStatus.Phase)
			assert.Equal(t, "pod deleted during operation", nodeStatus.Message)
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
		ExpectAuditEvent(func(e corev1.Event) bool {
			return strings.HasPrefix(e.InvolvedObject.Name, "failed-step-event-") &&
				e.Reason == argo.EventReasonWorkflowFailed &&
				e.Message == "failed with exit code 1"
		}).
		ExpectAuditEvent(func(e corev1.Event) bool {
			return e.InvolvedObject.Kind == workflow.WorkflowKind &&
				e.Reason == argo.EventReasonWorkflowNodeFailed &&
				strings.HasPrefix(e.Message, "Failed node failed-step-event-") &&
				e.Annotations["workflows.argoproj.io/node-type"] == "Pod" &&
				strings.Contains(e.Annotations["workflows.argoproj.io/node-name"], "failed-step-event-")
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
		ExpectAuditEvent(func(e corev1.Event) bool {
			return strings.HasPrefix(e.InvolvedObject.Name, "success-event-") &&
				e.Reason == argo.EventReasonWorkflowSucceeded &&
				e.Message == "Workflow completed"
		}).
		ExpectAuditEvent(func(e corev1.Event) bool {
			return e.InvolvedObject.Kind == workflow.WorkflowKind &&
				e.Reason == argo.EventReasonWorkflowNodeSucceeded &&
				strings.HasPrefix(e.Message, "Succeeded node success-event-") &&
				e.Annotations["workflows.argoproj.io/node-type"] == "Pod" &&
				strings.Contains(e.Annotations["workflows.argoproj.io/node-name"], "success-event-")
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
		ExpectAuditEvent(func(e corev1.Event) bool {
			return strings.HasPrefix(e.InvolvedObject.Name, "volumes-pvc-fail-event-") &&
				e.Reason == argo.EventReasonWorkflowFailed &&
				strings.Contains(e.Message, "pvc create error")
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
			if assert.Len(t, status.Nodes, 5) {
				nodeStatus := status.Nodes.FindByDisplayName("sleep")
				assert.Equal(t, wfv1.NodeSkipped, nodeStatus.Phase)
				assert.Equal(t, "Skipped, empty params", nodeStatus.Message)
			}
		})
}

// 128M is for argo executor
func (s *FunctionalSuite) TestPendingRetryWorkflow() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-limited-1
  labels:
    argo-e2e: true
spec:
  entrypoint: dag
  templates:
  - name: cowsay
    resubmitPendingPods: true
    container:
      image: argoproj/argosay:v2
      args: ["echo", "a"]
      resources:
        limits:
          memory: 128M
  - name: dag
    dag:
      tasks:
      - name: a
        template: cowsay
      - name: b
        template: cowsay
`).
		When().
		MemoryQuota("130M").
		SubmitWorkflow().
		WaitForWorkflowToStart(5*time.Second).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			a := wf.Status.Nodes.FindByDisplayName("a")
			b := wf.Status.Nodes.FindByDisplayName("b")
			return wfv1.NodePending == a.Phase &&
				regexp.MustCompile(`^Pending \d+\.\d+s$`).MatchString(a.Message) &&
				wfv1.NodePending == b.Phase &&
				regexp.MustCompile(`^Pending \d+\.\d+s$`).MatchString(b.Message)
		}, "pods pending", 30*time.Second).
		DeleteQuota().
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			a := wf.Status.Nodes.FindByDisplayName("a")
			b := wf.Status.Nodes.FindByDisplayName("b")
			return wfv1.NodeSucceeded == a.Phase && wfv1.NodeSucceeded == b.Phase
		}, "pods succeeded", 30*time.Second)
}

// 128M is for argo executor
func (s *FunctionalSuite) TestPendingRetryWorkflowWithRetryStrategy() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-limited-2
  labels:
    argo-e2e: true
spec:
  entrypoint: dag
  templates:
  - name: cowsay
    resubmitPendingPods: true
    retryStrategy:
      limit: 1
    container:
      image: argoproj/argosay:v2
      args: ["echo", "a"]
      resources:
        limits:
          memory: 128M
  - name: dag
    dag:
      tasks:
      - name: a
        template: cowsay
      - name: b
        template: cowsay
`).
		When().
		MemoryQuota("130M").
		SubmitWorkflow().
		WaitForWorkflowToStart(5*time.Second).
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			a := wf.Status.Nodes.FindByDisplayName("a(0)")
			b := wf.Status.Nodes.FindByDisplayName("b(0)")
			return wfv1.NodePending == a.Phase &&
				regexp.MustCompile(`^Pending \d+\.\d+s$`).MatchString(a.Message) &&
				wfv1.NodePending == b.Phase &&
				regexp.MustCompile(`^Pending \d+\.\d+s$`).MatchString(b.Message)
		}, "pods pending", 30*time.Second).
		DeleteQuota().
		WaitForWorkflowCondition(func(wf *wfv1.Workflow) bool {
			a := wf.Status.Nodes.FindByDisplayName("a(0)")
			b := wf.Status.Nodes.FindByDisplayName("b(0)")
			return wfv1.NodeSucceeded == a.Phase && wfv1.NodeSucceeded == b.Phase
		}, "pods succeeded", 30*time.Second)
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
			nodeStatus := status.Nodes.FindByDisplayName("print(0:res:1)")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			}
		})
}

func (s *FunctionalSuite) TestGlobalScope() {
	s.Given().
		Workflow("@functional/global-scope.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(60 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("consume-global-parameter-1")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
				assert.Equal(t, "initial", *nodeStatus.Outputs.Result)
			}
			nodeStatus = status.Nodes.FindByDisplayName("consume-global-parameter-2")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
				assert.Equal(t, "initial", *nodeStatus.Outputs.Result)
			}
			nodeStatus = status.Nodes.FindByDisplayName("consume-global-parameter-3")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
				assert.Equal(t, "final", *nodeStatus.Outputs.Result)
			}
			nodeStatus = status.Nodes.FindByDisplayName("consume-global-parameter-4")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
				assert.Equal(t, "final", *nodeStatus.Outputs.Result)
			}
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
		WaitForWorkflow(45 * time.Second).
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
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A.onExit")
			assert.Nil(t, nodeStatus)
			nodeStatus = status.Nodes.FindByDisplayName("stop-terminate.onExit")
			assert.Nil(t, nodeStatus)
		})
}

func (s *FunctionalSuite) TestDAGDepends() {
	s.Given().
		Workflow("@functional/dag-depends.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("A")
			assert.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			nodeStatus = status.Nodes.FindByDisplayName("B")
			assert.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			nodeStatus = status.Nodes.FindByDisplayName("C")
			assert.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
			nodeStatus = status.Nodes.FindByDisplayName("should-execute-1")
			assert.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			nodeStatus = status.Nodes.FindByDisplayName("should-execute-2")
			assert.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			nodeStatus = status.Nodes.FindByDisplayName("should-not-execute")
			assert.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSkipped, nodeStatus.Phase)
			nodeStatus = status.Nodes.FindByDisplayName("should-execute-3")
			assert.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
		})
}

func (s *FunctionalSuite) TestDefaultParameterOutputs() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: default-params
  labels:
    argo-e2e: true
spec:
  entrypoint: start
  templates:
  - name: start
    steps:
      - - name: generate-1
          template: generate
      - - name: generate-2
          when: "True == False"
          template: generate
    outputs:
      parameters:
        - name: nested-out-parameter
          valueFrom:
            default: "Default value"
            parameter: "{{steps.generate-2.outputs.parameters.out-parameter}}"

  - name: generate
    container:
      image: argoproj/argosay:v2
      args: [echo, my-output-parameter, /tmp/my-output-parameter.txt]
    outputs:
      parameters:
      - name: out-parameter
        valueFrom:
          path: /tmp/my-output-parameter.txt
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
			assert.True(t, status.Nodes.Any(func(node wfv1.NodeStatus) bool {
				if node.Outputs != nil {
					for _, param := range node.Outputs.Parameters {
						if param.Value != nil && *param.Value == "Default value" {
							return true
						}
					}
				}
				return false
			}))
		})
}

func (s *FunctionalSuite) TestSameInputOutputPathOptionalArtifact() {
	s.Given().
		Workflow("@testdata/same-input-output-path-optional.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func (s *FunctionalSuite) TestOptionalInputArtifacts() {
	s.Given().
		Workflow("@testdata/input-artifacts.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(30 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Phase)
		})
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
