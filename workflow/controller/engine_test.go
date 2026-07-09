package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestEngineExecuteFullLifecycle exercises all branches in Engine.Execute using
// a single DAG workflow that is driven to completion over multiple operate cycles.
//
// The DAG structure:
//
//	task-a ──→ task-b (withItems: ["x","y"])  ──→ task-e
//	       └─→ task-c (exit hook)            ──┘
//	       └─→ task-d (depends: task-a.Failed → omitted)
//
// Branches covered:
//   - converge:                      task-a, task-b, task-c, task-e scheduled across cycles
//   - assessTaskGroups:              task-b's TaskGroup assessed after children complete
//   - processHooks (1st pass):       task-c's exit hook fires after task-c completes
//   - processHooks (2nd pass):       exit hook completion detected in same cycle
//   - reconcileExternalCompletions:  pods that succeed between cycles are re-reconciled
//   - createOmittedNodes:            task-d omitted because task-a.Failed is false
//   - finalize:                      DAG transitions Running → Succeeded
var engineFullLifecycleDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: engine-lifecycle
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: task-a
        template: echo
      - name: task-b
        depends: "task-a"
        template: echo
        withItems: ["x", "y"]
      - name: task-c
        depends: "task-a"
        template: echo
        hooks:
          exit:
            template: echo
      - name: task-d
        depends: "task-a.Failed"
        template: echo
      - name: task-e
        depends: "task-b && task-c"
        template: echo
  - name: echo
    container:
      image: busybox
      command: [echo, hello]
`

func TestEngineExecuteFullLifecycle(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(engineFullLifecycleDAG)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// ── Cycle 1: task-a is the only task with no dependencies → scheduled ──
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	taskA := woc.wf.Status.Nodes.FindByDisplayName("task-a")
	require.NotNil(t, taskA, "task-a should be created")
	assert.Equal(t, wfv1.NodePending, taskA.Phase)

	// No other task nodes should exist yet.
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("task-b"))
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("task-c"))
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("task-d"))
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("task-e"))

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1, "one pod for task-a")

	// ── Cycle 2: task-a succeeds → task-b (withItems) + task-c scheduled, task-d omitted ──
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	taskA = woc.wf.Status.Nodes.FindByDisplayName("task-a")
	require.NotNil(t, taskA)
	assert.Equal(t, wfv1.NodeSucceeded, taskA.Phase)

	// task-b is a TaskGroup (withItems expansion)
	taskB := woc.wf.Status.Nodes.FindByDisplayName("task-b")
	require.NotNil(t, taskB, "task-b TaskGroup should exist")
	assert.Equal(t, wfv1.NodeTypeTaskGroup, taskB.Type)

	taskC := woc.wf.Status.Nodes.FindByDisplayName("task-c")
	require.NotNil(t, taskC, "task-c should be scheduled")
	assert.Equal(t, wfv1.NodePending, taskC.Phase)

	// task-d depends on task-a.Failed, but task-a succeeded → omitted
	taskD := woc.wf.Status.Nodes.FindByDisplayName("task-d")
	require.NotNil(t, taskD, "task-d should be created as omitted")
	assert.Equal(t, wfv1.NodeOmitted, taskD.Phase)

	// task-e still waiting on task-b and task-c
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("task-e"))

	// ── Cycle 3: task-b items + task-c succeed → exit hook fires, task-e scheduled ──
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	// task-b TaskGroup should be assessed as Succeeded
	taskB = woc.wf.Status.Nodes.FindByDisplayName("task-b")
	require.NotNil(t, taskB)
	assert.Equal(t, wfv1.NodeSucceeded, taskB.Phase, "TaskGroup should be Succeeded after children complete")

	taskC = woc.wf.Status.Nodes.FindByDisplayName("task-c")
	require.NotNil(t, taskC)
	assert.Equal(t, wfv1.NodeSucceeded, taskC.Phase)

	// task-c's exit hook should have been created by processHooks
	var exitHookNode *wfv1.NodeStatus
	for _, node := range woc.wf.Status.Nodes {
		if node.Type == wfv1.NodeTypePod && node.NodeFlag != nil && node.NodeFlag.Hooked {
			exitHookNode = &node
			break
		}
	}
	require.NotNil(t, exitHookNode, "exit hook node should exist")

	// task-e is NOT yet scheduled — task-c's exit hook is still pending,
	// and evaluateDependsReadiness treats deps with pending hooks as not ready.
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("task-e"),
		"task-e should wait for task-c's exit hook to complete")

	// DAG still running
	dagNode := woc.wf.Status.Nodes.FindByDisplayName("engine-lifecycle")
	require.NotNil(t, dagNode)
	assert.False(t, dagNode.Fulfilled(), "DAG should still be running")

	// ── Cycle 4: exit hook succeeds → task-e is now schedulable ──
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	taskE := woc.wf.Status.Nodes.FindByDisplayName("task-e")
	require.NotNil(t, taskE, "task-e should be scheduled after exit hook completes")

	// DAG still running (task-e not yet complete)
	dagNode = woc.wf.Status.Nodes.FindByDisplayName("engine-lifecycle")
	require.NotNil(t, dagNode)
	assert.False(t, dagNode.Fulfilled(), "DAG should still be running")

	// ── Cycle 5: task-e succeeds → DAG completes ──
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	taskE = woc.wf.Status.Nodes.FindByDisplayName("task-e")
	require.NotNil(t, taskE)
	assert.Equal(t, wfv1.NodeSucceeded, taskE.Phase)

	dagNode = woc.wf.Status.Nodes.FindByDisplayName("engine-lifecycle")
	require.NotNil(t, dagNode)
	assert.Equal(t, wfv1.NodeSucceeded, dagNode.Phase, "DAG should be Succeeded")

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "workflow should be Succeeded")
}

// TestAssessDAGPhaseDoesNotHangOnOmittedTasks is a regression test that verifies
// assessDAGPhase correctly handles transitively-omitted tasks. When task B is
// omitted because its dependency expression (A.Failed) is not satisfied, and
// task C depends on B, C should also be omitted — and the workflow should
// complete as Succeeded rather than getting stuck at Running.
//
// DAG structure:
//
//	A (succeeds) ──→ B (depends: "A.Failed" → omitted) ──→ C (depends: "B" → transitively omitted)
var assessPhaseOmitDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: assess-phase-omit-test
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        depends: "A.Failed"
        template: echo
      - name: C
        depends: "B"
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

func TestAssessDAGPhaseDoesNotHangOnOmittedTasks(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(assessPhaseOmitDAG)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// ── Cycle 1: A is the only root task → scheduled as Pending ──
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	taskA := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, taskA, "A should be created")
	assert.Equal(t, wfv1.NodePending, taskA.Phase)

	// B and C should not exist yet — they depend on A.
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("B"))
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("C"))

	// ── Cycle 2: A succeeds → B omitted (A.Failed is false), C transitively omitted, DAG completes ──
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	taskA = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, taskA)
	assert.Equal(t, wfv1.NodeSucceeded, taskA.Phase)

	taskB := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, taskB, "B should be created as omitted")
	assert.Equal(t, wfv1.NodeOmitted, taskB.Phase, "B should be omitted because A.Failed is false")

	taskC := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, taskC, "C should be created as omitted")
	assert.Equal(t, wfv1.NodeOmitted, taskC.Phase, "C should be transitively omitted because B is omitted")

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("assess-phase-omit-test")
	require.NotNil(t, dagNode)
	assert.Equal(t, wfv1.NodeSucceeded, dagNode.Phase, "DAG should be Succeeded, not stuck at Running")

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "workflow should be Succeeded")
}
