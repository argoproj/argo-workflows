package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
)

// TestEngine_HookScopeErrorReentersExistingHooks: when a task's hooks' scope
// cannot be built (here an ancestor fan-out's result is not JSON, so its
// aggregate fails), the error is recorded, but the hook nodes the task already
// has are still re-entered: a suspend exit hook left alone would never finish
// and the boundary would wait on it for good.
func TestEngine_HookScopeErrorReentersExistingHooks(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/engine-hook-scope-error-reenters-existing-hooks.yaml")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 3 {
		woc.operate(ctx)
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}
	hook, err := woc.wf.GetNodeByName("scope-err.a.onExit")
	require.NoError(t, err, "a's exit hook runs")
	require.Equal(t, wfv1.NodeRunning, hook.Phase)

	item, err := woc.wf.GetNodeByName("scope-err.g(0:one)")
	require.NoError(t, err)
	notJSON := "{not json"
	item.Outputs = &wfv1.Outputs{Result: &notJSON}
	woc.wf.Status.Nodes[item.ID] = *item
	hook.StartedAt = metav1.NewTime(time.Now().Add(-time.Hour))
	woc.wf.Status.Nodes[hook.ID] = *hook
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hook, err = woc.wf.GetNodeByName("scope-err.a.onExit")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, hook.Phase, "the existing hook is re-entered")
	assert.Contains(t, woc.wf.Status.Message, "failed to aggregate outputs", "the scope error is recorded")
}

// TestEngine_C67_NoScopeBuildWithoutHooks: driving the hooks of a task
// without any builds no scope. Building one walks every ancestor, and
// the walk drives every task's hooks on every reconcile. A scope build over a
// finished ancestor records the ancestor in engine.finished, so an empty
// engine.finished shows no scope was built; the hooked task is the control.
func TestEngine_C67_NoScopeBuildWithoutHooks(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/engine-no-scope-build-without-hooks.yaml")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	tmpl := woc.execWf.GetTemplateByName("main")
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)
	root, err := woc.wf.GetNodeByName(wf.Name)
	require.NoError(t, err)
	engine := NewEngine(woc, root.Name, tmplCtx, tmpl, root.ID, false)
	var tasks []dag.Task
	for i := range tmpl.DAG.Tasks {
		tasks = append(tasks, &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[i]})
	}
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(woc.wf, tasks, tmpl, root.ID, root.Name)
	require.NotNil(t, engine.getTaskNode(ctx, "B"), "B ran")

	assert.True(t, engine.processHooks(ctx, tasks[0]))
	assert.True(t, engine.processHooks(ctx, tasks[1]))
	assert.Empty(t, engine.finished, "no scope is built for a task without hooks")

	assert.True(t, engine.processHooks(ctx, tasks[2]))
	assert.Contains(t, engine.finished, "A", "the hooked task's scope is built")
}

// --- DAG exit hook tests ---

func TestDAGExitHookOnSuccess(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-exit-hook-on-success.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasOnExit := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			hasOnExit = true
			break
		}
	}
	assert.True(t, hasOnExit, "exit handler should fire on task success")
}

func TestDAGExitHookOnFailure(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-exit-hook-on-failure.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasOnExit := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			hasOnExit = true
			break
		}
	}
	assert.True(t, hasOnExit, "exit handler should fire on task failure")
}

// --- Steps exit hook tests ---

func TestStepsExitHookOnSuccess(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/steps-exit-hook-on-success.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasOnExit := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			hasOnExit = true
			break
		}
	}
	assert.True(t, hasOnExit, "exit handler should fire on step success")
}

func TestStepsExitHookOnFailure(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/steps-exit-hook-on-failure.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasOnExit := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			hasOnExit = true
			break
		}
	}
	assert.True(t, hasOnExit, "exit handler should fire on step failure")
}

// --- Expression-based exit hook tests ---

func TestDAGExitHookExpressionTrue(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-exit-hook-expression-true.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasOnExit := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			hasOnExit = true
			break
		}
	}
	assert.True(t, hasOnExit, "exit handler with expression=true should fire")
}

func TestDAGExitHookExpressionFalse(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-exit-hook-expression-false.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	for _, node := range woc.wf.Status.Nodes {
		assert.NotContains(t, node.Name, "onExit", "exit handler with expression=false should NOT fire")
	}
}

// --- Lifecycle hook tests ---

func TestDAGLifecycleHookRunning(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-lifecycle-hook-running.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	// Transition pods to Running so the lifecycle hook expression fires
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasHook := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".hooks.running") {
			hasHook = true
			break
		}
	}
	assert.True(t, hasHook, "lifecycle hook should fire when task is Running")
}

func TestStepsLifecycleHookRunning(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/steps-lifecycle-hook-running.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	// Transition pods to Running so the lifecycle hook expression fires
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasHook := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".hooks.running") {
			hasHook = true
			break
		}
	}
	assert.True(t, hasHook, "lifecycle hook should fire when step is Running")
}

// --- Exit hook blocks completion ---

func TestDAGExitHookBlocksCompletion(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-exit-hook-blocks-completion.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Step 1: Run and succeed the main task
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// The exit handler pod was created but not yet completed.
	// The DAG/Steps parent should still be Running.
	mainNode, err := woc.wf.GetNodeByName("dag-exit-blocks")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, mainNode.Phase, "DAG should be Running while exit handler is pending")

	// Step 2: Complete the exit handler
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	mainNode, err = woc.wf.GetNodeByName("dag-exit-blocks")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, mainNode.Phase, "DAG should Succeed after exit handler completes")
}

// --- No hooks: task completes normally ---

func TestDAGNoHooks(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-no-hooks.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	mainNode, err := woc.wf.GetNodeByName("dag-no-hooks")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, mainNode.Phase)

	for _, node := range woc.wf.Status.Nodes {
		assert.NotContains(t, node.Name, "onExit", "no exit handler should fire")
	}
}

// --- Exit hook with both lifecycle hook and exit hook on same task ---

func TestDAGExitAndLifecycleHookTogether(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-exit-and-lifecycle-hook.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Step 1: Start the task
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Step 2: Transition to Running to trigger lifecycle hook
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasRunningHook := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".hooks.running") {
			hasRunningHook = true
			break
		}
	}
	assert.True(t, hasRunningHook, "lifecycle hook should fire when Running")

	// Step 3: Complete the task to trigger exit hook
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasOnExit := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			hasOnExit = true
			break
		}
	}
	assert.True(t, hasOnExit, "exit handler should fire after task completes")
}

// --- Sequential DAG tasks with exit hooks ---

func TestDAGSequentialExitHooks(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-sequential-exit-hooks.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Step 1: Start - task-a runs
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Step 2: Complete task-a → exit hook fires + task-b starts
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Step 3: Complete task-a's exit hook + task-b
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Step 4: task-b's exit hook should fire now; complete it
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	exitCount := 0
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			exitCount++
		}
	}
	assert.Equal(t, 2, exitCount, "both tasks should have exit handlers")
}

// --- Multiple parallel tasks ---

func TestDAGMultipleTasksExitHooks(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-multiple-tasks-exit-hooks.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	exitCount := 0
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			exitCount++
		}
	}
	assert.Equal(t, 2, exitCount, "both parallel tasks should have exit handlers")
}

// --- Lifecycle hook on error ---

func TestDAGLifecycleHookOnError(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-lifecycle-hook-on-error.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasHook := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".hooks.failed") {
			hasHook = true
			break
		}
	}
	assert.True(t, hasHook, "lifecycle hook should fire when task fails")
}

// --- Exit hook with output param arguments ---

func TestDAGExitHookWithOutputArgs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/dag-exit-hook-with-output-args.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	hasOnExit := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			hasOnExit = true
			break
		}
	}
	assert.True(t, hasOnExit, "exit handler with output args should fire")
}

// TestBug_HookFailureDoesNotKillSiblings verifies a 3-task DAG where task A's
// exit hook references a non-existent template: tasks B and C must still
// be scheduled and have their hooks/exit-handlers processed.
//
// Bug: the Engine's hook pass short-circuited on the first per-task error, and
// engine.Execute called markBoundaryError, killing the DAG boundary. With
// the fix, A's error is isolated to A; B's exit handler still fires.
func TestBug_HookFailureDoesNotKillSiblings(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/bug-hook-failure-does-not-kill-siblings.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Cycle 1: A, B, C scheduled (no deps).
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for _, name := range []string{"t.A", "t.B", "t.C"} {
		node, err := woc.wf.GetNodeByName(name)
		require.NoError(t, err, "sibling %s must be scheduled on cycle 1", name)
		assert.NotEqual(t, wfv1.NodeOmitted, node.Phase, "sibling %s must not be omitted", name)
	}

	// Cycle 2: pods succeed. A's exit hook attempts to resolve a non-existent
	// template, which errors. With the bug, that aborts processing of B's exit
	// hook (no node created) and prevents sibling scheduling. With the fix,
	// A's error is isolated to A's task node and B's exit handler still runs.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// The core fix: B's exit hook node must exist — A's hook error must NOT abort
	// B's exit handler processing. Naming convention: GenerateOnExitNodeName(taskNode.Name).
	bHook, err := woc.wf.GetNodeByName("t.B.onExit")
	require.NoError(t, err,
		"B's exit hook node must be created even when A's hook errors")
	require.NotNil(t, bHook)

	// Note: the DAG boundary may legitimately settle to Error here because A's
	// task node was marked Error (by the per-task onError callback) — the
	// boundary phase reflects task phases. The bug was that the failure
	// propagated as a *boundary-level* abort that prevented sibling exit
	// hooks from running at all. The B-hook assertion above is the load-bearing
	// check.
}

// TestExitHookGatesBoundaryCompletion verifies that a DAG boundary whose
// task has settled-failed stays non-terminal (Running) while that task's
// exit hook is still pending, and only transitions terminal once the hook
// completes.
//
// This matches legacy behavior: assessDAGPhase (origin/main) BFS-walks the
// boundary's descendants and returns NodeRunning whenever any descendant is
// !Fulfilled(); the onExit node is parented under the task node, so a pending
// task exit hook keeps the boundary Running. The engine reproduces this via
// the hasPendingTaskHooks gate in finalize.
//
// Marking the boundary terminal early would set the workflow's completed=true
// label, after which the controller's reconciliationNeeded filter drops
// further events — stranding the still-running exit-hook pod and never
// recording its completion. See engine_test.go (TestEngine lifecycle), which
// likewise asserts the DAG stays Running while a task exit hook is pending.
func TestExitHookGatesBoundaryCompletion(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/exit-hook-gates-boundary-completion.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Mark A as failed (its container exited).
	aNode, err := woc.wf.GetNodeByName("t.A")
	require.NoError(t, err)
	woc.markNodePhase(ctx, aNode.Name, wfv1.NodeFailed, "container exited 1")
	// Re-operate: the engine sees A failed and creates the slow exit hook.
	// The exit hook stays Running because we never transition its pod.
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// The exit hook node should exist and be non-terminal — the "stuck/slow
	// hook" condition this test exercises.
	hookNode, err := woc.wf.GetNodeByName("t.A.onExit")
	require.NoError(t, err, "exit hook node must be created on task failure")
	require.False(t, hookNode.Fulfilled(),
		"test precondition: exit hook must still be pending")

	// While the exit hook is pending, the boundary must NOT be terminal — the
	// workflow must wait for the exit handler to finish.
	boundary, err := woc.wf.GetNodeByName("t")
	require.NoError(t, err)
	require.False(t, boundary.Fulfilled(),
		"boundary must stay Running while the task exit hook is pending; got %s", boundary.Phase)

	// Complete the exit hook, then re-operate: the gate clears and the boundary
	// settles to a terminal phase reflecting the failed task.
	woc.markNodePhase(ctx, hookNode.Name, wfv1.NodeSucceeded, "exit hook done")
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	boundary, err = woc.wf.GetNodeByName("t")
	require.NoError(t, err)
	assert.True(t, boundary.Phase == wfv1.NodeFailed || boundary.Phase == wfv1.NodeError,
		"boundary must be terminal once the exit hook completes; got %s", boundary.Phase)
}

// TestBug_ExitHookNotReRunUnderParallelism is the engine regression test for
// #14392 ("do not re-run onExitNode"). The legacy fix (d81ac3f78) lived in the
// executeDAG target-task loop that this engine refactor deleted.
//
// Mechanism: engine.Execute used to call processHooks twice per operate cycle
// (a first pass for tasks done in prior cycles, a second for tasks that just
// completed); the ordered walk now drives each task's hooks once. The exit hook driver re-enters reconcileTemplate on an
// existing-but-unfulfilled onExit node, which falls through to checkParallelism
// (a Pending node does not short-circuit via handleNodeFulfilled). With
// workflow parallelism:1, the pass that creates the onExit pod bumps activePods
// 0->1; a second invocation in the same cycle then sees activePods>=1 and gets
// ErrParallelismReached. Treating that throttle as the task's exit-handler error
// triggers a spurious markNodeError on the *task* node — exactly the re-run
// #14392 prohibits.
//
// The task-corruption symptom is independently blocked by node_phase_sm.go
// (which refuses terminal->Error), so node state alone cannot distinguish the
// bug. This test therefore parses the captured logs: the fix swallows
// ErrParallelismReached in hookHandler.DriveTaskHooks (mirroring operator.go's
// workflow-level onExit handling), so the "task hook errored" Error
// entry must NOT appear. It also asserts the user-visible outcome.
func TestBug_ExitHookNotReRunUnderParallelism(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/engine_hooks/bug-exit-hook-not-re-run-under-parallelism.yaml")
	// Capture controller logs so we can assert the exit handler is not errored
	// by a spurious parallelism throttle.
	hook := logging.NewTestHook()
	logger := logging.NewTestLogger(logging.Info, logging.Text, hook)
	ctx := logging.WithLogger(t.Context(), logger)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Drive to completion, succeeding whatever pod exists each cycle. With
	// parallelism:1 at most one pod runs at a time (task A, then its onExit).
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 10 {
		woc.operate(ctx)
		if woc.wf.Status.Phase.Completed() {
			break
		}
		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}

	// Load-bearing: the exit handler must not be reported as errored. With the
	// bug, the second invocation logs this at Error with "Max parallelism reached".
	for _, e := range hook.AllEntries() {
		if e.Level == logging.Error && strings.Contains(e.Msg, "task hook errored") {
			t.Errorf("exit handler errored spuriously (re-run #14392): %s fields=%v", e.Msg, e.Fields)
		}
	}

	// And the user-visible outcome: A stays Succeeded, onExit completes, wf Succeeds.
	a, err := woc.wf.GetNodeByName("t.A")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, a.Phase, "task A must stay Succeeded")

	onExit, err := woc.wf.GetNodeByName("t.A.onExit")
	require.NoError(t, err, "onExit node must exist")
	assert.Equal(t, wfv1.NodeSucceeded, onExit.Phase, "onExit handler must complete")

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}
