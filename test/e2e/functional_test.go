//go:build corefunctional

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		WaitForWorkflow(time.Second * 90).
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
		// patch the pod to remove the finalizer
		Exec("kubectl", []string{"-n", "argo", "patch", "pod", func() string {
			podList, err := s.KubeClient.CoreV1().Pods("argo").List(logging.TestContext(s.T().Context()), metav1.ListOptions{LabelSelector: "workflows.argoproj.io/workflow"})
			if err != nil {
				panic(err)
			}
			return podList.Items[0].Name
		}(), "-p", `{"metadata":{"finalizers":[]}}`, "--type", "merge"}, fixtures.OutputRegexp(`pod/.* patched`)).
		Wait(time.Second).
		Exec("kubectl", []string{"-n", "argo", "delete", "pod", "-l", "workflows.argoproj.io/workflow"}, fixtures.OutputRegexp(`pod "pending-.*" deleted`)).
		Wait(time.Duration(3*fixtures.EnvFactor)*time.Second). // allow 3s for reconciliation, we'll create a new pod
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
		Workflow("@corefunctional/metadata.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "bar", metadata.Labels["my-label"])
		})
}

func (s *FunctionalSuite) TestWhenExpressions() {
	s.Given().
		Workflow("@functional/conditionals.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 150*time.Second).
		Then().
		ExpectWorkflowNode(wfv1.NodeWithDisplayName("print-hello-govaluate"), func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.NotEqual(t, wfv1.NodeTypeSkipped, n.Type)
		}).
		ExpectWorkflowNode(wfv1.NodeWithDisplayName("print-hello-expr"), func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.NotEqual(t, wfv1.NodeTypeSkipped, n.Type)
		}).
		ExpectWorkflowNode(wfv1.NodeWithDisplayName("print-hello-expr-json"), func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.NotEqual(t, wfv1.NodeTypeSkipped, n.Type)
		})
}

func (s *FunctionalSuite) TestJSONVariables() {
	s.Given().
		Workflow("@testdata/json-variables.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			for _, c := range p.Spec.Containers {
				if c.Name == "main" {
					assert.Len(t, c.Args, 3)
					assert.Equal(t, "myLabelValue", c.Args[0])
					assert.Equal(t, "myAnnotationValue", c.Args[1])
					assert.Equal(t, "myParamValue", c.Args[2])
				}
			}
		})
}

func (s *FunctionalSuite) TestWorkflowTTL() {
	s.Given().
		Workflow("@corefunctional/workflow-ttl.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Wait(3 * time.Second). // enough time for TTL controller to delete the workflow
		Then().
		ExpectWorkflowDeleted()
}

func (s *FunctionalSuite) TestWorkflowRetention() {
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
		WaitForWorkflowListFailedCount(2)
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
		Workflow("@corefunctional/continue-on-fail.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 90*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Len(t, status.Nodes, 7)
			nodeStatus := status.Nodes.FindByDisplayName("B")
			require.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeFailed, nodeStatus.Phase)
			assert.Len(t, nodeStatus.Children, 1)
			assert.Len(t, nodeStatus.OutboundNodes, 1)
		})
}

func (s *FunctionalSuite) TestContinueOnFailDag() {
	s.Given().
		Workflow("@corefunctional/continue-on-failed-dag.yaml").
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
			ctx, cancel := context.WithTimeout(logging.TestContext(t.Context()), 15*time.Second)
			defer cancel()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					t.Error("timeout waiting for PVC to be deleted")
					t.FailNow()
				case <-ticker.C:
					list, err := s.KubeClient.CoreV1().PersistentVolumeClaims(fixtures.Namespace).List(logging.TestContext(t.Context()), metav1.ListOptions{})
					require.NoError(t, err)
					if len(list.Items) == 0 {
						return
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
						assert.Equal(t, "Pod", e.Annotations["workflows.argoproj.io/node-type"])
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
		WaitForWorkflow(90*time.Second).
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

func (s *FunctionalSuite) TestEventOnPVCFail() {
	//  Test whether an WorkflowFailed event (with appropriate message) is emitted in case of error in creating the PVC
	var uid types.UID
	s.Given().
		Workflow("@expectedfailures/volumes-pvc-fail-event.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(150*time.Second).
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
			require.NotEmpty(t, status.ArtifactRepositoryRef)
			assert.Equal(t, "argo", status.ArtifactRepositoryRef.Namespace)
			assert.Equal(t, "artifact-repositories", status.ArtifactRepositoryRef.ConfigMap)
			assert.Equal(t, "my-key", status.ArtifactRepositoryRef.Key)
			assert.False(t, status.ArtifactRepositoryRef.Default)

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
			require.Len(t, status.Nodes, 5)
			nodeStatus := status.Nodes.FindByDisplayName("sleep")
			assert.Equal(t, wfv1.NodeSkipped, nodeStatus.Phase)
			assert.Equal(t, "Skipped, empty params", nodeStatus.Message)
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
			require.Len(t, status.Nodes, 3)
			nodeStatus := status.Nodes.FindByDisplayName("sleep")
			assert.Equal(t, wfv1.NodeSkipped, nodeStatus.Phase)
			assert.Equal(t, "Skipped, empty params", nodeStatus.Message)
		})
}

// 128M is for argo executor
func (s *FunctionalSuite) TestPendingRetryWorkflow() {
	s.Given().
		Workflow("@corefunctional/pending-retry-workflow.yaml").
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
		Workflow("@corefunctional/pending-retry-workflow-with-retry-strategy.yaml").
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
		WaitForWorkflow(time.Second * 90).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("print(0:res:1)")
			require.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
		})
}

func (s *FunctionalSuite) TestParameterAggregationFromOutputs() {
	s.Given().
		Workflow("@functional/param-aggregation-fromoutputs.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Second * 90).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			assert.NotNil(t, status.Nodes.FindByDisplayName("task3(0:key1:value1)"))
			assert.NotNil(t, status.Nodes.FindByDisplayName("task3(1:key2:value2)"))
			assert.NotNil(t, status.Nodes.FindByDisplayName("task3(2:key3:value3)"))
			assert.NotNil(t, status.Nodes.FindByDisplayName("task3(0:key4:value4)"))
			assert.NotNil(t, status.Nodes.FindByDisplayName("task3(1:key5:value5)"))
		})
}

func (s *FunctionalSuite) TestParameterAggregationDAGWithRetry() {
	s.Given().
		Workflow("@functional/parameter-aggregation-dag-with-retry.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Second * 90).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("parameter-aggregation-dag-with-retry(0)")
			require.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			require.NotNil(t, nodeStatus.Outputs)
			assert.Len(t, nodeStatus.Outputs.Parameters, 1)
			assert.Equal(t, `["1","2","3"]`, nodeStatus.Outputs.Parameters[0].Value.String())
		})
}

func (s *FunctionalSuite) TestParameterAggregationStepsWithRetry() {
	s.Given().
		Workflow("@functional/parameter-aggregation-steps-with-retry.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Second * 90).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			nodeStatus := status.Nodes.FindByDisplayName("parameter-aggregation-steps-with-retry(0)")
			require.NotNil(t, nodeStatus)
			assert.Equal(t, wfv1.NodeSucceeded, nodeStatus.Phase)
			require.NotNil(t, nodeStatus.Outputs)
			assert.Len(t, nodeStatus.Outputs.Parameters, 1)
			assert.Equal(t, `["1","2","3"]`, nodeStatus.Outputs.Parameters[0].Value.String())
		})
}

func (s *FunctionalSuite) TestDAGDepends() {
	s.Given().
		Workflow("@functional/dag-depends.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Second * 90).
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
		WorkflowTemplate("@corefunctional/test-exit-handler.yaml").
		Workflow("@corefunctional/test-exit-handler.yaml").
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Contains(t, status.Message, "invalid spec")
		})
}

func (s *FunctionalSuite) TestWorkflowLifecycleHookWithWorkflowTemplate() {
	s.Given().
		WorkflowTemplate("@corefunctional/test-exit-handler.yaml").
		Workflow("@corefunctional/test-lifecycle-hook.yaml").
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

func (s *FunctionalSuite) TestWorkflowHookParameterTemplates() {
	s.Given().
		Workflow("@testdata/workflow-hook-parameter.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(wfv1.NodeWithDisplayName("workflow-hook-parameter.onExit"), func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
			assert.Equal(t, "Succeeded", n.Inputs.Parameters[0].Value.String())
			assert.Equal(t, "Succeeded", n.Inputs.Parameters[1].Value.String())
		})
}

func (s *FunctionalSuite) TestParametrizableAds() {
	s.Given().
		Workflow("@corefunctional/param-ads.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflowNode(wfv1.FailedPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.Equal(t, int64(5), *p.Spec.ActiveDeadlineSeconds)
		})
}

func (s *FunctionalSuite) TestParametrizableLimit() {
	s.Given().
		Workflow("@corefunctional/param-limit.yaml").
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
		Workflow("@corefunctional/steps-tmpl-timeout.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			return wf.Status.Phase == wfv1.WorkflowFailed, "Waiting for timeout"
		}), 60*time.Second)
}

func (s *FunctionalSuite) TestTemplateLevelTimeoutWithForbidden() {
	s.Given().
		Workflow("@corefunctional/steps-tmpl-timeout.yaml").
		When().
		MemoryQuota("130M").
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed)
}

func (s *FunctionalSuite) TestWorkflowPodSpecPatch() {
	s.Given().
		Workflow("@corefunctional/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflowNode(wfv1.SucceededPodNode, func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.Equal(t, int64(5), *p.Spec.TerminationGracePeriodSeconds)
			for _, c := range p.Spec.Containers {
				switch c.Name {
				case "main":
					assert.Equal(t, "100m", c.Resources.Limits.Cpu().String())
				case "wait":
					assert.Equal(t, "101m", c.Resources.Limits.Cpu().String())
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
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			paths := status.Nodes.FindByDisplayName("get-artifact-path")
			require.NotNil(t, paths)
			assert.Equal(t, `["foo/script.py","script.py"]`, *paths.Outputs.Result)

			assert.NotNil(t, status.Nodes.FindByDisplayName("process-artifact(0:foo/script.py)"))
			assert.NotNil(t, status.Nodes.FindByDisplayName("process-artifact(1:script.py)"))
			for _, value := range status.TaskResultsCompletionStatus {
				assert.True(t, value)
			}
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
		Workflow("@corefunctional/script-nonroot.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestPauseBefore() {
	s.Given().
		Workflow(`@functional/pause-before.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveRunningPod).
		Exec("bash", []string{"-c", "sleep 5 &&  kubectl exec -i $(kubectl get pods | awk '/pause-before/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/before'"}, fixtures.NoError).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestPauseAfter() {
	s.Given().
		Workflow(`@functional/pause-after.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveRunningPod).
		Exec("bash", []string{"-c", "sleep 5 && kubectl exec -i $(kubectl get pods -n argo | awk '/pause-after/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/after'"}, fixtures.NoError).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestPauseAfterAndBefore() {
	s.Given().
		Workflow(`@functional/pause-before-after.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHaveRunningPod).
		Exec("bash", []string{"-c", "sleep 5 && kubectl exec -i $(kubectl get pods | awk '/pause-before-after/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/before'"}, fixtures.NoError).
		Exec("bash", []string{"-c", "kubectl exec -i $(kubectl get pods | awk '/pause-before-after/ {print $1;exit}') -c main -- bash -c 'touch /proc/1/root/run/argo/ctr/main/after'"}, fixtures.NoError).
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}

func (s *FunctionalSuite) TestStepLevelMemoize() {
	s.Given().
		Workflow("@corefunctional/steps-memoize.yaml").
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

func (s *FunctionalSuite) TestStepLevelMemoizeNoOutput() {
	s.Given().
		Workflow("@corefunctional/steps-memoize-noout.yaml").
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

func (s *FunctionalSuite) TestDAGLevelMemoize() {
	s.Given().
		Workflow("@corefunctional/dag-memoize.yaml").
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

func (s *FunctionalSuite) TestDAGLevelMemoizeNoOutput() {
	s.Given().
		Workflow("@corefunctional/dag-memoize-noout.yaml").
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
		Workflow("@corefunctional/containerset-retry-success.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestContainerSetRetrySuccess() {
	s.Given().
		Workflow("@corefunctional/containerset-no-retry-fail.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed)
}

func (s *FunctionalSuite) TestTTY() {
	s.Given().
		Workflow(`@functional/tty.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestTemplateDefaultImage() {
	s.Given().
		Workflow(`@functional/template-default-image.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

func (s *FunctionalSuite) TestEntrypointName() {
	s.Given().
		WorkflowTemplate(`@functional/entrypointName-template.yaml`).
		Workflow(`@functional/entrypointName-workflow.yaml`).
		When().
		CreateWorkflowTemplates().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflowNode(wfv1.NodeWithDisplayName("step"), func(t *testing.T, n *wfv1.NodeStatus, p *apiv1.Pod) {
			assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
			assert.Equal(t, "bar", n.Inputs.Parameters[0].Value.String())
		})
}

func (s *FunctionalSuite) TestMissingStepsInUI() {
	s.Given().
		Workflow(`@functional/missing-steps.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflowNode(wfv1.NodeWithName(`missing-steps[0].step1[0].execute-script`), func(t *testing.T, n *wfv1.NodeStatus, _ *apiv1.Pod) {
			assert.NotNil(t, n)
			assert.NotNil(t, n.Children)
			assert.Len(t, n.Children, 1)
		})
}

func (s *FunctionalSuite) TestWithItemsWithHooks() {
	s.Given().
		Workflow("@smoke/with-items-with-hooks.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded)
}

// when you terminate a workflow with onexit handler,
// then the onexit handler should fail along with steps and stepsGroup
func (s *FunctionalSuite) TestTerminateWorkflowWhileOnExitHandlerRunning() {
	s.Given().
		Workflow("@functional/workflow-exit-handler-sleep.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		WaitForWorkflow(fixtures.Condition(func(wf *wfv1.Workflow) (bool, string) {
			a := wf.Status.Nodes.FindByDisplayName("workflow-exit-handler-sleep")
			return wfv1.NodeSucceeded == a.Phase, "nodes succeeded"
		})).
		ShutdownWorkflow(wfv1.ShutdownStrategyTerminate).
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			for _, node := range status.Nodes {
				if node.Type == wfv1.NodeTypeStepGroup || node.Type == wfv1.NodeTypeSteps {
					assert.Equal(t, wfv1.NodeFailed, node.Phase)
				}
			}
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
		})
}

// Exit handler ensure when failed steps ensure no crash and output parameter
func (s *FunctionalSuite) TestWorkflowExitHandlerCrashEnsureNodeIsPresent() {
	s.Given().
		Workflow("@expectedfailures/exit-handler-fail-missing-output.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			var hasExitNode bool
			var exitNodeName string

			for _, node := range status.Nodes {
				if !node.IsExitNode() {
					continue
				}
				hasExitNode = true
				exitNodeName = node.DisplayName
			}
			assert.True(t, hasExitNode)
			assert.NotEmpty(t, exitNodeName)

			hookNode := status.Nodes.FindByDisplayName(exitNodeName)

			require.NotNil(t, hookNode)
			assert.NotNil(t, hookNode.Inputs)
			require.Len(t, hookNode.Inputs.Parameters, 1)
			assert.NotNil(t, hookNode.Inputs.Parameters[0].Value)
		})
}

func (s *FunctionalSuite) TestWorkflowParallelismStepFailFast() {
	s.Given().
		Workflow("@expectedfailures/parallelism-step-fail-fast.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "template has failed or errored children and failFast enabled", status.Message)
			assert.Equal(t, wfv1.NodeFailed, status.Nodes.FindByDisplayName("[0]").Phase)
			assert.Equal(t, wfv1.NodeFailed, status.Nodes.FindByDisplayName("step1").Phase)
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("step2").Phase)
		})
}

func (s *FunctionalSuite) TestWorkflowParallelismDAGFailFast() {
	s.Given().
		Workflow("@expectedfailures/parallelism-dag-fail-fast.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning).
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "template has failed or errored children and failFast enabled", status.Message)
			assert.Equal(t, wfv1.NodeFailed, status.Nodes.FindByDisplayName("task1").Phase)
			assert.Equal(t, wfv1.NodeSucceeded, status.Nodes.FindByDisplayName("task2").Phase)
		})
}

func (s *FunctionalSuite) TestWorkflowInvalidServiceAccountError() {
	s.Given().
		Workflow("@expectedfailures/serviceaccount-insufficient-permissions.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored).
		Then().
		ExpectContainerLogs("main", func(t *testing.T, logs string) {
			assert.Contains(t, logs, "hello argo")
		}).
		ExpectContainerLogs("wait", func(t *testing.T, logs string) {
			assert.Contains(t, logs, "Error: workflowtaskresults.argoproj.io is forbidden: User \"system:serviceaccount:argo:github.com\" cannot create resource")
			// Shouldn't have print help text
			assert.NotContains(t, logs, "Usage:")
		})
}
