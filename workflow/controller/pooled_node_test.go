package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

const pooledNodesWf = `
metadata:
  name: pooled
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
  - name: main-parallelism
    parallelism: 1
    steps:
    - - name: a
        template: work
  - name: main-fail-fast
    failFast: true
    steps:
    - - name: a
        template: work
  - name: work
    container:
      image: alpine:3.23
  - name: work-pending-timeout
    pendingTimeout: 1m
    container:
      image: alpine:3.23
`

// pooledChild names the child of the Steps node the tests build.
const pooledChild = "pooled[0].a"

// pooledTask is the ledger of a node whose task has been enqueued for a pool
// and taken to state, claimed by work-0123456789-abcde.
func pooledTask(state wfv1.PooledTaskState) *wfv1.PooledTaskStatus {
	task := &wfv1.PooledTaskStatus{
		PoolName:   "work",
		PoolHash:   "hash",
		TaskName:   "task",
		TaskSetUID: "task-set-uid",
	}
	now := metav1.Now()
	task.Enqueue(now)
	if state != wfv1.PooledTaskEnqueued {
		task.Claim("claim-uid", now, "work-0123456789-abcde", "worker-pod-uid", 0)
	}
	if state == wfv1.PooledTaskInterrupted {
		task.Interrupt(now)
	}
	return task
}

// newPooledNodesWoc returns an operation context for pooledNodesWf whose
// running Steps node, run with boundaryTmpl, has the given children.
func newPooledNodesWoc(t *testing.T, boundaryTmpl string, children ...wfv1.NodeStatus) (context.Context, *wfOperationCtx) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(pooledNodesWf)
	wf.Status.Phase = wfv1.WorkflowRunning
	boundary := wfv1.NodeStatus{
		ID:           wf.NodeID(wf.Name),
		Name:         wf.Name,
		DisplayName:  wf.Name,
		Type:         wfv1.NodeTypeSteps,
		TemplateName: boundaryTmpl,
		Phase:        wfv1.NodeRunning,
		StartedAt:    metav1.Now(),
	}
	wf.Status.Nodes = wfv1.Nodes{boundary.ID: boundary}
	for _, child := range children {
		child.ID = wf.NodeID(child.Name)
		child.DisplayName = child.Name
		child.Type = wfv1.NodeTypePod
		child.BoundaryID = boundary.ID
		if child.TemplateName == "" {
			child.TemplateName = "work"
		}
		if child.StartedAt.IsZero() {
			child.StartedAt = metav1.Now()
		}
		wf.Status.Nodes[child.ID] = child
	}
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)
	return ctx, newWorkflowOperationCtx(ctx, wf, controller)
}

// addChildPod puts a pod carrying the child node's ID in the pod informer.
func addChildPod(t *testing.T, woc *wfOperationCtx, phase apiv1.PodPhase) *apiv1.Pod {
	t.Helper()
	node, err := woc.wf.GetNodeByName(pooledChild)
	require.NoError(t, err)
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        woc.getPodName(node.Name, node.TemplateName),
			Namespace:   woc.wf.Namespace,
			UID:         "pod-uid",
			Labels:      map[string]string{common.LabelKeyWorkflow: woc.wf.Name},
			Annotations: map[string]string{common.AnnotationKeyNodeID: node.ID, common.AnnotationKeyNodeName: node.Name},
		},
		Status: apiv1.PodStatus{Phase: phase},
	}
	require.NoError(t, woc.controller.PodController.TestingPodInformer().GetIndexer().Add(pod))
	return pod
}

func nodePhase(t *testing.T, woc *wfOperationCtx, nodeName string) wfv1.NodePhase {
	t.Helper()
	node, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	return node.Phase
}

// TestPooledNodeCountsAsBackingExecution checks that, once enqueued, a pooled
// node with no pod counts against workflow, template and boundary parallelism
// and holds back failFast, as a created pending pod does.
func TestPooledNodeCountsAsBackingExecution(t *testing.T) {
	cases := []struct {
		name   string
		phase  wfv1.NodePhase
		pooled *wfv1.PooledTaskStatus
		pod    bool
		active bool
	}{
		{name: "enqueued pooled node", phase: wfv1.NodePending, pooled: pooledTask(wfv1.PooledTaskEnqueued), active: true},
		{name: "claimed pooled node", phase: wfv1.NodeRunning, pooled: pooledTask(wfv1.PooledTaskClaimed), active: true},
		{name: "created pending pod", phase: wfv1.NodePending, pod: true, active: true},
		{name: "pending node with no pod yet", phase: wfv1.NodePending},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newWoc := func(t *testing.T, boundaryTmpl string, failedSibling bool) (context.Context, *wfOperationCtx, *wfv1.NodeStatus) {
				children := []wfv1.NodeStatus{{Name: pooledChild, Phase: tc.phase, PooledTask: tc.pooled}}
				if failedSibling {
					children = append(children, wfv1.NodeStatus{Name: "pooled[0].b", Phase: wfv1.NodeFailed})
				}
				ctx, woc := newPooledNodesWoc(t, boundaryTmpl, children...)
				if tc.pod {
					addChildPod(t, woc, apiv1.PodPending)
				}
				boundary, err := woc.wf.GetNodeByName("pooled")
				require.NoError(t, err)
				return ctx, woc, boundary
			}
			work := func(woc *wfOperationCtx) *wfv1.Template {
				return woc.execWf.GetTemplateByName("work")
			}

			t.Run("counted", func(t *testing.T) {
				_, woc, boundary := newWoc(t, "main", false)
				want := int64(0)
				if tc.active {
					want = 1
				}
				assert.Equal(t, want, woc.getActivePods(""))
				assert.Equal(t, want, woc.getActivePods(boundary.ID))
			})

			t.Run("workflow parallelism", func(t *testing.T) {
				ctx, woc, boundary := newWoc(t, "main", false)
				woc.execWf.Spec.Parallelism = new(int64(1))
				// as operate does
				woc.activePods = woc.getActivePods("")
				err := woc.checkParallelism(ctx, work(woc), nil, boundary.ID)
				if tc.active {
					require.ErrorIs(t, err, ErrParallelismReached)
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("template parallelism", func(t *testing.T) {
				ctx, woc, boundary := newWoc(t, "main-parallelism", false)
				err := woc.checkParallelism(ctx, woc.execWf.GetTemplateByName("main-parallelism"), boundary, "")
				if tc.active {
					require.ErrorIs(t, err, ErrParallelismReached)
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("boundary parallelism", func(t *testing.T) {
				// counts children by phase alone, so any pending child counts
				ctx, woc, boundary := newWoc(t, "main-parallelism", false)
				err := woc.checkParallelism(ctx, work(woc), nil, boundary.ID)
				require.ErrorIs(t, err, ErrParallelismReached)
			})

			t.Run("template failFast", func(t *testing.T) {
				ctx, woc, boundary := newWoc(t, "main-fail-fast", true)
				err := woc.checkParallelism(ctx, woc.execWf.GetTemplateByName("main-fail-fast"), boundary, "")
				require.ErrorIs(t, err, ErrParallelismReached)
				// the boundary fails only once nothing in it is still running
				assert.Equal(t, tc.active, nodePhase(t, woc, "pooled") == wfv1.NodeRunning)
			})

			t.Run("boundary failFast", func(t *testing.T) {
				ctx, woc, boundary := newWoc(t, "main-fail-fast", true)
				err := woc.checkParallelism(ctx, work(woc), nil, boundary.ID)
				require.ErrorIs(t, err, ErrParallelismReached)
				assert.Equal(t, tc.active, nodePhase(t, woc, "pooled") == wfv1.NodeRunning)
			})
		})
	}
}

// TestPooledNodeNotMissing checks that a started pooled node with no pod is
// never marked missing, and that a pod carrying its node ID is not taken for
// its task's.
func TestPooledNodeNotMissing(t *testing.T) {
	cases := []struct {
		name   string
		pooled *wfv1.PooledTaskStatus
	}{
		{name: "pooled node", pooled: pooledTask(wfv1.PooledTaskClaimed)},
		{name: "pod node"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("no pod", func(t *testing.T) {
				ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
					Name:       pooledChild,
					Phase:      wfv1.NodeRunning,
					StartedAt:  metav1.NewTime(time.Now().Add(-time.Hour)),
					PooledTask: tc.pooled,
				})
				_, err := woc.podReconciliation(ctx)
				require.NoError(t, err)

				node, err := woc.wf.GetNodeByName(pooledChild)
				require.NoError(t, err)
				attribs := attribute.NewSet(
					attribute.String(telemetry.AttribNodePhase, string(wfv1.NodeRunning)),
					attribute.Bool(telemetry.AttribRecentlyStarted, false),
				)
				missing, err := testExporter.GetInt64CounterValue(ctx, telemetry.InstrumentPodMissing.Name(), &attribs)
				if tc.pooled == nil {
					assert.Equal(t, wfv1.NodeError, node.Phase)
					assert.Equal(t, "pod deleted", node.Message)
					require.NoError(t, err)
					assert.Equal(t, int64(1), missing)
				} else {
					assert.Equal(t, wfv1.NodeRunning, node.Phase)
					assert.Empty(t, node.Message)
					require.Error(t, err, "no pod_missing series for a pooled node")
				}
			})

			t.Run("pod carrying its node ID", func(t *testing.T) {
				ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
					Name:       pooledChild,
					Phase:      wfv1.NodeRunning,
					PooledTask: tc.pooled,
				})
				addChildPod(t, woc, apiv1.PodFailed)
				_, err := woc.podReconciliation(ctx)
				require.NoError(t, err)
				want := wfv1.NodeRunning
				if tc.pooled == nil {
					want = wfv1.NodeFailed
				}
				assert.Equal(t, want, nodePhase(t, woc, pooledChild))
			})
		})
	}
}

// TestPooledNodeNotKilled checks that stop, terminate, the workflow deadline
// and a template timeout never act on a pod carrying a pooled node's ID, and
// that the pod-lookup paths never return one for it.
func TestPooledNodeNotKilled(t *testing.T) {
	cases := []struct {
		name   string
		pooled *wfv1.PooledTaskStatus
	}{
		{name: "pooled node", pooled: pooledTask(wfv1.PooledTaskClaimed)},
		{name: "pod node"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			podNode := tc.pooled == nil

			for _, ending := range []string{"terminate", "stop", "workflow deadline"} {
				t.Run(ending, func(t *testing.T) {
					ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{Name: pooledChild, Phase: wfv1.NodeRunning, PooledTask: tc.pooled})
					switch ending {
					case "terminate":
						woc.execWf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
					case "stop":
						woc.execWf.Spec.Shutdown = wfv1.ShutdownStrategyStop
					default:
						woc.workflowDeadline = new(time.Now().Add(-time.Minute))
					}
					pod := addChildPod(t, woc, apiv1.PodRunning)

					woc.applyExecutionControl(ctx, pod, &sync.RWMutex{})

					terminations := woc.controller.PodController.TestingQueueNumRequeues(pod.Namespace + "/" + pod.Name + "/terminateContainers")
					assert.Equal(t, podNode, terminations > 0)
					want := wfv1.NodeRunning
					if podNode {
						want = wfv1.NodeFailed
					}
					assert.Equal(t, want, nodePhase(t, woc, pooledChild))
				})
			}

			t.Run("template timeout", func(t *testing.T) {
				var pooled *wfv1.PooledTaskStatus
				if !podNode {
					pooled = pooledTask(wfv1.PooledTaskEnqueued)
				}
				ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
					Name:         pooledChild,
					TemplateName: "work-pending-timeout",
					Phase:        wfv1.NodePending,
					StartedAt:    metav1.NewTime(time.Now().Add(-time.Hour)),
					PooledTask:   pooled,
				})
				pod := addChildPod(t, woc, apiv1.PodPending)
				node, err := woc.wf.GetNodeByName(pooledChild)
				require.NoError(t, err)

				err = woc.checkConstraints(ctx, node.Name, node, woc.execWf.GetTemplateByName("work-pending-timeout"), node.BoundaryID)
				if podNode {
					require.ErrorIs(t, err, ErrTimeout)
				} else {
					require.NoError(t, err, "a pooled node's timeouts don't end a pod")
				}

				deletions := woc.controller.PodController.TestingQueueNumRequeues(pod.Namespace + "/" + pod.Name + "/deletePodByUID/" + string(pod.UID))
				assert.Equal(t, podNode, deletions > 0)
			})

			t.Run("pod by node", func(t *testing.T) {
				_, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{Name: pooledChild, Phase: wfv1.NodeRunning, PooledTask: tc.pooled})
				addChildPod(t, woc, apiv1.PodRunning)
				node, err := woc.wf.GetNodeByName(pooledChild)
				require.NoError(t, err)

				pod, err := woc.getPodByNode(node)
				require.NoError(t, err)
				assert.Equal(t, podNode, pod != nil)
			})
		})
	}
}
