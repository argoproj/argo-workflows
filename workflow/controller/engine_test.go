package controller

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
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
//   - visit:                         task-a, task-b, task-c, task-e scheduled across cycles
//   - reconcileTaskGroup:            task-b's TaskGroup completed after its items complete
//   - processHooks:                  task-c's exit hook fires after task-c completes
//   - reconcileExternalCompletions:  pods that succeed between cycles are re-reconciled
//   - visit (omission):              task-d omitted because task-a.Failed is false
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

// An evaluator error for a task must be recorded as a terminal Error node.
// Left unrecorded, the task would stay Pending and the boundary would never
// complete.
func TestVisit_EvaluatorErrorBecomesErrorNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, _, tasks := engineWithFakeReconciler(ctx, t)

	result := dag.EvaluationResult{
		TaskName:     "client",
		CurrentPhase: wfv1.NodePending,
		Error:        errors.New("depends expression failed to evaluate"),
	}
	task := engine.getTaskByName(tasks, "client")
	fake.calls = nil
	ran, _ := engine.visit(ctx, task, result, true)
	assert.Empty(t, fake.calls, "an unassessable task must not be dispatched")
	assert.False(t, ran)

	node, err := engine.woc.wf.GetNodeByName(engine.taskNodeName("client"))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Equal(t, "depends expression failed to evaluate", node.Message)

	// Idempotent: a second visit neither re-creates nor re-dispatches.
	ran, _ = engine.visit(ctx, task, result, true)
	assert.Empty(t, fake.calls)
	assert.False(t, ran)
}

// An explicitly empty withItems list is not an expansion: the task runs once
// as a plain pod (as in every release), not as a TaskGroup with no children.
func TestDAGEmptyWithItemsRunsOnce(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-empty-withitems
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
        withItems: []
  - name: echo
    container:
      image: argoproj/argosay:v2
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	node := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	for _, n := range woc.wf.Status.Nodes {
		assert.NotEqual(t, wfv1.NodeTypeTaskGroup, n.Type, "node %q must not be a TaskGroup", n.Name)
	}
}

// Dispatching an expanded task completes its TaskGroup once every item has
// finished, with the worst of their phases: Error outranks Failed regardless
// of item order.
func TestReconcileTaskGroup_WorstPhaseWins(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := engineWithFakeReconciler(ctx, t)
	markChildPhase(t, woc, "client(0:0)", wfv1.NodeFailed)
	markChildPhase(t, woc, "client(1:1)", wfv1.NodeError)
	markChildPhase(t, woc, "client(2:2)", wfv1.NodeSucceeded)

	engine.visit(ctx, engine.getTaskByName(tasks, "client"), engine.evaluator.Evaluate(ctx, "client"), true)

	assert.Empty(t, fake.calls, "finished items are not reconciled again")
	tgNode, err := woc.wf.GetNodeByName(engine.taskNodeName("client"))
	require.NoError(t, err)
	require.Equal(t, wfv1.NodeTypeTaskGroup, tgNode.Type)
	assert.Equal(t, wfv1.NodeError, tgNode.Phase)
}

// The Engine logs the evaluator's diagnostics for a waiting task at debug
// level, so why a task has not started can be read from the logs.
func TestEngineLogsWaitingTask(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: log-waiting
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: a
        template: echo
      - name: b
        template: echo
        depends: a
  - name: echo
    container:
      image: argoproj/argosay:v2
`)
	hook := logging.NewTestHook()
	ctx := logging.WithLogger(t.Context(), logging.NewTestLogger(logging.Debug, logging.Text, hook))
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	var found *logging.TestEntry
	for _, e := range hook.AllEntries() {
		if e.Msg == "task evaluation" && e.Fields["task"] == "b" {
			found = &e
			break
		}
	}
	require.NotNil(t, found, "b's evaluation is logged while it waits on a")
	assert.Equal(t, logging.Debug, found.Level)
	assert.Equal(t, true, found.Fields["waiting"])
	assert.Equal(t, []string{"a"}, found.Fields["waitingOn"])
	assert.Equal(t, dag.ActionNone, found.Fields["action"])
}

// A finished task's part of a dependant's scope is built once per reconcile
// and then reused (Engine.finished); the reused part must be what building it
// gives, skipped marks included.
func TestBuildLocalScopeReusesFinishedTaskScope(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: scope-reuse
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - {name: A, template: out, when: "false"}
      - {name: B, template: out, depends: A}
  - name: out
    outputs:
      parameters:
      - name: p
        valueFrom:
          path: /tmp/p
    container:
      image: busybox
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	tmpl := woc.execWf.GetTemplateByName("main")
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	root, err := woc.wf.GetNodeByName(wf.Name)
	require.NoError(t, err)
	engine := NewEngine(woc, root.Name, tmplCtx, tmpl, root, root.ID, false)
	tasks := []dag.Task{&dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[0]}, &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[1]}}
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(woc.wf, tasks, tmpl, root.ID, root.Name)
	require.Equal(t, wfv1.NodeSkipped, engine.getTaskNode(ctx, "A").Phase)

	built, err := engine.buildLocalScopeFromTask(ctx, tasks[1])
	require.NoError(t, err)
	require.Contains(t, engine.finished, "A")
	reused, err := engine.buildLocalScopeFromTask(ctx, tasks[1])
	require.NoError(t, err)

	assert.Equal(t, built.scope.AsAnyMap(), reused.scope.AsAnyMap())
	assert.Equal(t, "Skipped", reused.scope.AsAnyMap()["tasks.A.status"])
	assert.True(t, reused.scope.IsSkipped("tasks.A.outputs.parameters.p"), "A's absent optional output keeps its skipped mark")
}
