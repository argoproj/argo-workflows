//go:build functional
// +build functional

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

func (s *FunctionalSuite) TestArchiveStrategies() {
	s.Given().
		Workflow(`@testdata/archive-strategies.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

// when you delete a pending pod,
// then the pod is re- created automatically
func (s *FunctionalSuite) TestDeletingPendingPod() {
	s.Given().
		Workflow("@testdata/pending-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart).
		Exec("kubectl", []string{"-n", "argo", "delete", "pod", "-l", "workflows.argoproj.io/workflow"}, fixtures.OutputRegexp(`pod "pending-.*" deleted`)).
		Wait(3*time.Second). // allow 3s for reconciliation, we'll create a new pod
		Exec("kubectl", []string{"-n", "argo", "get", "pod", "-l", "workflows.argoproj.io/workflow"}, fixtures.OutputRegexp(`pending-.*Pending`))
}

func (s *FunctionalSuite) TestWorkflowLevelErrorRetryPolicy() {
	s.Given().
		Workflow("@testdata/retry-on-error-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeTypeRetry, status.Nodes[metadata.Name].Type)
		})
}

func (s *FunctionalSuite) TestWorkflowMetadataLabelsFrom() {
	s.Given().
		Workflow(`
metadata:
  generateName: metadata-
spec:
  arguments:
    parameters:
      - name: foo
        value: bar
  workflowMetadata:
    labelsFrom:
      my-label: 
        expression: workflow.parameters.foo
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "bar", metadata.Labels["my-label"])
		})
}

func (s *FunctionalSuite) TestWorkflowTTL() {
	s.Given().
		Workflow(`
metadata:
  generateName: workflow-ttl-
spec:
  ttlStrategy:
    secondsAfterCompletion: 0
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Wait(3 * time.Second). // enough time for TTL controller to delete the workflow
		Then().
		ExpectWorkflowDeleted()
}

func (s *FunctionalSuite) TestWorkflowRetention() {
	listOptions := metav1.ListOptions{LabelSelector: "workflows.argoproj.io/phase=Failed"}
	s.Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Given().
		Workflow("@testdata/exit-1.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		WaitForWorkflowList(listOptions, func(list []wfv1.Workflow) bool {
			return len(list) == 2
		})
}

// in this test we create a poi quota, and then  we create a workflow that needs one more pod than the quota allows
// because we run them in parallel, the first node will run to completion, and then the second one
func (s *FunctionalSuite) TestResourceQuota() {
	s.Given().
		Workflow(`@testdata/two-items.yaml`).
		When().
		PodsQuota(2).
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *FunctionalSuite) TestContinueOnFail() {
	s.Given().
		Workflow(`
metadata:
  generateName: continue-on-fail-
spec:
  entrypoint: main
  parallelism: 2
  templates:
  - name: main
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

  - name: whalesplosion
    container:
      image: argoproj/argosay:v2
      args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
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
metadata:
  generateName: continue-on-failed-dag-
spec:
  entrypoint: workflow-ignore
  templates:
    - name: workflow-ignore
      dag:
        failFast: false
        tasks:
          - name: F
            template: fail
            continueOn:
              failed: true
          - name: P
            template: pass
            dependencies:
              - F

    - name: pass
      container:
        image: argoproj/argosay:v2

    - name: fail
      container:
        image: argoproj/argosay:v2
        args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeFailed, status.Nodes.FindByDisplayName("F").Phase)
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("P").Phase)
		})
}

func (s *FunctionalSuite) TestVolumeClaimTemplate() {
	s.Given().
		Workflow(`@testdata/volume-claim-template-workflow.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		// test that the PVC was deleted (because the `kubernetes.io/pvc-protection` finalizer was deleted)
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					t.Error("timeout waiting for PVC to be deleted")
					t.FailNow()
				case <-ticker.C:
					list, err := s.KubeClient.CoreV1().PersistentVolumeClaims(fixtures.Namespace).List(context.Background(), metav1.ListOptions{})
					if assert.NoError(t, err) {
						if len(list.Items) == 0 {
							return
						}
					}
				}
			}
		})
}

func (s *FunctionalSuite) TestEventOnNodeFail() {
	// Test whether an WorkflowFailed event (with appropriate message) is emitted in case of node failure
	var uid types.UID
	s.Given().
		Workflow("@expectedfailures/failed-step-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		}).
		ExpectAuditEvents(
			fixtures.HasInvolvedObject(workflow.WorkflowKind, uid),
			4,
			func(t *testing.T, es []apiv1.Event) {
				for _, e := range es {
					switch e.Reason {
					case "WorkflowNodeRunning":
						assert.Contains(t, e.Message, "Running node failed-step-event-")
					case "WorkflowRunning":
					case "WorkflowNodeFailed":
						assert.Contains(t, e.Message, "Failed node failed-step-event-")
						assert.Equal(t, e.Annotations["workflows.argoproj.io/node-type"], "Pod")
						assert.Contains(t, e.Annotations["workflows.argoproj.io/node-name"], "failed-step-event-")
					case "WorkflowFailed":
						assert.Contains(t, e.Message, "exit code 1")
					default:
						assert.Fail(t, e.Reason)
					}
				}
			},
		)
}

func (s *FunctionalSuite) TestEventOnWorkflowSuccess() {
	// Test whether an WorkflowSuccess event is emitted in case of successfully completed workflow
	var uid types.UID
	s.Given().
		Workflow("@functional/success-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(60*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		}).
		ExpectAuditEvents(
			fixtures.HasInvolvedObject(workflow.WorkflowKind, uid),
			4,
			func(t *testing.T, es []apiv1.Event) {
				for _, e := range es {
					println(e.Reason, e.Message)
					switch e.Reason {
					case "WorkflowNodeRunning":
						assert.Contains(t, e.Message, "Running node success-event-")
					case "WorkflowRunning":
					case "WorkflowNodeSucceeded":
						assert.Contains(t, e.Message, "Succeeded node success-event-")
						assert.Equal(t, "Pod", e.Annotations["workflows.argoproj.io/node-type"])
						assert.Contains(t, e.Annotations["workflows.argoproj.io/node-name"], "success-event-")
					case "WorkflowSucceeded":
						assert.Equal(t, "Workflow completed", e.Message)
					default:
						assert.Fail(t, e.Reason)
					}
				}
			},
		)
}

func (s *FunctionalSuite) TestLargeWorkflowFailure() {
	var uid types.UID
	s.Given().
		Workflow("@expectedfailures/large-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(120*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		}).
		ExpectAuditEvents(
			fixtures.HasInvolvedObject(workflow.WorkflowKind, uid),
			2,
			func(t *testing.T, e []apiv1.Event) {
				assert.Equal(t, "WorkflowRunning", e[0].Reason)

				assert.Equal(t, "WorkflowFailed", e[1].Reason)
				assert.Contains(t, e[1].Message, "workflow templates are limited to 128KB, this workflow is 128001 bytes")
			},
		)
}

func (s *FunctionalSuite) TestEventOnPVCFail() {
	//  Test whether an WorkflowFailed event (with appropriate message) is emitted in case of error in creating the PVC
	var uid types.UID
	s.Given().
		Workflow("@expectedfailures/volumes-pvc-fail-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(120*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		}).
		ExpectAuditEvents(
			fixtures.HasInvolvedObject(workflow.WorkflowKind, uid),
			2,
			func(t *testing.T, e []apiv1.Event) {
				assert.Equal(t, "WorkflowRunning", e[0].Reason)

				assert.Equal(t, "WorkflowFailed", e[1].Reason)
				assert.Contains(t, e[1].Message, "pvc create error")
			},
		)
}

func (s *FunctionalSuite) TestArtifactRepositoryRef() {
	s.Given().
		Workflow("@testdata/artifact-repository-ref.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			if assert.NotEmpty(t, status.ArtifactRepositoryRef) {
				assert.Equal(t, "argo", status.ArtifactRepositoryRef.Namespace)
				assert.Equal(t, "artifact-repositories", status.ArtifactRepositoryRef.ConfigMap)
				assert.Equal(t, "my-key", status.ArtifactRepositoryRef.Key)
				assert.False(t, status.ArtifactRepositoryRef.Default)
			}
			// these should never be set because we must get them from the artifactRepositoryRef
			generated := status.Nodes.FindByDisplayName("generate").Outputs.Artifacts[0].S3
			assert.Empty(t, generated.Bucket)
			assert.NotEmpty(t, generated.Key)
			consumed := status.Nodes.FindByDisplayName("consume").Inputs.Artifacts[0].S3
			assert.Empty(t, consumed.Bucket)
			assert.NotEmpty(t, consumed.Key)
		})
}

func (s *FunctionalSuite) TestLoopEmptyParam() {
	s.Given().
		Workflow("@functional/loops-empty-param.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			if assert.Len(t, status.Nodes, 5) {
				nodeStatus := status.Nodes.FindByDisplayName("sleep")
				assert.Equal(t, wfv1.NodeSkipped, nodeStatus.Phase)
				assert.Equal(t, "Skipped, empty params", nodeStatus.Message)
			}
		})
}

func (s *FunctionalSuite) TestDAGEmptyParam() {
	s.Given().
		Workflow("@functional/dag-empty-param.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			if assert.Len(t, status.Nodes, 3) {
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
  generateName: pending-retry-workflow-
spec:
  entrypoint: dag
  templates:
  - name: cowsay
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
		WaitForWorkflow(fixtures.ToStart).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			a := wf.Status.Nodes.FindByDisplayName("a")
			b := wf.Status.Nodes.FindByDisplayName("b")
			return wfv1.NodePending == a.Phase && wfv1.NodePending == b.Phase, "pods pending"
		})).
		DeleteMemoryQuota().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			a := wf.Status.Nodes.FindByDisplayName("a")
			b := wf.Status.Nodes.FindByDisplayName("b")
			return wfv1.NodeSucceeded == a.Phase && wfv1.NodeSucceeded == b.Phase, "pods succeeded"
		}))
}

// 128M is for argo executor
func (s *FunctionalSuite) TestPendingRetryWorkflowWithRetryStrategy() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: pending-retry-workflow-with-retry-strategy-
spec:
  entrypoint: dag
  templates:
  - name: cowsay
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
		WaitForWorkflow(fixtures.ToStart).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			a := wf.Status.Nodes.FindByDisplayName("a(0)")
			b := wf.Status.Nodes.FindByDisplayName("b(0)")
			return wfv1.NodePending == a.Phase && wfv1.NodePending == b.Phase, "pods pending"
		})).
		DeleteMemoryQuota().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			a := wf.Status.Nodes.FindByDisplayName("a(0)")
			b := wf.Status.Nodes.FindByDisplayName("b(0)")
			return wfv1.NodeSucceeded == a.Phase && wfv1.NodeSucceeded == b.Phase, "pods succeeded"
		}))
}

func (s *FunctionalSuite) TestParameterAggregation() {
	s.Given().
		Workflow("@functional/param-aggregation.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("print(0:res:1)")
			if assert.NotNil(t, nodeStatus) {
				assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			}
		})
}

func (s *FunctionalSuite) TestDAGDepends() {
	s.Given().
		Workflow("@functional/dag-depends.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("should-execute-1").Phase)
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("should-execute-2").Phase)
			assert.Equal(t, wfv1.NodeOmitted, status.Nodes.FindByDisplayName("should-not-execute").Phase)
		})
}

func (s *FunctionalSuite) TestOptionalInputArtifacts() {
	s.Given().
		Workflow("@testdata/input-artifacts.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *FunctionalSuite) TestWorkflowTemplateRefWithExitHandler() {
	s.Given().
		WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
		Workflow("@testdata/workflow-template-ref-exithandler.yaml").
		When().
		CreateWorkflowTemplates().
		Wait(1 * time.Second). // allow the template to reach the informer
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			assert.Empty(t, status.Message)
		})
}

func (s *FunctionalSuite) TestWorkflowTemplateRefWithExitHandlerError() {
	s.Given().
		WorkflowTemplate(`
metadata:
  name: test-exit-handler
spec:
  entrypoint: main
  onExit: exit-handler
  templates:
    - name: main
      container:
        name: main
        image: argoproj/argosay:v2
    - name: exit-handler
      templateRef:
        name: nonexistent
        template: exit-handler
`).
		Workflow(`
metadata:
  generateName: test-exit-handler-
spec:
  workflowTemplateRef:
    name: test-exit-handler
`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Contains(t, status.Message, "error in exit template execution")
		})
}

func (s *FunctionalSuite) TestWorkflowLifecycleHookWithWorkflowTemplate() {
	s.Given().
		WorkflowTemplate(`
metadata:
  name: test-exit-handler
spec:
  entrypoint: main
  templates:
    - name: main
      inputs:
        parameters:
        - name: message
      container:
        image: argoproj/argosay:v2 
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
`).
		Workflow(`
metadata:
  generateName: test-lifecycle-hook-
spec:
  entrypoint: hooks-exit-test
  templates:
  - name: hooks-exit-test
    container:
      image: argoproj/argosay:v2
    hooks:
      exit:
        templateRef:
          name: test-exit-handler
          template: main
        arguments:
          parameters:
            - name: message
              value: "hello world"
`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			assert.Empty(t, status.Message)
		})
}

func (s *FunctionalSuite) TestParametrizableAds() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: param-ads-
spec:
  entrypoint: whalesay
  arguments:
    parameters:
      - name: ads
        value: "5"
  templates:
  - name: whalesay
    inputs:
      parameters:
        - name: ads
    activeDeadlineSeconds: "{{inputs.parameters.ads}}"
    container:
      image: argoproj/argosay:v2
      args: [sleep, 10s]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			if node := status.Nodes.FindByDisplayName(md.Name); assert.NotNil(t, node) {
				assert.Contains(t, node.Message, "Pod was active on the node longer than the specified deadline")
			}
		})
}

func (s *FunctionalSuite) TestParametrizableLimit() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: param-limit-
spec:
  entrypoint: whalesay
  arguments:
    parameters:
      - name: limit
        value: "1"
  templates:
  - name: whalesay
    inputs:
      parameters:
        - name: limit
    retryStrategy:
      limit: "{{inputs.parameters.limit}}"
    container:
      image: argoproj/argosay:v2
      args: [exit, 1]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			node := status.Nodes[md.Name]
			assert.Contains(t, node.Message, "No more retries left")
			assert.Len(t, status.Nodes, 3)
		})
}

func (s *FunctionalSuite) TestTemplateLevelTimeout() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-tmpl-timeout-
spec:
  entrypoint: hello-hello-hello
  templates:
  - name: hello-hello-hello
    steps:
    - - name: hello1
        template: whalesay
        arguments:
          parameters: [{name: message, value: "5s"}]
      - name: hello2a
        template: whalesay
        arguments:
          parameters: [{name: message, value: "10s"}]
      - name: hello2b
        template: whalesay
        arguments:
          parameters: [{name: message, value: "15s"}]

  - name: whalesay
    timeout: "{{inputs.parameters.message}}"
    inputs:
      parameters:
      - name: message
    container:
      image: argoproj/argosay:v2
      args: [sleep, 30s]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.Phase == wfv1.WorkflowFailed, "Waiting for timeout"
		}), 30*time.Second)
}

func (s *FunctionalSuite) TestTemplateLevelTimeoutWithForbidden() {
	s.Given().
		Workflow(`
metadata:
  generateName: steps-tmpl-timeout-
spec:
  entrypoint: hello-hello-hello
  templates:
  - name: hello-hello-hello
    steps:
    - - name: hello1
        template: whalesay
        arguments:
          parameters: [{name: message, value: "5s"}]
      - name: hello2a
        template: whalesay
        arguments:
          parameters: [{name: message, value: "10s"}]
      - name: hello2b
        template: whalesay
        arguments:
          parameters: [{name: message, value: "15s"}]

  - name: whalesay
    resources:
      limits:
        memory: 145M
    timeout: "{{inputs.parameters.message}}"
    inputs:
      parameters:
      - name: message
    container:
      image: argoproj/argosay:v2
      args: [sleep, 30s]
`).
		When().
		MemoryQuota("130M").
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed)
}

func (s *FunctionalSuite) TestWorkflowPodSpecPatch() {
	s.Given().
		Workflow(`
metadata:
  generateName: basic-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
      # ordering of the containers in the next line is intentionally reversed
      podSpecPatch: '{"terminationGracePeriodSeconds":5, "containers":[{"name":"main", "resources":{"limits":{"cpu": "100m"}}}, {"name":"wait", "resources":{"limits":{"cpu": "101m"}}}]}'
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.Equal(t, *p.Spec.TerminationGracePeriodSeconds, int64(5))
			for _, c := range p.Spec.Containers {
				if c.Name == "main" {
					assert.Equal(t, c.Resources.Limits.Cpu().String(), "100m")
				} else if c.Name == "wait" {
					assert.Equal(t, c.Resources.Limits.Cpu().String(), "101m")
				}
			}
		})
}

func (s *FunctionalSuite) TestOutputArtifactS3BucketCreationEnabled() {
	s.Given().
		Workflow("@testdata/output-artifact-with-s3-bucket-creation-enabled.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestDataTransformation() {
	s.Given().
		Workflow("@testdata/data-transformation.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(1 * time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			paths := status.Nodes.FindByDisplayName("get-artifact-path")
			if assert.NotNil(t, paths) {
				assert.Equal(t, `["foo/script.py","script.py"]`, *paths.Outputs.Result)
			}
			assert.NotNil(t, status.Nodes.FindByDisplayName("process-artifact(0:foo/script.py)"))
			assert.NotNil(t, status.Nodes.FindByDisplayName("process-artifact(1:script.py)"))
		})
}

func (s *FunctionalSuite) TestHTTPOutputs() {
	s.Given().
		Workflow("@testdata/http-outputs.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			httpNode := status.Nodes.FindByDisplayName("http")
			assert.NotNil(t, httpNode.Outputs.Result)
			echoNode := status.Nodes.FindByDisplayName("echo")
			assert.Equal(t, *httpNode.Outputs.Result, echoNode.Inputs.Parameters[0].Value.String())
		})
}

func (s *FunctionalSuite) TestScriptAsNonRoot() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: script-nonroot-
spec:
  entrypoint: whalesay
  securityContext:
    runAsUser: 1000
    runAsGroup: 1000
    runAsNonRoot: true
  templates:
    - name: whalesay
      script:
        image: argoproj/argosay:v2
        command: ["bash"]
        source: |
          ls -l /argo/staging
          cat /argo/stahing/script
          sleep 10s
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestPauseBefore() {
	s.Given().
		Workflow(`@functional/pause-before.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Exec("bash", []string{"-c", "sleep 5 &&  kubectl exec -i $(kubectl get pods | awk '/pause-before/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/before'"}, fixtures.NoError).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestPauseAfter() {
	s.Given().
		Workflow(`@functional/pause-after.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Exec("bash", []string{"-c", "sleep 5 && kubectl exec -i $(kubectl get pods -n argo | awk '/pause-after/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/after'"}, fixtures.NoError).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestPauseAfterAndBefore() {
	s.Given().
		Workflow(`@functional/pause-before-after.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		Exec("bash", []string{"-c", "sleep 5 && kubectl exec -i $(kubectl get pods | awk '/pause-before-after/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/before'"}, fixtures.NoError).
		Exec("bash", []string{"-c", "kubectl exec -i $(kubectl get pods | awk '/pause-before-after/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/after'"}, fixtures.NoError).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}

func (s *FunctionalSuite) TestStepLevelMemozie() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-memozie-
spec:
  entrypoint: hello-hello-hello
  templates:
    - name: hello-hello-hello
      steps:
        - - name: hello1
            template: memostep
            arguments:
              parameters: [{name: message, value: "hello1"}]
        - - name: hello2a
            template: memostep
            arguments:
              parameters: [{name: message, value: "hello1"}]
    - name: memostep
      inputs:
        parameters:
        - name: message
      memoize:
        key: "{{inputs.parameters.message}}"
        maxAge: "10s"
        cache:
          configMap:
            name: my-config-memo-step
      steps:
      - - name: cache
          template: whalesay
          arguments:
            parameters: [{name: message, value: "{{inputs.parameters.message}}"}]
      outputs:
        parameters:
        - name: output
          valueFrom:
            Parameter: "{{steps.cache.outputs.result}}"
    - name: whalesay
      inputs:
        parameters:
        - name: message
      container:
        image: argoproj/argosay:v2
        command: [echo]
        args: ["{{inputs.parameters.message}}"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			memoHit := false
			for _, node := range status.Nodes {
				if node.MemoizationStatus != nil && node.MemoizationStatus.Hit {
					memoHit = true
				}
			}
			assert.True(t, memoHit)

		})

}

func (s *FunctionalSuite) TestDAGLevelMemozie() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-memozie-
spec:
  entrypoint: hello-hello-hello
  templates:
    - name: hello-hello-hello
      steps:
        - - name: hello1
            template: memostep
            arguments:
              parameters: [{name: message, value: "hello1"}]
        - - name: hello2a
            template: memostep
            arguments:
              parameters: [{name: message, value: "hello1"}]
    - name: memostep
      inputs:
        parameters:
        - name: message
      memoize:
        key: "{{inputs.parameters.message}}"
        maxAge: "10s"
        cache:
          configMap:
            name: my-config-memo-dag
      dag:
        tasks:
        - name: cache
          template: whalesay
          arguments:
            parameters: [{name: message, value: "{{inputs.parameters.message}}"}]
      outputs:
        parameters:
        - name: output
          valueFrom:
            Parameter: "{{tasks.cache.outputs.result}}"
    - name: whalesay
      inputs:
        parameters:
        - name: message
      container:
        image: argoproj/argosay:v2
        command: [echo]
        args: ["{{inputs.parameters.message}}"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			memoHit := false
			for _, node := range status.Nodes {
				if node.MemoizationStatus != nil && node.MemoizationStatus.Hit {
					memoHit = true
				}
			}
			assert.True(t, memoHit)

		})

}

func (s *FunctionalSuite) TestContainerSetRetryFail() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: containerset-retry-success-
spec:
  entrypoint: main
  templates:
    - name: main
      containerSet:
        containers:
          - name: a
            image: argoproj/argosay:v2
            command: [sh, -c]
            args: ['FILE=test.yml; EXITCODE=1; if test -f "$FILE"; then EXITCODE=0; else touch $FILE; fi; exit $EXITCODE']
        retryStrategy:
          retries: 2
          duration: "5s"
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestContainerSetRetrySuccess() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: containerset-no-retry-fail-
spec:
  entrypoint: main
  templates:
    - name: main
      containerSet:
        containers:
          - name: a
            image: argoproj/argosay:v2
            command: [sh, -c]
            args: ['FILE=test.yml; EXITCODE=1; if test -f "$FILE"; then EXITCODE=0; else touch $FILE; fi; exit $EXITCODE']
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed)
}
