package dag

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	intstrutil "github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// testCtx returns the test's context with a test logger attached.
func testCtx(t testing.TB) context.Context {
	t.Helper()
	return logging.TestContext(t.Context())
}

// --- Helper functions for creating test workflows ---

func newTestWorkflow(name string) *wfv1.Workflow {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{},
		},
	}
}

func addNodeToWorkflow(ctx context.Context, wf *wfv1.Workflow, name string, phase wfv1.NodePhase) {
	nodeID := wf.NodeID(name)
	node := wfv1.NodeStatus{
		ID:    nodeID,
		Name:  name,
		Phase: phase,
		Type:  wfv1.NodeTypePod,
	}
	wf.Status.Nodes.Set(ctx, nodeID, node)
}

func createDAGTemplate(tasks []wfv1.DAGTask) *wfv1.Template {
	return &wfv1.Template{
		Name: "dag-template",
		DAG: &wfv1.DAGTemplate{
			Tasks: tasks,
		},
	}
}

// --- Tests for workflowStore ---

func TestWorkflowStore_New(t *testing.T) {
	t.Run("creates store with workflow context", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		store := newWorkflowStore(wf, "boundary-id", "boundary-name")

		assert.NotNil(t, store)
		assert.Equal(t, "boundary-id", store.boundaryID)
		assert.Equal(t, "boundary-name", store.boundaryName)
	})
}

func TestWorkflowStore_GetState(t *testing.T) {
	t.Run("returns state from node phase", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		store := newWorkflowStore(wf, "", "dag")

		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeSucceeded)

		state := store.getPhase(t.Context(), "taskA")
		assert.Equal(t, wfv1.NodeSucceeded, state)
	})

	t.Run("returns pending for nonexistent node", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		store := newWorkflowStore(wf, "", "dag")

		state := store.getPhase(t.Context(), "nonexistent")
		assert.Equal(t, wfv1.NodePending, state)
	})
}

func TestWorkflowStore_GetNode(t *testing.T) {
	t.Run("returns node for task", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		store := newWorkflowStore(wf, "", "dag")

		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeSucceeded)

		node := store.getNode("taskA")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	})

	t.Run("returns nil for nonexistent task", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		store := newWorkflowStore(wf, "", "dag")

		node := store.getNode("nonexistent")
		assert.Nil(t, node)
	})
}

// --- Tests for WorkflowTasks ---

func toTasks(dagTasks []wfv1.DAGTask) []Task {
	tasks := make([]Task, len(dagTasks))
	for i := range dagTasks {
		tasks[i] = &DAGTask{&dagTasks[i]}
	}
	return tasks
}

func TestWorkflowTasks_NewWorkflowTasks(t *testing.T) {
	t.Run("creates tasks adapter", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
		}

		tasks := newWorkflowTasks(toTasks(dagTasks))

		assert.NotNil(t, tasks)
		assert.Len(t, tasks.TaskNames(), 2)
	})
}

func TestWorkflowTasks_GetDependencies(t *testing.T) {
	t.Run("parses depends expression", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA && taskB"},
		}
		tasks := newWorkflowTasks(toTasks(dagTasks))

		ctx := t.Context()
		deps, err := tasks.GetDependencies(ctx, "taskC")

		require.NoError(t, err)
		assert.Len(t, deps, 2)
		assert.Contains(t, deps, Key("taskA"))
		assert.Contains(t, deps, Key("taskB"))
	})

	t.Run("parses complex depends expression", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA.Succeeded && taskB.Failed"},
		}
		tasks := newWorkflowTasks(toTasks(dagTasks))

		ctx := t.Context()
		deps, err := tasks.GetDependencies(ctx, "taskC")

		require.NoError(t, err)
		assert.Contains(t, deps, Key("taskA"))
		assert.Contains(t, deps, Key("taskB"))
	})

	t.Run("handles legacy dependencies field", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Dependencies: []string{"taskA", "taskB"}},
		}
		tasks := newWorkflowTasks(toTasks(dagTasks))

		ctx := t.Context()
		deps, err := tasks.GetDependencies(ctx, "taskC")

		require.NoError(t, err)
		assert.Len(t, deps, 2)
		assert.Contains(t, deps, Key("taskA"))
		assert.Contains(t, deps, Key("taskB"))
	})

	t.Run("returns empty for task with no dependencies", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskA"},
		}
		tasks := newWorkflowTasks(toTasks(dagTasks))

		ctx := t.Context()
		deps, err := tasks.GetDependencies(ctx, "taskA")

		require.NoError(t, err)
		assert.Empty(t, deps)
	})
}

func TestWorkflowTasks_GetDependsLogic(t *testing.T) {
	t.Run("returns expanded depends expression", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
		}
		tasks := newWorkflowTasks(toTasks(dagTasks))

		ctx := t.Context()
		logic := tasks.GetDependsLogic(ctx, "taskB")

		// Should be expanded to include .Succeeded, .Skipped, .Daemoned
		assert.Contains(t, logic, normalizeTaskName("taskA")+".Succeeded")
	})

	t.Run("preserves explicit expressions", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA.Failed || taskA.Succeeded"},
		}
		tasks := newWorkflowTasks(toTasks(dagTasks))

		ctx := t.Context()
		logic := tasks.GetDependsLogic(ctx, "taskB")

		assert.Contains(t, logic, normalizeTaskName("taskA")+".Failed")
		assert.Contains(t, logic, normalizeTaskName("taskA")+".Succeeded")
	})
}

func TestWorkflowTasks_TaskNames(t *testing.T) {
	t.Run("returns sorted task names", func(t *testing.T) {
		dagTasks := []wfv1.DAGTask{
			{Name: "taskC"},
			{Name: "taskA"},
			{Name: "taskB"},
		}
		tasks := newWorkflowTasks(toTasks(dagTasks))

		names := tasks.TaskNames()

		assert.Equal(t, []string{"taskA", "taskB", "taskC"}, names)
	})
}

// --- Tests for DAGEvaluator ---

func TestDAGEvaluator_NewDAGEvaluator(t *testing.T) {
	t.Run("creates evaluator", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
		})

		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		assert.NotNil(t, evaluator)
		// Verify it can evaluate (internals are properly initialized)
		ctx := t.Context()
		result := evaluator.EvaluateTask(ctx, "taskA")
		assert.Equal(t, "taskA", result.TaskName)
	})
}

func TestDAGEvaluator_EvaluateTask(t *testing.T) {
	t.Run("pending task with no dependencies should run", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := logging.TestContext(t.Context())
		result := evaluator.EvaluateTask(ctx, "taskA")

		assert.Equal(t, "taskA", result.TaskName)
		assert.True(t, result.ShouldRun)
		assert.False(t, result.Suspended)
		assert.NoError(t, result.Error)
	})

	t.Run("succeeded task should not run", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeSucceeded)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := logging.TestContext(t.Context())
		result := evaluator.EvaluateTask(ctx, "taskA")

		assert.False(t, result.ShouldRun)
	})

	t.Run("running_task_should_continue_running", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeRunning)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := logging.TestContext(t.Context())
		result := evaluator.EvaluateTask(ctx, "taskA")

		assert.True(t, result.ShouldRun)
	})

	t.Run("task with unfulfilled dependencies is suspended", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := logging.TestContext(t.Context())
		result := evaluator.EvaluateTask(ctx, "taskB")
		assert.False(t, result.ShouldRun)
		assert.True(t, result.Suspended)
		assert.Contains(t, result.WaitingOn, "taskA")
	})

	t.Run("task with fulfilled dependencies should run", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeSucceeded)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := logging.TestContext(t.Context())
		result := evaluator.EvaluateTask(ctx, "taskB")

		assert.True(t, result.ShouldRun)
		assert.False(t, result.Suspended)
	})

	t.Run("task omitted when depends condition not met", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeFailed)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA.Succeeded"}, // taskA failed, not succeeded
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := logging.TestContext(t.Context())
		result := evaluator.EvaluateTask(ctx, "taskB")

		assert.False(t, result.ShouldRun)
		assert.True(t, result.Skipped)
	})
}

func TestDAGEvaluator_DiamondDAG(t *testing.T) {
	t.Run("evaluates diamond DAG", func(t *testing.T) {
		//     A
		//    / \
		//   B   C
		//    \ /
		//     D
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "A"},
			{Name: "B", Depends: "A"},
			{Name: "C", Depends: "A"},
			{Name: "D", Depends: "B && C"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()

		// Initially, only A should be ready to run
		result := evaluator.EvaluateTask(ctx, "A")
		assert.True(t, result.ShouldRun)

		result = evaluator.EvaluateTask(ctx, "B")
		assert.True(t, result.Suspended)

		result = evaluator.EvaluateTask(ctx, "C")
		assert.True(t, result.Suspended)

		result = evaluator.EvaluateTask(ctx, "D")
		assert.True(t, result.Suspended)

		// After A succeeds
		addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeSucceeded)
		evaluator = NewDAGEvaluator(wf, tmpl, "", "dag")

		result = evaluator.EvaluateTask(ctx, "B")
		assert.True(t, result.ShouldRun)

		result = evaluator.EvaluateTask(ctx, "C")
		assert.True(t, result.ShouldRun)

		result = evaluator.EvaluateTask(ctx, "D")
		assert.True(t, result.Suspended)

		// After B and C succeed
		addNodeToWorkflow(testCtx(t), wf, "dag.B", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(t), wf, "dag.C", wfv1.NodeSucceeded)
		evaluator = NewDAGEvaluator(wf, tmpl, "", "dag")

		result = evaluator.EvaluateTask(ctx, "D")
		assert.True(t, result.ShouldRun)
	})
}

func TestDAGEvaluator_FindLeafTaskNames(t *testing.T) {
	t.Run("finds leaf tasks", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
			{Name: "taskC", Depends: "taskA"},
			{Name: "taskD", Depends: "taskB && taskC"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		leafTasks := evaluator.FindLeafTaskNames(ctx)

		assert.Len(t, leafTasks, 1)
		assert.Equal(t, "taskD", leafTasks[0])
	})

	t.Run("multiple leaf tasks", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
			{Name: "taskC", Depends: "taskA"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		leafTasks := evaluator.FindLeafTaskNames(ctx)

		assert.Len(t, leafTasks, 2)
		assert.Contains(t, leafTasks, "taskB")
		assert.Contains(t, leafTasks, "taskC")
	})

	t.Run("all tasks are leaves when no dependencies", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		leafTasks := evaluator.FindLeafTaskNames(ctx)

		assert.Len(t, leafTasks, 3)
	})
}

func TestDAGEvaluator_GetTargetTasks(t *testing.T) {
	t.Run("returns explicit targets", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC"},
		})
		tmpl.DAG.Target = "taskA taskB"
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		targets := evaluator.GetTargetTasks(ctx)

		assert.Equal(t, []string{"taskA", "taskB"}, targets)
	})

	t.Run("returns leaf tasks when no explicit targets", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		targets := evaluator.GetTargetTasks(ctx)

		assert.Equal(t, []string{"taskB"}, targets)
	})
}

func TestDAGEvaluator_EvaluateAll(t *testing.T) {
	t.Run("evaluates all tasks", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB", Depends: "taskA"},
			{Name: "taskC"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		results := evaluator.EvaluateAll(ctx)

		assert.Len(t, results, 3)
		assert.Contains(t, results, "taskA")
		assert.Contains(t, results, "taskB")
		assert.Contains(t, results, "taskC")
	})
}

// --- Tests for depends expression evaluation ---

func TestDAGEvaluator_ComplexDependsExpressions(t *testing.T) {
	t.Run("OR expression with one succeeded", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(t), wf, "dag.taskB", wfv1.NodeFailed)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA.Succeeded || taskB.Succeeded"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		result := evaluator.EvaluateTask(ctx, "taskC")

		assert.True(t, result.ShouldRun)
	})

	t.Run("AND expression with both conditions met", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(t), wf, "dag.taskB", wfv1.NodeFailed)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA.Succeeded && taskB.Failed"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		result := evaluator.EvaluateTask(ctx, "taskC")

		assert.True(t, result.ShouldRun)
	})

	t.Run("AND expression with one condition not met", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.taskA", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(t), wf, "dag.taskB", wfv1.NodeSucceeded)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA.Succeeded && taskB.Failed"}, // B didn't fail
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := t.Context()
		result := evaluator.EvaluateTask(ctx, "taskC")

		assert.False(t, result.ShouldRun)
		assert.True(t, result.Skipped)
	})
}

// --- Tests for unreachable task evaluation ---

func TestDAGEvaluator_UnreachableTask(t *testing.T) {
	// A fails, B depends on A.Succeeded → B should be Skipped
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Succeeded"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)
	result := evaluator.EvaluateTask(ctx, "B")

	assert.False(t, result.ShouldRun, "B should not run since A failed")
	assert.True(t, result.Skipped, "B should be skipped since A.Succeeded can never be true")
	assert.False(t, result.Suspended, "B should not be suspended")
}

func TestDAGEvaluator_CascadingOmission(t *testing.T) {
	// A fails, B depends on A.Succeeded, C depends on B
	// B is marked Omitted, C sees B as Omitted and is also Skipped
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Succeeded"},
		{Name: "C", Depends: "B"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)

	resultB := evaluator.EvaluateTask(ctx, "B")
	assert.True(t, resultB.Skipped, "B should be skipped")
	assert.False(t, resultB.ShouldRun, "B should not run")

	resultC := evaluator.EvaluateTask(ctx, "C")
	assert.True(t, resultC.Skipped, "C should be skipped (B is Omitted, cascading)")
	assert.False(t, resultC.Suspended, "C should not be suspended")
}

func TestDAGEvaluator_EnhancedDependsAfterFailure(t *testing.T) {
	// A fails, B depends on A.Failed → B should run
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Failed"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)
	result := evaluator.EvaluateTask(ctx, "B")

	assert.True(t, result.ShouldRun, "B should run since A.Failed is true")
	assert.False(t, result.Suspended, "B should not be suspended")
	assert.False(t, result.Skipped, "B should not be skipped")
}

func TestDAGEvaluator_MixedReachability(t *testing.T) {
	// Diamond: A(failed), B depends on A.Succeeded (unreachable),
	// C depends on A.Failed (reachable), D depends on B && C
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Succeeded"},
		{Name: "C", Depends: "A.Failed"},
		{Name: "D", Depends: "B && C"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)

	resultB := evaluator.EvaluateTask(ctx, "B")
	assert.True(t, resultB.Skipped, "B should be skipped (A.Succeeded is false)")
	assert.False(t, resultB.ShouldRun, "B should not run")

	resultC := evaluator.EvaluateTask(ctx, "C")
	assert.True(t, resultC.ShouldRun, "C should run (A.Failed is true)")
	assert.False(t, resultC.Skipped, "C should not be skipped")

	// D waits for C even though B && C can no longer be true: a depends
	// expression is only evaluated once every task it references has finished.
	resultD := evaluator.EvaluateTask(ctx, "D")
	assert.False(t, resultD.Skipped, "D waits for C before it is omitted")
	assert.True(t, resultD.Suspended, "D is waiting")
	assert.Contains(t, resultD.WaitingOn, "C")

	addNodeToWorkflow(ctx, wf, "dag.C", wfv1.NodeSucceeded)
	resultD = NewDAGEvaluator(wf, tmpl, "", "dag").EvaluateTask(ctx, "D")
	assert.True(t, resultD.Skipped, "D is omitted once C has finished (B is omitted, so B && C is false)")
	assert.False(t, resultD.Suspended)
}

func TestWorkflowStore_SetStateAndGetState(t *testing.T) {
	wf := newTestWorkflow("test-wf")
	store := newWorkflowStore(wf, "", "dag")
	ctx := testCtx(t)

	t.Run("SetState is now reflected by GetState", func(t *testing.T) {
		store.setPhase(ctx, "taskX", wfv1.NodeOmitted)

		state := store.getPhase(ctx, "taskX")
		assert.Equal(t, wfv1.NodeOmitted, state, "GetState should return Omitted from internal map")
	})

	t.Run("GetState reads from workflow nodes", func(t *testing.T) {
		addNodeToWorkflow(ctx, wf, "dag.taskY", wfv1.NodeSucceeded)
		state := store.getPhase(ctx, "taskY")
		assert.Equal(t, wfv1.NodeSucceeded, state, "GetState should read from workflow nodes")
	})
}

func TestWorkflowStore_GetStateWithDaemonedNode(t *testing.T) {
	wf := newTestWorkflow("test-wf")
	store := newWorkflowStore(wf, "", "dag")
	ctx := testCtx(t)

	// Create a daemoned running node
	nodeID := wf.NodeID("dag.daemon-task")
	daemoned := true
	node := wfv1.NodeStatus{
		ID:       nodeID,
		Name:     "dag.daemon-task",
		Phase:    wfv1.NodeRunning,
		Daemoned: &daemoned,
		Type:     wfv1.NodeTypePod,
	}
	wf.Status.Nodes.Set(ctx, nodeID, node)

	state := store.getPhase(ctx, "daemon-task")
	assert.Equal(t, wfv1.NodeSucceeded, state, "Daemoned running node should return Succeeded")
}

func TestDAGEvaluator_DaemonedCompletedNode(t *testing.T) {
	wf := newTestWorkflow("test-wf")
	ctx := testCtx(t)

	nodeID := wf.NodeID("dag.A")
	daemoned := true
	node := wfv1.NodeStatus{
		ID:       nodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeSucceeded,
		Daemoned: &daemoned,
		Type:     wfv1.NodeTypePod,
	}
	wf.Status.Nodes.Set(ctx, nodeID, node)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Daemoned"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := evaluator.EvaluateTask(ctx, "B")
	assert.True(t, result.ShouldRun, "B should run because A is daemoned and non-pending")
}

func TestDAGEvaluator_DaemonedFailedNode(t *testing.T) {
	wf := newTestWorkflow("test-wf")
	ctx := testCtx(t)

	nodeID := wf.NodeID("dag.A")
	daemoned := true
	node := wfv1.NodeStatus{
		ID:       nodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeFailed,
		Daemoned: &daemoned,
		Type:     wfv1.NodeTypePod,
	}
	wf.Status.Nodes.Set(ctx, nodeID, node)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := evaluator.EvaluateTask(ctx, "B")
	assert.True(t, result.ShouldRun, "B should run because A is daemoned (Failed but non-Pending)")
}

func TestResolveTaskDepends_StepNames(t *testing.T) {
	// Steps tasks carry synthetic legacy dependencies named "[groupIndex].stepName".
	// They are structured data, expanded directly and never parsed as an
	// expression, so the ".A" suffix is never mistaken for a result qualifier.
	stepA := wfv1.DAGTask{Name: "[0].A"}
	stepB := wfv1.DAGTask{Name: "[1].B", Dependencies: []string{"[0].A"}}
	provider := func(name string) Task {
		if name == "[0].A" {
			return &DAGTask{DAGTask: &stepA}
		}
		return nil
	}
	deps, logic, err := resolveTaskDepends(&DAGTask{DAGTask: &stepB}, provider)
	require.NoError(t, err)
	assert.Equal(t, []string{"[0].A"}, deps)
	assert.Equal(t, common.ExpandDependency("[0].A", nil, normalizeTaskName), logic)
}

func TestResolveTaskDepends_UsesValidationGrammar(t *testing.T) {
	// A user-written depends is tokenized by common.ParseDepends, the same
	// grammar validation applies, then task names are hex-normalized.
	taskA := wfv1.DAGTask{Name: "A"}
	taskC := wfv1.DAGTask{Name: "C", Depends: "A.Failed || B"}
	provider := func(name string) Task {
		if name == "A" {
			return &DAGTask{DAGTask: &taskA}
		}
		return nil
	}
	deps, logic, err := resolveTaskDepends(&DAGTask{DAGTask: &taskC}, provider)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, deps)
	assert.Equal(t, normalizeTaskName("A")+".Failed || "+common.ExpandDependency("B", nil, normalizeTaskName), logic)

	taskD := wfv1.DAGTask{Name: "D", Depends: "A.Bogus"}
	_, _, err = resolveTaskDepends(&DAGTask{DAGTask: &taskD}, provider)
	assert.EqualError(t, err, "task result 'Bogus' for task 'A' is invalid")
}

// The depends result qualifiers users may write are defined once, in
// workflow/common; the evaluator's taskResult scope struct must expose
// exactly that set so evaluation cannot drift from validation.
func TestTaskResultFieldsMatchDependsVocabulary(t *testing.T) {
	want := []string{
		string(common.TaskResultSucceeded), string(common.TaskResultFailed), string(common.TaskResultErrored),
		string(common.TaskResultSkipped), string(common.TaskResultOmitted), string(common.TaskResultDaemoned),
		string(common.TaskResultAnySucceeded), string(common.TaskResultAllFailed),
	}
	var got []string
	for field := range reflect.TypeFor[taskResult]().Fields() {
		got = append(got, field.Name)
	}
	assert.ElementsMatch(t, want, got)
}

func TestNormalizeTaskName_HexLikeNames(t *testing.T) {
	normalized := normalizeTaskName("t0a1b2c")
	assert.NotEqual(t, "t0a1b2c", normalized, "task name starting with 't' followed by valid hex should still be normalized")

	a := normalizeTaskName("t0a1b2c")
	b := normalizeTaskName("t0a1b2d")
	assert.NotEqual(t, a, b, "different task names must not collide")
}

func TestDAGEvaluator_LegacyDependencies(t *testing.T) {
	// Task B uses legacy "dependencies: [A]" instead of "depends: A"
	// Both should produce equivalent evaluation results.
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeSucceeded)

	tmplWithDepends := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A"},
	})

	tmplWithDependencies := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Dependencies: []string{"A"}},
	})

	ctx := testCtx(t)

	evalDepends := NewDAGEvaluator(wf, tmplWithDepends, "", "dag")
	resultDepends := evalDepends.EvaluateTask(ctx, "B")

	evalDeps := NewDAGEvaluator(wf, tmplWithDependencies, "", "dag")
	resultDeps := evalDeps.EvaluateTask(ctx, "B")

	assert.True(t, resultDepends.ShouldRun, "B should run with depends field")
	assert.True(t, resultDeps.ShouldRun, "B should run with legacy dependencies field")
	assert.NoError(t, resultDeps.Error, "legacy dependencies should not produce eval errors")
}

// TestDAGEvaluator_BrokenDependsExpression verifies that a malformed depends
// expression surfaces an error rather than silently omitting the task.
// Bug: evaluateAllStates discards the error from isReady (line 264) and
// marks the task as Omitted, causing the user to see "depends condition not met"
// when the real problem is a broken expression.
func TestDAGEvaluator_BrokenDependsExpression(t *testing.T) {
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeSucceeded)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.InvalidStatus"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)
	result := evaluator.EvaluateTask(ctx, "B")

	// B's depends expression references "A.InvalidStatus" which is not a valid
	// status field. This should surface as an error, NOT silently omit B.
	assert.Error(t, result.Error,
		"broken depends expression should produce an error, not silently omit the task")
}

// --- Tests for evaluateRetryNode ---

func TestEvaluateRetryNode_ChildRunning(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	childNodeID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(testCtx(t), childNodeID, wfv1.NodeStatus{
		ID:    childNodeID,
		Name:  "dag.A(0)",
		Phase: wfv1.NodeRunning,
		Type:  wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{childNodeID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)
	result := eval.EvaluateTask(ctx, "A")

	assert.Equal(t, ActionNone, result.Action, "should wait for running child")
	assert.False(t, result.FulfilledForDeps, "running child is not fulfilled for deps")
	assert.False(t, result.ShouldRun, "should not schedule new work while child is running")
}

func TestEvaluateRetryNode_ChildDaemoned(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	childNodeID := wf.NodeID("dag.A(0)")
	daemoned := true

	wf.Status.Nodes.Set(testCtx(t), childNodeID, wfv1.NodeStatus{
		ID:       childNodeID,
		Name:     "dag.A(0)",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypePod,
		Daemoned: &daemoned,
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{childNodeID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)
	result := eval.EvaluateTask(ctx, "A")

	assert.Equal(t, ActionNone, result.Action, "daemoned child needs no action")
	assert.True(t, result.FulfilledForDeps, "daemoned child should be fulfilled for deps")
}

func TestEvaluateRetryNode_ChildFailed_WithinLimit(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	childNodeID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(testCtx(t), childNodeID, wfv1.NodeStatus{
		ID:       childNodeID,
		Name:     "dag.A(0)",
		Phase:    wfv1.NodeFailed,
		Type:     wfv1.NodeTypePod,
		NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{childNodeID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2")})

	ctx := testCtx(t)
	result := eval.EvaluateTask(ctx, "A")

	assert.Equal(t, ActionExecute, result.Action, "should schedule retry within limit")
	assert.True(t, result.ShouldRun, "should be marked as should run")
}

func TestEvaluateRetryNode_ChildFailed_Exhausted(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	child0ID := wf.NodeID("dag.A(0)")
	child1ID := wf.NodeID("dag.A(1)")
	child2ID := wf.NodeID("dag.A(2)")

	wf.Status.Nodes.Set(testCtx(t), child0ID, wfv1.NodeStatus{
		ID: child0ID, Name: "dag.A(0)", Phase: wfv1.NodeFailed,
		Type: wfv1.NodeTypePod, NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(t), child1ID, wfv1.NodeStatus{
		ID: child1ID, Name: "dag.A(1)", Phase: wfv1.NodeFailed,
		Type: wfv1.NodeTypePod, NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(t), child2ID, wfv1.NodeStatus{
		ID: child2ID, Name: "dag.A(2)", Phase: wfv1.NodeFailed,
		Type: wfv1.NodeTypePod, NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{child0ID, child1ID, child2ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2")})

	ctx := testCtx(t)
	result := eval.EvaluateTask(ctx, "A")

	assert.Equal(t, ActionFail, result.Action, "should fail when retry limit exhausted")
	assert.False(t, result.ShouldRun, "should not run when limit exhausted")
	assert.Contains(t, result.ActionReason, "retry limit exhausted")
}

func TestEvaluateRetryNode_ChildSucceeded(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	childNodeID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(testCtx(t), childNodeID, wfv1.NodeStatus{
		ID:    childNodeID,
		Name:  "dag.A(0)",
		Phase: wfv1.NodeSucceeded,
		Type:  wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{childNodeID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx(t)
	result := eval.EvaluateTask(ctx, "A")

	assert.Equal(t, ActionSucceed, result.Action, "should succeed when child succeeded")
}

func TestEvaluateRetryNode_DaemonChildFailed_Retries(t *testing.T) {
	// A daemon pod that failed (Daemoned=nil, Phase=Failed) should be retried.
	// This simulates a daemon pod that crashed before becoming daemoned.
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	childNodeID := wf.NodeID("dag.A(0)")

	// Daemoned is nil (pod crashed before becoming a daemon), Phase is Failed.
	wf.Status.Nodes.Set(testCtx(t), childNodeID, wfv1.NodeStatus{
		ID:       childNodeID,
		Name:     "dag.A(0)",
		Phase:    wfv1.NodeFailed,
		Type:     wfv1.NodeTypePod,
		NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{childNodeID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3")})

	ctx := testCtx(t)
	result := eval.EvaluateTask(ctx, "A")

	assert.Equal(t, ActionExecute, result.Action, "failed daemon pod should be retried")
	assert.True(t, result.ShouldRun, "should schedule a retry for failed daemon pod")
}

func TestDependsReadiness_RetryDaemonFulfillsDeps(t *testing.T) {
	// Task B depends on task A. A is a retry node with a daemoned child.
	// B should be ready because A's daemon is running.
	wf := &wfv1.Workflow{}
	wf.Name = "test"
	wf.Status.Nodes = wfv1.Nodes{}
	retryNodeID := wf.NodeID("test.A")
	childNodeID := wf.NodeID("test.A(0)")
	daemon := true
	wf.Status.Nodes[retryNodeID] = wfv1.NodeStatus{
		ID: retryNodeID, Name: "test.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childNodeID},
	}
	wf.Status.Nodes[childNodeID] = wfv1.NodeStatus{
		ID: childNodeID, Name: "test.A(0)", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypePod, Daemoned: &daemon,
	}

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{
			{Name: "A", Template: "daemon-tmpl"},
			{Name: "B", Template: "echo", Depends: "A"},
		},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "test", "test")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2")})

	ctx := t.Context()
	result := eval.EvaluateTask(ctx, "B")
	assert.True(t, result.ShouldRun, "B should be ready when A's retry daemon child is running")
}

// TestR3S_DaemonedNodeIncorrectlyReEvaluates
// Bug: Line 365 in argo.go uses !node.Phase.Fulfilled(node.TaskResultSynced)
// instead of !node.Fulfilled(). For a daemoned running node:
// - node.Phase.Fulfilled(node.TaskResultSynced) returns false (Running is not completed)
// - node.Fulfilled() returns true (due to IsDaemoned() && Phase != Pending check)
//
// This causes the code to unnecessarily re-evaluate the depends logic for a
// daemoned node that should be considered fulfilled.
func TestEval_DaemonedRunningNodeNotReEvaluated(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Name = "test"
	wf.Status.Nodes = wfv1.Nodes{}

	// Create a daemoned running task node
	taskID := wf.NodeID("test.daemoned-task")
	daemonedVal := true
	syncedVal := true
	daemonedNode := wfv1.NodeStatus{
		ID:               taskID,
		Name:             "test.daemoned-task",
		Phase:            wfv1.NodeRunning,
		Type:             wfv1.NodeTypePod,
		Daemoned:         &daemonedVal,
		TaskResultSynced: &syncedVal,
	}
	wf.Status.Nodes[taskID] = daemonedNode

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{
			{Name: "daemoned-task", Template: "t"},
		},
	}}

	eval := NewDAGEvaluator(wf, tmpl, "test", "test")
	ctx := t.Context()

	// Verify the node is actually daemoned and running
	node := eval.store.getNode("daemoned-task")
	require.NotNil(t, node)
	assert.True(t, node.IsDaemoned())
	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// With the bug: Line 365 uses !node.Phase.Fulfilled() which is true for Running,
	// so it would evaluate depends and possibly set ShouldRun=true.
	// The daemoned node should NOT need re-evaluation since it's fulfilled for deps.
	result := eval.EvaluateTask(ctx, "daemoned-task")
	assert.False(t, result.ShouldRun,
		"BUG: daemoned running node incorrectly re-evaluates depends - "+
			"node.Phase.Fulfilled()=false but node.Fulfilled()=true for daemoned+Running")
}

// TestEval_PendingNodeDependsBecameFalseStillShouldRun covers C22 (task
// 1.3): once a task's node exists and is unfinished, it keeps being
// dispatched (ShouldRun) even if its depends expression has since turned
// false, whatever its Phase — not only Running (main's evaluateDependsLogic
// rule; a re-evaluation regression would only re-check depends for a
// Pending node like this one, not for a Running one).
func TestEval_PendingNodeDependsBecameFalseStillShouldRun(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	aID := wf.NodeID("dag.A")
	bID := wf.NodeID("dag.B")

	// A has since failed, so B's default depends expression
	// ("A.Succeeded || A.Skipped || A.Daemoned") is now false.
	wf.Status.Nodes.Set(ctx, aID, wfv1.NodeStatus{
		ID: aID, Name: "dag.A", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	// B's own node already exists and is unfinished (Pending).
	wf.Status.Nodes.Set(ctx, bID, wfv1.NodeStatus{
		ID: bID, Name: "dag.B", Phase: wfv1.NodePending, Type: wfv1.NodeTypePod,
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{
			{Name: "A", Template: "t"},
			{Name: "B", Template: "t", Dependencies: []string{"A"}},
		},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "B")
	assert.True(t, result.ShouldRun,
		"B's node already exists and is unfinished (Pending); it must keep being dispatched even though A failed and B's depends expression is now false")
}

// ============================================================
// Retry edge cases — TestEval_Retry_*
// ============================================================

// 1. Retry limit zero — limit=0, one child Failed → ActionFail (0 retries allowed)
func TestEval_Retry_LimitZero(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("0")})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action, "limit=0: one child is already 1 attempt > 0 retries allowed")
}

// 2. Retry nil limit — limit=nil (no limit set), child Failed → ActionExecute (unlimited retries)
func TestEval_Retry_NilLimit(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	// RetryStrategy with no Limit field (nil) means unlimited
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: nil})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "nil limit = unlimited retries → should retry")
}

// 3. Retry all children are hooks — only hook children → ActionExecute (treated as no children)
func TestEval_Retry_AllChildrenAreHooks(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	hookChildID := wf.NodeID("dag.A.hook")

	wf.Status.Nodes.Set(ctx, hookChildID, wfv1.NodeStatus{
		ID: hookChildID, Name: "dag.A.hook", Phase: wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypePod,
		NodeFlag: &wfv1.NodeFlag{Hooked: true},
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{hookChildID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "all hook children → treated as no real children → first attempt needed")
}

// 4. Retry OnError policy with Failed child → ActionFail (not retried)
func TestEval_Retry_OnErrorPolicy_FailedChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("5"),
		RetryPolicy: wfv1.RetryPolicyOnError,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action, "OnError policy should not retry a Failed child")
}

// 5. Retry OnError policy with Error child → ActionExecute (retried)
func TestEval_Retry_OnErrorPolicy_ErrorChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeError, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("5"),
		RetryPolicy: wfv1.RetryPolicyOnError,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "OnError policy should retry an Error child")
}

// 6. Retry Always policy with Failed child → ActionExecute
func TestEval_Retry_AlwaysPolicy_FailedChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("5"),
		RetryPolicy: wfv1.RetryPolicyAlways,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "Always policy should retry a Failed child")
}

// 7. Retry Always policy with Error child → ActionExecute
func TestEval_Retry_AlwaysPolicy_ErrorChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeError, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("5"),
		RetryPolicy: wfv1.RetryPolicyAlways,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "Always policy should retry an Error child")
}

// 8. Retry OnTransientError with Failed → ActionExecute
func TestEval_Retry_OnTransientError_FailedChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("5"),
		RetryPolicy: wfv1.RetryPolicyOnTransientError,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "OnTransientError policy should retry a Failed child")
}

// 9. Retry OnTransientError with Error → ActionExecute
func TestEval_Retry_OnTransientError_ErrorChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeError, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("5"),
		RetryPolicy: wfv1.RetryPolicyOnTransientError,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "OnTransientError policy should retry an Error child")
}

// 10. Retry succeeded sets FulfilledForDeps — child Succeeded → ActionSucceed + FulfilledForDeps=true
func TestEval_Retry_SucceededSetsFulfilledForDeps(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionSucceed, result.Action)
	assert.True(t, result.FulfilledForDeps, "succeeded retry node should be fulfilled for deps")
}

// 11. Retry daemon child sets CurrentPhase=Succeeded — daemoned running child → CurrentPhase=Succeeded + FulfilledForDeps=true
func TestEval_Retry_DaemonChildSetsCurrentPhaseSucceeded(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")
	daemoned := true

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeRunning,
		Type:     wfv1.NodeTypePod,
		Daemoned: &daemoned,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, wfv1.NodeSucceeded, result.CurrentPhase, "daemoned child should set CurrentPhase=Succeeded")
	assert.True(t, result.FulfilledForDeps, "daemoned running child should be fulfilled for deps")
}

// 12. Dead daemon triggers retry — child Daemoned=true + Phase=Failed → ActionExecute (phase guard works)
func TestEval_Retry_DeadDaemonTriggersRetry(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")
	daemoned := true

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeFailed,
		Type:     wfv1.NodeTypePod,
		Daemoned: &daemoned,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3")})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "dead daemon (Daemoned=true + Failed) should trigger retry")
}

// 13. Retry Skipped child with Always policy → ActionExecute (retries)
func TestEval_Retry_SkippedChild_AlwaysPolicy(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeSkipped, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("3"),
		RetryPolicy: wfv1.RetryPolicyAlways,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "Always policy should retry a Skipped child")
}

// 14. Retry Skipped child with default policy → ActionFail
func TestEval_Retry_SkippedChild_DefaultPolicy(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeSkipped, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	// Default policy (OnFailure) — no RetryStrategy with explicit policy
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3")})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action, "default policy should not retry a Skipped child")
}

// 15. Retry Omitted child → ActionFail
func TestEval_Retry_OmittedChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)", Phase: wfv1.NodeOmitted, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3")})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action, "Omitted child without Always policy should result in ActionFail")
}

// 16. Retry exhausted sets FulfilledForDeps — 3 children all Failed, limit=2 → ActionFail + FulfilledForDeps=true
func TestEval_Retry_ExhaustedSetsFulfilledForDeps(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	child0ID := wf.NodeID("dag.A(0)")
	child1ID := wf.NodeID("dag.A(1)")
	child2ID := wf.NodeID("dag.A(2)")

	for _, id := range []string{child0ID, child1ID, child2ID} {
		wf.Status.Nodes.Set(ctx, id, wfv1.NodeStatus{
			ID: id, Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
		})
	}
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{child0ID, child1ID, child2ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2")})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action)
	assert.True(t, result.FulfilledForDeps, "exhausted retry node should be fulfilled for deps")
}

// 17. Retry exhausted propagates child phase — child=NodeError, limit exhausted → CurrentPhase=NodeError
func TestEval_Retry_ExhaustedPropagatesChildPhase(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	child0ID := wf.NodeID("dag.A(0)")
	child1ID := wf.NodeID("dag.A(1)")

	wf.Status.Nodes.Set(ctx, child0ID, wfv1.NodeStatus{
		ID: child0ID, Phase: wfv1.NodeError, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, child1ID, wfv1.NodeStatus{
		ID: child1ID, Phase: wfv1.NodeError, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{child0ID, child1ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit:       intstrutil.ParsePtr("1"),
		RetryPolicy: wfv1.RetryPolicyOnError,
	})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action)
	assert.Equal(t, wfv1.NodeError, result.CurrentPhase, "exhausted retry with Error child should propagate NodeError")
}

// 18. Retry at exact limit boundary — limit=2, 2 children Failed → ActionExecute (2 <= 2). 3 children Failed → ActionFail
func TestEval_Retry_ExactLimitBoundary(t *testing.T) {
	makeEval := func(numChildren int) EvaluationResult {
		wf := newTestWorkflow("test")
		ctx := testCtx(t)

		retryNodeID := wf.NodeID("dag.A")
		childIDs := make([]string, numChildren)
		for i := range numChildren {
			id := wf.NodeID(fmt.Sprintf("dag.A(%d)", i))
			childIDs[i] = id
			wf.Status.Nodes.Set(ctx, id, wfv1.NodeStatus{
				ID: id, Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
			})
		}
		wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
			ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
			Type: wfv1.NodeTypeRetry, Children: childIDs,
		})

		tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
			Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
		}}
		eval := NewDAGEvaluator(wf, tmpl, "", "dag")
		eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2")})
		return eval.EvaluateTask(ctx, "A")
	}

	result2 := makeEval(2)
	assert.Equal(t, ActionExecute, result2.Action, "2 children with limit=2 → should still retry (2 <= 2)")

	result3 := makeEval(3)
	assert.Equal(t, ActionFail, result3.Action, "3 children with limit=2 → exhausted (3 > 2)")
}

// 19. Retry fallback child lookup — store lookup fails but node.Children has valid IDs → children resolved via fallback
func TestEval_Retry_FallbackChildLookup(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	// Use a boundary that won't match the store's naming convention
	// so getRetryChildren returns nil, triggering the fallback path.
	retryNodeID := wf.NodeID("dag.A")
	// Use a real child ID but store the node directly (not via store naming)
	childID := wf.NodeID("dag.A(0)-custom")

	wf.Status.Nodes.Set(ctx, childID, wfv1.NodeStatus{
		ID: childID, Name: "dag.A(0)-custom", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	// The retry node has the child in its Children list, but the node key
	// doesn't match what getRetryChildren looks up by task name convention.
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeRetry, Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3")})

	result := eval.EvaluateTask(ctx, "A")
	// The fallback should find the child via node.Children and return ActionExecute
	assert.Equal(t, ActionExecute, result.Action, "fallback child lookup should find the child and allow retry")
}

// 20. Retry hook children don't count toward limit — 1 real child + 1 hook child, limit=1 → ActionExecute
func TestEval_Retry_HookChildrenDontCountTowardLimit(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx(t)

	retryNodeID := wf.NodeID("dag.A")
	realChildID := wf.NodeID("dag.A(0)")
	hookChildID := wf.NodeID("dag.A.hook")

	wf.Status.Nodes.Set(ctx, realChildID, wfv1.NodeStatus{
		ID: realChildID, Name: "dag.A(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, hookChildID, wfv1.NodeStatus{
		ID: hookChildID, Name: "dag.A.hook", Phase: wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypePod,
		NodeFlag: &wfv1.NodeFlag{Hooked: true},
	})
	wf.Status.Nodes.Set(ctx, retryNodeID, wfv1.NodeStatus{
		ID: retryNodeID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{realChildID, hookChildID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("1")})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionExecute, result.Action, "hook children should not count toward limit: 1 real child <= limit=1 → should retry")
}

// ============================================================
// TaskGroup edge cases — TestEval_TaskGroup_*
// ============================================================

// 21. TaskGroup all children succeeded → ActionSucceed + CurrentPhase=Succeeded
// --- TaskGroup test helpers ---

// addTaskGroupChild adds a Pod child node under a TaskGroup parent. The
// parent's Children list is updated.
func addTaskGroupChild(t testing.TB, wf *wfv1.Workflow, parent *wfv1.NodeStatus, name string, phase wfv1.NodePhase, flag *wfv1.NodeFlag) {
	childID := wf.NodeID(name)
	child := wfv1.NodeStatus{
		ID:       childID,
		Name:     name,
		Phase:    phase,
		Type:     wfv1.NodeTypePod,
		NodeFlag: flag,
	}
	wf.Status.Nodes.Set(testCtx(t), childID, child)
	parent.Children = append(parent.Children, childID)
	wf.Status.Nodes.Set(testCtx(t), parent.ID, *parent)
}

// addTaskGroupParent creates a TaskGroup parent node (Running) for a withSequence/withItems/withParam task.
func addTaskGroupParent(t testing.TB, wf *wfv1.Workflow, name string) *wfv1.NodeStatus {
	id := wf.NodeID(name)
	parent := wfv1.NodeStatus{
		ID:    id,
		Name:  name,
		Phase: wfv1.NodeRunning,
		Type:  wfv1.NodeTypeTaskGroup,
	}
	wf.Status.Nodes.Set(testCtx(t), id, parent)
	return &parent
}

// withSequenceTemplate builds a DAG template with a single withSequence task.
func withSequenceTemplate(taskName string, count string) *wfv1.Template {
	return createDAGTemplate([]wfv1.DAGTask{
		{
			Name:         taskName,
			Template:     "echo",
			WithSequence: &wfv1.Sequence{Count: intstrPtr(count)},
		},
	})
}

func TestHasExpansion(t *testing.T) {
	t.Run("withItems", func(t *testing.T) {
		assert.True(t, HasExpansion(&DAGTask{DAGTask: &wfv1.DAGTask{
			Name:      "x",
			WithItems: []wfv1.Item{{Value: []byte(`"a"`)}},
		}}))
	})
	t.Run("withParam", func(t *testing.T) {
		assert.True(t, HasExpansion(&DAGTask{DAGTask: &wfv1.DAGTask{
			Name:      "x",
			WithParam: "{{tasks.upstream.outputs.result}}",
		}}))
	})
	t.Run("withSequence", func(t *testing.T) {
		assert.True(t, HasExpansion(&DAGTask{DAGTask: &wfv1.DAGTask{
			Name:         "x",
			WithSequence: &wfv1.Sequence{Count: intstrPtr("3")},
		}}))
	})
	t.Run("none of them", func(t *testing.T) {
		assert.False(t, HasExpansion(&DAGTask{DAGTask: &wfv1.DAGTask{Name: "x"}}))
	})
	t.Run("empty withItems slice is not an expansion", func(t *testing.T) {
		// An explicit `withItems: []` is equivalent to omitting withItems: the
		// task runs once (DAGTask.ShouldExpand, validation and ExpandTask all
		// use len > 0). Only an expansion that yields zero items is skipped.
		assert.False(t, HasExpansion(&DAGTask{DAGTask: &wfv1.DAGTask{
			Name:      "x",
			WithItems: []wfv1.Item{},
		}}))
	})
}

// EvaluateAll returns one result per task: an expanded task has one, for its
// TaskGroup, however many items it has; the Engine drives the items.
func TestDAGEvaluator_EvaluateAll_OneResultPerTask(t *testing.T) {
	wf := newTestWorkflow("wf")
	parent := addTaskGroupParent(t, wf, "dag.client")
	addTaskGroupChild(t, wf, parent, "dag.client(0:0)", wfv1.NodePending, nil)
	addTaskGroupChild(t, wf, parent, "dag.client(1:1)", wfv1.NodeSucceeded, nil)
	tmpl := withSequenceTemplate("client", "2")
	tmpl.DAG.Tasks = append(tmpl.DAG.Tasks, wfv1.DAGTask{Name: "after", Depends: "client"})
	results := NewDAGEvaluator(wf, tmpl, "", "dag").EvaluateAll(testCtx(t))

	assert.ElementsMatch(t, []string{"client", "after"}, slices.Collect(maps.Keys(results)))
	assert.Equal(t, ActionExecute, results["client"].Action)
}

func TestWorkflowStore_GetTaskGroupChildren(t *testing.T) {
	t.Run("returns expanded children, skips Hooked and Retried", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(t, wf, "dag.client")
		addTaskGroupChild(t, wf, parent, "dag.client(0:0)", wfv1.NodePending, nil)
		addTaskGroupChild(t, wf, parent, "dag.client(1:1)", wfv1.NodeRunning, nil)
		addTaskGroupChild(t, wf, parent, "dag.client.onExit", wfv1.NodeSucceeded, &wfv1.NodeFlag{Hooked: true})
		addTaskGroupChild(t, wf, parent, "dag.client(2:2)(0)", wfv1.NodeSucceeded, &wfv1.NodeFlag{Retried: true})

		s := newWorkflowStore(wf, "", "dag")
		children := s.getTaskGroupChildren("client")

		require.Len(t, children, 2)
		names := []string{children[0].Name, children[1].Name}
		assert.Contains(t, names, "dag.client(0:0)")
		assert.Contains(t, names, "dag.client(1:1)")
	})

	t.Run("returns nil when the named task has no node", func(t *testing.T) {
		s := newWorkflowStore(newTestWorkflow("wf"), "", "dag")
		assert.Nil(t, s.getTaskGroupChildren("missing"))
	})

	t.Run("returns nil when the node is not a TaskGroup", func(t *testing.T) {
		// A regular Pod task with no expansion shouldn't be treated as a TaskGroup
		// even if it somehow has child references.
		wf := newTestWorkflow("wf")
		addNodeToWorkflow(testCtx(t), wf, "dag.client", wfv1.NodeRunning) // type=Pod

		s := newWorkflowStore(wf, "", "dag")
		assert.Nil(t, s.getTaskGroupChildren("client"))
	})

	t.Run("TaskGroup with only Hooked children returns empty slice (not nil)", func(t *testing.T) {
		// Distinguishes "node not a TaskGroup" (nil) from "TaskGroup with no
		// schedulable children" (empty). Both behave identically downstream
		// (range loop is a no-op), but exposing the distinction keeps the
		// diagnostic useful if someone audits the state.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(t, wf, "dag.client")
		addTaskGroupChild(t, wf, parent, "dag.client.onExit", wfv1.NodePending, &wfv1.NodeFlag{Hooked: true})

		s := newWorkflowStore(wf, "", "dag")
		children := s.getTaskGroupChildren("client")
		assert.Empty(t, children, "all children filtered out, expect empty result")
	})

	t.Run("Children IDs that don't resolve to a node are silently skipped", func(t *testing.T) {
		// Defensive: garbage-collected children or pre-init parent state
		// shouldn't crash the evaluator.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(t, wf, "dag.client")
		// Add a real child plus a dangling ID
		addTaskGroupChild(t, wf, parent, "dag.client(0:0)", wfv1.NodePending, nil)
		parent.Children = append(parent.Children, "non-existent-id")
		wf.Status.Nodes.Set(testCtx(t), parent.ID, *parent)

		s := newWorkflowStore(wf, "", "dag")
		children := s.getTaskGroupChildren("client")
		assert.Len(t, children, 1, "dangling child IDs should be skipped, real child preserved")
	})
}

func TestEvaluateRetryNode_RetryDeciderIsAuthoritative(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	childNodeID := wf.NodeID("dag.A(0)")

	wf.Status.Nodes.Set(testCtx(t), childNodeID, wfv1.NodeStatus{
		ID:       childNodeID,
		Name:     "dag.A(0)",
		Phase:    wfv1.NodeFailed,
		Type:     wfv1.NodeTypePod,
		NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{childNodeID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	// Policy Always would retry; the engine-provided decider says no (e.g. a
	// retryStrategy.expression evaluated to false) and must win.
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("2"), RetryPolicy: wfv1.RetryPolicyAlways})
	eval.SetRetryDecider("A", func(_ context.Context, _, _ *wfv1.NodeStatus, _ *wfv1.RetryStrategy) bool { return false })

	result := eval.EvaluateTask(testCtx(t), "A")

	assert.Equal(t, ActionFail, result.Action)
	assert.False(t, result.ShouldRun)
	assert.True(t, result.FulfilledForDeps)
}

func TestTaskNodeName_RoundTrip(t *testing.T) {
	for _, tt := range []struct{ boundary, task, node string }{
		{"wf.dag", "build", "wf.dag.build"},
		{"wf.dag", "build(0:x)", "wf.dag.build(0:x)"},
		{"wf.steps", "[1].step-b", "wf.steps[1].step-b"},
	} {
		assert.Equal(t, tt.node, TaskNodeName(tt.boundary, tt.task))
		assert.Equal(t, tt.task, TaskNameFromNodeName(tt.boundary, tt.node))
	}
	// A node that merely shares the boundary as a string prefix is not a child.
	assert.Equal(t, "boundaryother", TaskNameFromNodeName("boundary", "boundaryother"))
	assert.Equal(t, "elsewhere.x", TaskNameFromNodeName("boundary", "elsewhere.x"))
}

func TestActionString(t *testing.T) {
	assert.Equal(t, "None", ActionNone.String())
	assert.Equal(t, "Execute", ActionExecute.String())
	assert.Equal(t, "Succeed", ActionSucceed.String())
	assert.Equal(t, "Fail", ActionFail.String())
	assert.Equal(t, "Action(9)", Action(9).String())
}
