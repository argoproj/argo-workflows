package dag

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	intstrutil "github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// testCtx returns a context with a test logger for Argo workflow operations.
func testCtx() context.Context {
	return logging.TestContext(context.Background())
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

// --- Tests for isTerminalPhase ---

func TestIsTerminalPhase(t *testing.T) {
	assert.True(t, isTerminalPhase(wfv1.NodeSucceeded))
	assert.True(t, isTerminalPhase(wfv1.NodeFailed))
	assert.True(t, isTerminalPhase(wfv1.NodeError))
	assert.True(t, isTerminalPhase(wfv1.NodeSkipped))
	assert.True(t, isTerminalPhase(wfv1.NodeOmitted))
	assert.False(t, isTerminalPhase(wfv1.NodePending))
	assert.False(t, isTerminalPhase(wfv1.NodeRunning))
	assert.False(t, isTerminalPhase(""))
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

		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeSucceeded)

		state := store.getPhase(context.Background(), "taskA")
		assert.Equal(t, wfv1.NodeSucceeded, state)
	})

	t.Run("returns pending for nonexistent node", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		store := newWorkflowStore(wf, "", "dag")

		state := store.getPhase(context.Background(), "nonexistent")
		assert.Equal(t, wfv1.NodePending, state)
	})
}

func TestWorkflowStore_GetNode(t *testing.T) {
	t.Run("returns node for task", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		store := newWorkflowStore(wf, "", "dag")

		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeSucceeded)

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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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
		ctx := context.Background()
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
		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeSucceeded)

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
		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeRunning)

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
		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeSucceeded)

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
		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeFailed)

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

		ctx := context.Background()

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
		addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeSucceeded)
		evaluator = NewDAGEvaluator(wf, tmpl, "", "dag")

		result = evaluator.EvaluateTask(ctx, "B")
		assert.True(t, result.ShouldRun)

		result = evaluator.EvaluateTask(ctx, "C")
		assert.True(t, result.ShouldRun)

		result = evaluator.EvaluateTask(ctx, "D")
		assert.True(t, result.Suspended)

		// After B and C succeed
		addNodeToWorkflow(testCtx(), wf, "dag.B", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(), wf, "dag.C", wfv1.NodeSucceeded)
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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
		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(), wf, "dag.taskB", wfv1.NodeFailed)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA.Succeeded || taskB.Succeeded"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := context.Background()
		result := evaluator.EvaluateTask(ctx, "taskC")

		assert.True(t, result.ShouldRun)
	})

	t.Run("AND expression with both conditions met", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(), wf, "dag.taskB", wfv1.NodeFailed)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA.Succeeded && taskB.Failed"},
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := context.Background()
		result := evaluator.EvaluateTask(ctx, "taskC")

		assert.True(t, result.ShouldRun)
	})

	t.Run("AND expression with one condition not met", func(t *testing.T) {
		wf := newTestWorkflow("test-wf")
		addNodeToWorkflow(testCtx(), wf, "dag.taskA", wfv1.NodeSucceeded)
		addNodeToWorkflow(testCtx(), wf, "dag.taskB", wfv1.NodeSucceeded)

		tmpl := createDAGTemplate([]wfv1.DAGTask{
			{Name: "taskA"},
			{Name: "taskB"},
			{Name: "taskC", Depends: "taskA.Succeeded && taskB.Failed"}, // B didn't fail
		})
		evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

		ctx := context.Background()
		result := evaluator.EvaluateTask(ctx, "taskC")

		assert.False(t, result.ShouldRun)
		assert.True(t, result.Skipped)
	})
}

// --- Tests for unreachable task evaluation ---

func TestDAGEvaluator_UnreachableTask(t *testing.T) {
	// A fails, B depends on A.Succeeded → B should be Skipped
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Succeeded"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx()
	result := evaluator.EvaluateTask(ctx, "B")

	assert.False(t, result.ShouldRun, "B should not run since A failed")
	assert.True(t, result.Skipped, "B should be skipped since A.Succeeded can never be true")
	assert.False(t, result.Suspended, "B should not be suspended")
}

func TestDAGEvaluator_CascadingOmission(t *testing.T) {
	// A fails, B depends on A.Succeeded, C depends on B
	// B is marked Omitted, C sees B as Omitted and is also Skipped
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Succeeded"},
		{Name: "C", Depends: "B"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx()

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
	addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Failed"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx()
	result := evaluator.EvaluateTask(ctx, "B")

	assert.True(t, result.ShouldRun, "B should run since A.Failed is true")
	assert.False(t, result.Suspended, "B should not be suspended")
	assert.False(t, result.Skipped, "B should not be skipped")
}

func TestDAGEvaluator_MixedReachability(t *testing.T) {
	// Diamond: A(failed), B depends on A.Succeeded (unreachable),
	// C depends on A.Failed (reachable), D depends on B && C
	wf := newTestWorkflow("test-wf")
	addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeFailed)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.Succeeded"},
		{Name: "C", Depends: "A.Failed"},
		{Name: "D", Depends: "B && C"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx()

	resultB := evaluator.EvaluateTask(ctx, "B")
	assert.True(t, resultB.Skipped, "B should be skipped (A.Succeeded is false)")
	assert.False(t, resultB.ShouldRun, "B should not run")

	resultC := evaluator.EvaluateTask(ctx, "C")
	assert.True(t, resultC.ShouldRun, "C should run (A.Failed is true)")
	assert.False(t, resultC.Skipped, "C should not be skipped")

	resultD := evaluator.EvaluateTask(ctx, "D")
	assert.True(t, resultD.Skipped, "D should be skipped (B is unreachable, so B && C can never be true)")
	assert.False(t, resultD.Suspended, "D should not be suspended")
}

func TestWorkflowStore_SetStateAndGetState(t *testing.T) {
	wf := newTestWorkflow("test-wf")
	store := newWorkflowStore(wf, "", "dag")
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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

func TestResolveDependencies_BooleanKeywords(t *testing.T) {
	// "taskA.Succeeded || false" — "false" should NOT be treated as a task name
	taskA := wfv1.DAGTask{Name: "taskA"}
	provider := func(name string) Task {
		if name == "taskA" {
			return &DAGTask{DAGTask: &taskA}
		}
		return nil
	}
	deps, logic, err := resolveDependencies("taskA.Succeeded || false", provider)
	require.NoError(t, err)
	assert.Equal(t, []string{"taskA"}, deps)
	assert.Contains(t, logic, "false")
	assert.NotContains(t, logic, normalizeTaskName("false"))
}

func TestResolveDependencies_StepNames(t *testing.T) {
	// Step tasks use "[groupIndex].stepName" naming (e.g., "[0].A").
	// The ".A" suffix must NOT be interpreted as a result qualifier —
	// it's part of the task name. The full "[0].A" should be treated
	// as a bare task name and expanded via expandDependency.
	taskA := wfv1.DAGTask{Name: "[0].A"}
	provider := func(name string) Task {
		if name == "[0].A" {
			return &DAGTask{DAGTask: &taskA}
		}
		return nil
	}
	deps, logic, err := resolveDependencies("[0].A", provider)
	require.NoError(t, err)
	assert.Equal(t, []string{"[0].A"}, deps)
	// Should be expanded like a bare task name, not split into [0] + A
	assert.Contains(t, logic, "Succeeded")
	assert.Contains(t, logic, "Skipped")
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
	addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeSucceeded)

	tmplWithDepends := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A"},
	})

	tmplWithDependencies := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Dependencies: []string{"A"}},
	})

	ctx := testCtx()

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
	addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeSucceeded)

	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A"},
		{Name: "B", Depends: "A.InvalidStatus"},
	})
	evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")

	ctx := testCtx()
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

	wf.Status.Nodes.Set(testCtx(), childNodeID, wfv1.NodeStatus{
		ID:    childNodeID,
		Name:  "dag.A(0)",
		Phase: wfv1.NodeRunning,
		Type:  wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(testCtx(), retryNodeID, wfv1.NodeStatus{
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

	ctx := testCtx()
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

	wf.Status.Nodes.Set(testCtx(), childNodeID, wfv1.NodeStatus{
		ID:       childNodeID,
		Name:     "dag.A(0)",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypePod,
		Daemoned: &daemoned,
	})
	wf.Status.Nodes.Set(testCtx(), retryNodeID, wfv1.NodeStatus{
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

	ctx := testCtx()
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

	wf.Status.Nodes.Set(testCtx(), childNodeID, wfv1.NodeStatus{
		ID:       childNodeID,
		Name:     "dag.A(0)",
		Phase:    wfv1.NodeFailed,
		Type:     wfv1.NodeTypePod,
		NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(), retryNodeID, wfv1.NodeStatus{
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

	ctx := testCtx()
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

	wf.Status.Nodes.Set(testCtx(), child0ID, wfv1.NodeStatus{
		ID: child0ID, Name: "dag.A(0)", Phase: wfv1.NodeFailed,
		Type: wfv1.NodeTypePod, NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(), child1ID, wfv1.NodeStatus{
		ID: child1ID, Name: "dag.A(1)", Phase: wfv1.NodeFailed,
		Type: wfv1.NodeTypePod, NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(), child2ID, wfv1.NodeStatus{
		ID: child2ID, Name: "dag.A(2)", Phase: wfv1.NodeFailed,
		Type: wfv1.NodeTypePod, NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(), retryNodeID, wfv1.NodeStatus{
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

	ctx := testCtx()
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

	wf.Status.Nodes.Set(testCtx(), childNodeID, wfv1.NodeStatus{
		ID:    childNodeID,
		Name:  "dag.A(0)",
		Phase: wfv1.NodeSucceeded,
		Type:  wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(testCtx(), retryNodeID, wfv1.NodeStatus{
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

	ctx := testCtx()
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
	wf.Status.Nodes.Set(testCtx(), childNodeID, wfv1.NodeStatus{
		ID:       childNodeID,
		Name:     "dag.A(0)",
		Phase:    wfv1.NodeFailed,
		Type:     wfv1.NodeTypePod,
		NodeFlag: &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(), retryNodeID, wfv1.NodeStatus{
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

	ctx := testCtx()
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

	ctx := context.Background()
	result := eval.EvaluateTask(ctx, "B")
	assert.True(t, result.ShouldRun, "B should be ready when A's retry daemon child is running")
}

func TestEvaluateTaskGroupNode_AllSucceeded(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Name = "test"
	wf.Status.Nodes = wfv1.Nodes{}
	tgID := wf.NodeID("test.A")
	c0ID := wf.NodeID("test.A(0)")
	c1ID := wf.NodeID("test.A(1)")
	wf.Status.Nodes[tgID] = wfv1.NodeStatus{
		ID: tgID, Name: "test.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	}
	wf.Status.Nodes[c0ID] = wfv1.NodeStatus{
		ID: c0ID, Name: "test.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	}
	wf.Status.Nodes[c1ID] = wfv1.NodeStatus{
		ID: c1ID, Name: "test.A(1)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	}
	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "test", "test")
	ctx := context.Background()
	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionSucceed, result.Action)
	assert.True(t, result.FulfilledForDeps)
}

func TestEvaluateTaskGroupNode_ChildFailed(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Name = "test"
	wf.Status.Nodes = wfv1.Nodes{}
	tgID := wf.NodeID("test.A")
	c0ID := wf.NodeID("test.A(0)")
	c1ID := wf.NodeID("test.A(1)")
	wf.Status.Nodes[tgID] = wfv1.NodeStatus{
		ID: tgID, Name: "test.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	}
	wf.Status.Nodes[c0ID] = wfv1.NodeStatus{
		ID: c0ID, Name: "test.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	}
	wf.Status.Nodes[c1ID] = wfv1.NodeStatus{
		ID: c1ID, Name: "test.A(1)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	}
	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "test", "test")
	ctx := context.Background()
	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action)
	assert.True(t, result.FulfilledForDeps)
}

func TestEvaluateTaskGroupNode_ChildStillRunning(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Name = "test"
	wf.Status.Nodes = wfv1.Nodes{}
	tgID := wf.NodeID("test.A")
	c0ID := wf.NodeID("test.A(0)")
	c1ID := wf.NodeID("test.A(1)")
	wf.Status.Nodes[tgID] = wfv1.NodeStatus{
		ID: tgID, Name: "test.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	}
	wf.Status.Nodes[c0ID] = wfv1.NodeStatus{
		ID: c0ID, Name: "test.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	}
	wf.Status.Nodes[c1ID] = wfv1.NodeStatus{
		ID: c1ID, Name: "test.A(1)", Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod,
	}
	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "test", "test")
	ctx := context.Background()
	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionNone, result.Action)
	assert.False(t, result.FulfilledForDeps)
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
	ctx := context.Background()

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

// ============================================================
// Retry edge cases — TestEval_Retry_*
// ============================================================

// 1. Retry limit zero — limit=0, one child Failed → ActionFail (0 retries allowed)
func TestEval_Retry_LimitZero(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
		ctx := testCtx()

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
	ctx := testCtx()

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
	ctx := testCtx()

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
func TestEval_TaskGroup_AllSucceeded(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	c0ID := wf.NodeID("dag.A(0)")
	c1ID := wf.NodeID("dag.A(1)")

	wf.Status.Nodes.Set(ctx, c0ID, wfv1.NodeStatus{
		ID: c0ID, Name: "dag.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, c1ID, wfv1.NodeStatus{
		ID: c1ID, Name: "dag.A(1)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionSucceed, result.Action)
	assert.Equal(t, wfv1.NodeSucceeded, result.CurrentPhase)
}

// 22. TaskGroup child failed → ActionFail + CurrentPhase=Failed
func TestEval_TaskGroup_ChildFailed(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	c0ID := wf.NodeID("dag.A(0)")
	c1ID := wf.NodeID("dag.A(1)")

	wf.Status.Nodes.Set(ctx, c0ID, wfv1.NodeStatus{
		ID: c0ID, Name: "dag.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, c1ID, wfv1.NodeStatus{
		ID: c1ID, Name: "dag.A(1)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action)
	assert.Equal(t, wfv1.NodeFailed, result.CurrentPhase)
}

// 23. TaskGroup child Error → ActionFail + CurrentPhase=Error (worst phase)
func TestEval_TaskGroup_ChildError(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	c0ID := wf.NodeID("dag.A(0)")
	c1ID := wf.NodeID("dag.A(1)")

	wf.Status.Nodes.Set(ctx, c0ID, wfv1.NodeStatus{
		ID: c0ID, Name: "dag.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, c1ID, wfv1.NodeStatus{
		ID: c1ID, Name: "dag.A(1)", Phase: wfv1.NodeError, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action)
	assert.Equal(t, wfv1.NodeError, result.CurrentPhase, "Error is worse than Failed")
}

// 24. TaskGroup child still running → ActionNone
func TestEval_TaskGroup_ChildStillRunning(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	c0ID := wf.NodeID("dag.A(0)")
	c1ID := wf.NodeID("dag.A(1)")

	wf.Status.Nodes.Set(ctx, c0ID, wfv1.NodeStatus{
		ID: c0ID, Name: "dag.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, c1ID, wfv1.NodeStatus{
		ID: c1ID, Name: "dag.A(1)", Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionNone, result.Action)
}

// 25. TaskGroup no children → ActionNone (still expanding)
func TestEval_TaskGroup_NoChildren(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")

	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionNone, result.Action, "TaskGroup with no children should wait (still expanding)")
}

// 26. TaskGroup daemoned child → ActionNone (daemon hasn't completed)
func TestEval_TaskGroup_DaemonedChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	c0ID := wf.NodeID("dag.A(0)")
	daemoned := true

	wf.Status.Nodes.Set(ctx, c0ID, wfv1.NodeStatus{
		ID: c0ID, Name: "dag.A(0)", Phase: wfv1.NodeRunning,
		Type:     wfv1.NodeTypePod,
		Daemoned: &daemoned,
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionNone, result.Action, "daemoned running child in TaskGroup should cause waiting")
}

// 27. TaskGroup stale Succeeded with failed child — node.Phase=Succeeded, child=Failed → CurrentPhase != Succeeded
func TestEval_TaskGroup_StaleSucceededWithFailedChild(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	c0ID := wf.NodeID("dag.A(0)")
	c1ID := wf.NodeID("dag.A(1)")

	wf.Status.Nodes.Set(ctx, c0ID, wfv1.NodeStatus{
		ID: c0ID, Name: "dag.A(0)", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, c1ID, wfv1.NodeStatus{
		ID: c1ID, Name: "dag.A(1)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	// TaskGroup node says Succeeded (stale), but a child actually failed
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeSucceeded,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.NotEqual(t, wfv1.NodeSucceeded, result.CurrentPhase,
		"stale Succeeded TaskGroup node with failed child should not report Succeeded")
}

// 28. TaskGroup orphaned children — node.Phase=Succeeded, children not in store → FulfilledForDeps=true (trust phase)
func TestEval_TaskGroup_OrphanedChildren(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	// Children exist in the node's Children list but are not in the store (pruned/GC'd)
	missingChildID := "nonexistent-child-id"

	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeSucceeded,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{missingChildID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.True(t, result.FulfilledForDeps,
		"TaskGroup with Succeeded phase and missing children (GC'd) should trust phase and be fulfilled for deps")
}

// 29. TaskGroup with Retry child exhausted → ActionFail
func TestEval_TaskGroup_RetryChildExhausted(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	// The retry node is a child of the task group
	retryChildID := wf.NodeID("dag.A-retry")
	retryGrandChildID := wf.NodeID("dag.A-retry(0)")
	retryGrandChild1ID := wf.NodeID("dag.A-retry(1)")

	wf.Status.Nodes.Set(ctx, retryGrandChildID, wfv1.NodeStatus{
		ID: retryGrandChildID, Name: "dag.A-retry(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryGrandChild1ID, wfv1.NodeStatus{
		ID: retryGrandChild1ID, Name: "dag.A-retry(1)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryChildID, wfv1.NodeStatus{
		ID: retryChildID, Name: "dag.A-retry", Phase: wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{retryGrandChildID, retryGrandChild1ID},
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{retryChildID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	// Register retry strategy under the boundary-stripped key, matching the
	// lookup performed by evaluateTaskGroupNode after taskNameFromNodeName.
	eval.SetRetryStrategy("A-retry", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("1")})

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action, "TaskGroup with exhausted retry child should fail")
}

// 30. TaskGroup mixed Error and Failed — one child Error, one Failed → CurrentPhase=Error (worst)
func TestEval_TaskGroup_MixedErrorAndFailed(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	tgID := wf.NodeID("dag.A")
	c0ID := wf.NodeID("dag.A(0)")
	c1ID := wf.NodeID("dag.A(1)")

	wf.Status.Nodes.Set(ctx, c0ID, wfv1.NodeStatus{
		ID: c0ID, Name: "dag.A(0)", Phase: wfv1.NodeError, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, c1ID, wfv1.NodeStatus{
		ID: c1ID, Name: "dag.A(1)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{c0ID, c1ID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	result := eval.EvaluateTask(ctx, "A")
	assert.Equal(t, ActionFail, result.Action)
	assert.Equal(t, wfv1.NodeError, result.CurrentPhase, "Error is worse than Failed — should be the current phase")
}

// --- Tests for the per-child evaluator path ---

// addTaskGroupChild adds a child node under a TaskGroup parent. The parent's
// Children list is updated. nodeType defaults to NodeTypePod when empty.
func addTaskGroupChild(wf *wfv1.Workflow, parent *wfv1.NodeStatus, name string, phase wfv1.NodePhase, nodeType wfv1.NodeType, flag *wfv1.NodeFlag) {
	if nodeType == "" {
		nodeType = wfv1.NodeTypePod
	}
	childID := wf.NodeID(name)
	child := wfv1.NodeStatus{
		ID:       childID,
		Name:     name,
		Phase:    phase,
		Type:     nodeType,
		NodeFlag: flag,
	}
	wf.Status.Nodes.Set(testCtx(), childID, child)
	parent.Children = append(parent.Children, childID)
	wf.Status.Nodes.Set(testCtx(), parent.ID, *parent)
}

// addTaskGroupParent creates a TaskGroup parent node (Running) for a withSequence/withItems/withParam task.
func addTaskGroupParent(wf *wfv1.Workflow, name string) *wfv1.NodeStatus {
	id := wf.NodeID(name)
	parent := wfv1.NodeStatus{
		ID:    id,
		Name:  name,
		Phase: wfv1.NodeRunning,
		Type:  wfv1.NodeTypeTaskGroup,
	}
	wf.Status.Nodes.Set(testCtx(), id, parent)
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
	t.Run("empty withItems slice still counts", func(t *testing.T) {
		// Distinguishes between nil (no expansion) and an empty slice (explicit empty
		// expansion that should still skip via empty-expansion handling). Either way,
		// the helper signals that this task uses one of the with* mechanisms.
		assert.True(t, HasExpansion(&DAGTask{DAGTask: &wfv1.DAGTask{
			Name:      "x",
			WithItems: []wfv1.Item{},
		}}))
	})
}

func TestDAGEvaluator_EvaluateAll_NoExpansion_OneResultPerTask(t *testing.T) {
	wf := newTestWorkflow("wf")
	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "a"},
		{Name: "b", Depends: "a"},
	})
	results := NewDAGEvaluator(wf, tmpl, "", "dag").EvaluateAll(testCtx())

	assert.Len(t, results, 2)
	for _, name := range []string{"a", "b"} {
		assert.Empty(t, results[name].ParentTaskName,
			"static task %q should have empty ParentTaskName", name)
	}
}

func TestDAGEvaluator_EvaluateAll_TaskGroupChildren(t *testing.T) {
	t.Run("emits one result per expanded child plus the parent", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", nil)
		addTaskGroupChild(wf, parent, "dag.client(1:1)", wfv1.NodePending, "", nil)
		addTaskGroupChild(wf, parent, "dag.client(2:2)", wfv1.NodePending, "", nil)

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "3"), "", "dag").EvaluateAll(testCtx())

		assert.Len(t, results, 4, "1 parent + 3 children")
		for _, name := range []string{"client", "client(0:0)", "client(1:1)", "client(2:2)"} {
			assert.Contains(t, results, name)
		}
	})

	t.Run("pending child gets ActionExecute and ShouldRun", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]

		assert.Equal(t, ActionExecute, r.Action)
		assert.True(t, r.ShouldRun)
		assert.Equal(t, "client", r.ParentTaskName,
			"ParentTaskName must point at the static parent for the engine to dispatch")
		assert.Equal(t, wfv1.NodePending, r.CurrentPhase)
	})

	t.Run("running child gets ActionNone — kube reconciler owns it", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeRunning, "", nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]

		assert.Equal(t, ActionNone, r.Action)
		assert.False(t, r.ShouldRun)
		assert.Equal(t, "client", r.ParentTaskName)
	})

	t.Run("succeeded child gets ActionNone", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeSucceeded, "", nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]

		assert.Equal(t, ActionNone, r.Action)
		assert.False(t, r.ShouldRun)
	})

	t.Run("failed child gets ActionNone (no auto-retry at this layer)", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeFailed, "", nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]

		assert.Equal(t, ActionNone, r.Action)
	})

	t.Run("hooked children are not emitted", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", nil)
		addTaskGroupChild(wf, parent, "dag.client.onExit", wfv1.NodePending, "", &wfv1.NodeFlag{Hooked: true})

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").EvaluateAll(testCtx())

		assert.Contains(t, results, "client(0:0)")
		assert.NotContains(t, results, "client.onExit",
			"Hook scaffolding must not show up as a schedulable child")
	})

	t.Run("retry-attempt children are not emitted", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", &wfv1.NodeFlag{Retried: true})

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").EvaluateAll(testCtx())

		assert.NotContains(t, results, "client(0:0)",
			"Retry-attempt scaffolding must not show up as a schedulable child")
	})

	t.Run("retry-typed child defers to evaluateRetryNode", func(t *testing.T) {
		// Retry-typed TaskGroup children carry their own state machine; the per-child
		// evaluator must hand off, not fabricate ActionExecute. With no retry attempts
		// yet, evaluateRetryNode returns ActionExecute ("first retry attempt needed").
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeRunning, wfv1.NodeTypeRetry, nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]

		assert.Equal(t, ActionExecute, r.Action,
			"with no retry attempts, evaluateRetryNode should request the first attempt")
		assert.Equal(t, "first retry attempt needed", r.ActionReason)
		assert.Equal(t, "client", r.ParentTaskName,
			"ParentTaskName is set even when delegating to evaluateRetryNode")
	})

	t.Run("parent without a TaskGroup node yet contributes no child results", func(t *testing.T) {
		// First operate cycle: parent task is in the template with withSequence,
		// but no TaskGroup node has been created yet. EvaluateAll should still
		// return the parent's own result without crashing or fabricating children.
		wf := newTestWorkflow("wf")
		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "3"), "", "dag").EvaluateAll(testCtx())
		assert.Len(t, results, 1)
		assert.Contains(t, results, "client")
	})
}

// --- Tests for workflowStore helpers added for the per-child path ---

func TestWorkflowStore_TaskNameFromNodeName(t *testing.T) {
	t.Run("DAG boundary uses dot separator", func(t *testing.T) {
		s := newWorkflowStore(newTestWorkflow("wf"), "", "dag")
		assert.Equal(t, "client", s.taskNameFromNodeName("dag.client"))
		assert.Equal(t, "client(0:0)", s.taskNameFromNodeName("dag.client(0:0)"))
	})
	t.Run("Steps boundary preserves [N] prefix", func(t *testing.T) {
		// Step task names are formatted as "[N].stepName"; node names look like
		// "wf.steps[0].stepName". After stripping the boundary, the [N] must remain.
		s := newWorkflowStore(newTestWorkflow("wf"), "", "wf.steps")
		assert.Equal(t, "[0].step1", s.taskNameFromNodeName("wf.steps[0].step1"))
	})
	t.Run("inverse of taskNodeName for both shapes", func(t *testing.T) {
		s := newWorkflowStore(newTestWorkflow("wf"), "", "boundary")
		for _, taskName := range []string{"a", "task(0:foo)", "[0].step1"} {
			roundTripped := s.taskNameFromNodeName(s.taskNodeName(taskName))
			assert.Equal(t, taskName, roundTripped, "round-trip mismatch for %q", taskName)
		}
	})
	t.Run("input without boundary prefix is returned as-is", func(t *testing.T) {
		s := newWorkflowStore(newTestWorkflow("wf"), "", "boundary")
		assert.Equal(t, "unrelated", s.taskNameFromNodeName("unrelated"))
	})
	t.Run("rejects nodes whose name only happens to share the boundary prefix", func(t *testing.T) {
		// taskNodeName always inserts "." or "[" between boundary and task name.
		// A node literally named "boundaryother" is NOT a child of "boundary"
		// even though it has the boundary as a prefix; the inverse must not
		// strip a partial prefix.
		s := newWorkflowStore(newTestWorkflow("wf"), "", "boundary")
		// Bug: a naive TrimPrefix would yield "other" here. We expect it to
		// recognize that there's no separator after the boundary and either
		// return the input unchanged or otherwise not pretend it's a child.
		got := s.taskNameFromNodeName("boundaryother")
		assert.NotEqual(t, "other", got,
			"a node merely sharing the boundary prefix is not a child — no truncation expected")
	})
}

func TestWorkflowStore_GetTaskGroupChildren(t *testing.T) {
	t.Run("returns expanded children, skips Hooked and Retried", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", nil)
		addTaskGroupChild(wf, parent, "dag.client(1:1)", wfv1.NodeRunning, "", nil)
		addTaskGroupChild(wf, parent, "dag.client.onExit", wfv1.NodeSucceeded, "", &wfv1.NodeFlag{Hooked: true})
		addTaskGroupChild(wf, parent, "dag.client(2:2)(0)", wfv1.NodeSucceeded, "", &wfv1.NodeFlag{Retried: true})

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
		addNodeToWorkflow(testCtx(), wf, "dag.client", wfv1.NodeRunning) // type=Pod

		s := newWorkflowStore(wf, "", "dag")
		assert.Nil(t, s.getTaskGroupChildren("client"))
	})

	t.Run("TaskGroup with only Hooked children returns empty slice (not nil)", func(t *testing.T) {
		// Distinguishes "node not a TaskGroup" (nil) from "TaskGroup with no
		// schedulable children" (empty). Both behave identically downstream
		// (range loop is a no-op), but exposing the distinction keeps the
		// diagnostic useful if someone audits the state.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client.onExit", wfv1.NodePending, "", &wfv1.NodeFlag{Hooked: true})

		s := newWorkflowStore(wf, "", "dag")
		children := s.getTaskGroupChildren("client")
		assert.Empty(t, children, "all children filtered out, expect empty result")
	})

	t.Run("Children IDs that don't resolve to a node are silently skipped", func(t *testing.T) {
		// Defensive: garbage-collected children or pre-init parent state
		// shouldn't crash the evaluator.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		// Add a real child plus a dangling ID
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", nil)
		parent.Children = append(parent.Children, "non-existent-id")
		wf.Status.Nodes.Set(testCtx(), parent.ID, *parent)

		s := newWorkflowStore(wf, "", "dag")
		children := s.getTaskGroupChildren("client")
		assert.Len(t, children, 1, "dangling child IDs should be skipped, real child preserved")
	})
}

// --- Edge case tests for DAGEvaluator's per-child path ---

func TestDAGEvaluator_EvaluateAll_TaskGroupChildren_EdgeCases(t *testing.T) {
	t.Run("mixed phases: Pending re-dispatches, Running/Succeeded do not", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeSucceeded, "", nil)
		addTaskGroupChild(wf, parent, "dag.client(1:1)", wfv1.NodePending, "", nil)
		addTaskGroupChild(wf, parent, "dag.client(2:2)", wfv1.NodeRunning, "", nil)

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "3"), "", "dag").EvaluateAll(testCtx())

		assert.Equal(t, ActionNone, results["client(0:0)"].Action, "Succeeded → ActionNone")
		assert.Equal(t, ActionExecute, results["client(1:1)"].Action, "Pending → ActionExecute")
		assert.True(t, results["client(1:1)"].ShouldRun)
		assert.Equal(t, ActionNone, results["client(2:2)"].Action, "Running → ActionNone")

		// All carry ParentTaskName so the engine can dispatch consistently.
		for _, name := range []string{"client(0:0)", "client(1:1)", "client(2:2)"} {
			assert.Equal(t, "client", results[name].ParentTaskName, "ParentTaskName for %q", name)
		}
	})

	t.Run("all children Pending: every one gets ActionExecute", func(t *testing.T) {
		// This is the core "lock release" scenario: many siblings queued on a
		// shared mutex, none have run yet, evaluator should mark every one for
		// dispatch so handleSynchronization gets a chance to TryAcquire.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		for i := range 5 {
			addTaskGroupChild(wf, parent, fmt.Sprintf("dag.client(%d:%d)", i, i), wfv1.NodePending, "", nil)
		}

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "5"), "", "dag").EvaluateAll(testCtx())

		for i := range 5 {
			r := results[fmt.Sprintf("client(%d:%d)", i, i)]
			assert.Equal(t, ActionExecute, r.Action)
			assert.True(t, r.ShouldRun)
		}
	})

	t.Run("all children Succeeded: no dispatches, parent should be ready to terminate", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		for i := range 3 {
			addTaskGroupChild(wf, parent, fmt.Sprintf("dag.client(%d:%d)", i, i), wfv1.NodeSucceeded, "", nil)
		}

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "3"), "", "dag").EvaluateAll(testCtx())

		for i := range 3 {
			r := results[fmt.Sprintf("client(%d:%d)", i, i)]
			assert.Equal(t, ActionNone, r.Action)
			assert.False(t, r.ShouldRun)
		}
	})

	t.Run("Failed and Pending mixed: Failed left alone (no auto-retry), Pending re-dispatches", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeFailed, "", nil)
		addTaskGroupChild(wf, parent, "dag.client(1:1)", wfv1.NodePending, "", nil)

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "2"), "", "dag").EvaluateAll(testCtx())

		assert.Equal(t, ActionNone, results["client(0:0)"].Action,
			"Failed without retryStrategy stays terminal — must not be re-dispatched")
		assert.Equal(t, ActionExecute, results["client(1:1)"].Action)
	})

	t.Run("Errored child also stays terminal", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeError, "", nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]
		assert.Equal(t, ActionNone, r.Action)
	})

	t.Run("daemoned Running child: ActionNone, kube reconciler owns it", func(t *testing.T) {
		// Daemoned children look like Running pods that are intentionally
		// long-lived. The per-child evaluator must treat them as Running
		// (no re-dispatch) — they're not "stuck pending on sync" candidates.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		childID := wf.NodeID("dag.client(0:0)")
		daemoned := true
		wf.Status.Nodes.Set(testCtx(), childID, wfv1.NodeStatus{
			ID:       childID,
			Name:     "dag.client(0:0)",
			Phase:    wfv1.NodeRunning,
			Type:     wfv1.NodeTypePod,
			Daemoned: &daemoned,
		})
		parent.Children = []string{childID}
		wf.Status.Nodes.Set(testCtx(), parent.ID, *parent)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]
		assert.Equal(t, ActionNone, r.Action,
			"daemoned Running child should not be re-dispatched")
	})

	t.Run("TaskGroup with only Hooked children: parent result only, no children", func(t *testing.T) {
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client.onExit", wfv1.NodeSucceeded, "", &wfv1.NodeFlag{Hooked: true})

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").EvaluateAll(testCtx())
		assert.Len(t, results, 1, "only the parent should appear")
		assert.Contains(t, results, "client")
	})

	t.Run("parent's own evaluation is not disturbed by per-child path", func(t *testing.T) {
		// The per-child results are appended; the parent's EvaluationResult
		// must come from evaluateTaskResult (which routes to evaluateTaskGroupNode).
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		// All children Succeeded → evaluateTaskGroupNode returns ActionSucceed.
		for i := range 3 {
			addTaskGroupChild(wf, parent, fmt.Sprintf("dag.client(%d:%d)", i, i), wfv1.NodeSucceeded, "", nil)
		}

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "3"), "", "dag").EvaluateAll(testCtx())

		// Parent assessment: all children done → ActionSucceed (parent's own action).
		parentResult := results["client"]
		assert.Equal(t, ActionSucceed, parentResult.Action)
		assert.True(t, parentResult.FulfilledForDeps)
		assert.Empty(t, parentResult.ParentTaskName, "parent itself has no parent")
	})

	t.Run("retry-typed child with exhausted attempts gets ActionFail from delegate", func(t *testing.T) {
		// Wire up: TaskGroup parent → Retry-typed child → all attempt children Failed.
		// retryStrategy.Limit defaults to none, so absent strategy means "no
		// retries allowed" — first failure is terminal.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		retryChildName := "dag.client(0:0)"
		retryChildID := wf.NodeID(retryChildName)
		retryChild := wfv1.NodeStatus{
			ID:    retryChildID,
			Name:  retryChildName,
			Phase: wfv1.NodeRunning,
			Type:  wfv1.NodeTypeRetry,
		}
		// One failed attempt under the Retry node.
		attemptName := "dag.client(0:0)(0)"
		attemptID := wf.NodeID(attemptName)
		wf.Status.Nodes.Set(testCtx(), attemptID, wfv1.NodeStatus{
			ID:       attemptID,
			Name:     attemptName,
			Phase:    wfv1.NodeFailed,
			Type:     wfv1.NodeTypePod,
			NodeFlag: &wfv1.NodeFlag{Retried: true},
		})
		retryChild.Children = []string{attemptID}
		wf.Status.Nodes.Set(testCtx(), retryChildID, retryChild)
		parent.Children = []string{retryChildID}
		wf.Status.Nodes.Set(testCtx(), parent.ID, *parent)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]

		assert.Equal(t, ActionFail, r.Action,
			"with no retry strategy and a failed attempt, delegate must return ActionFail")
		assert.Equal(t, "client", r.ParentTaskName,
			"ParentTaskName must be set even when delegating to evaluateRetryNode")
	})

	t.Run("nested TaskGroup: only the immediate parent's children are emitted", func(t *testing.T) {
		// If a TaskGroup's child happens to also be a TaskGroup (rare but possible
		// via templates that fan out), the per-child evaluator must NOT recurse.
		// The grandchildren belong to the inner TaskGroup's own dispatch loop.
		wf := newTestWorkflow("wf")
		outer := addTaskGroupParent(wf, "dag.outer")
		// Inner is itself a TaskGroup, child of outer
		innerName := "dag.outer(0:0)"
		innerID := wf.NodeID(innerName)
		inner := wfv1.NodeStatus{
			ID:    innerID,
			Name:  innerName,
			Phase: wfv1.NodeRunning,
			Type:  wfv1.NodeTypeTaskGroup,
		}
		// Grandchild under inner
		grandName := "dag.outer(0:0).inner(0:0)"
		grandID := wf.NodeID(grandName)
		wf.Status.Nodes.Set(testCtx(), grandID, wfv1.NodeStatus{
			ID:    grandID,
			Name:  grandName,
			Phase: wfv1.NodePending,
			Type:  wfv1.NodeTypePod,
		})
		inner.Children = []string{grandID}
		wf.Status.Nodes.Set(testCtx(), innerID, inner)
		outer.Children = []string{innerID}
		wf.Status.Nodes.Set(testCtx(), outer.ID, *outer)

		// We only care about outer's per-child emission here.
		results := NewDAGEvaluator(wf, withSequenceTemplate("outer", "1"), "", "dag").EvaluateAll(testCtx())

		// outer(0:0) (the inner TaskGroup) should appear as outer's child.
		assert.Contains(t, results, "outer(0:0)")
		// Grandchildren must NOT be emitted from outer's per-child loop.
		assert.NotContains(t, results, "outer(0:0).inner(0:0)",
			"grandchildren are not direct children of outer; they belong to inner's own dispatch")
	})

	t.Run("child node missing from store is silently skipped (no panic)", func(t *testing.T) {
		// Defensive: if a child ID in parent.Children doesn't resolve to a node
		// (GC, partial write, etc.), the evaluator must not crash.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", nil)
		parent.Children = append(parent.Children, "phantom-id")
		wf.Status.Nodes.Set(testCtx(), parent.ID, *parent)

		results := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").EvaluateAll(testCtx())
		assert.Contains(t, results, "client(0:0)", "real child preserved")
		// Should not blow up; phantom not present.
	})

	t.Run("Skipped child is treated as terminal — no re-dispatch", func(t *testing.T) {
		// A child whose 'when' evaluated false is marked Skipped (terminal). The
		// per-child evaluator must NOT try to re-dispatch it; that would loop.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeSkipped, "", nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]
		assert.Equal(t, ActionNone, r.Action)
		assert.False(t, r.ShouldRun)
	})

	t.Run("Omitted child is treated as terminal — no re-dispatch", func(t *testing.T) {
		// Same logic as Skipped: terminal, leave alone.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodeOmitted, "", nil)

		r := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag").
			EvaluateAll(testCtx())["client(0:0)"]
		assert.Equal(t, ActionNone, r.Action)
	})

	t.Run("results map is not mutated after EvaluateAll returns (snapshot semantics)", func(t *testing.T) {
		// EvaluateAll returns a fresh map each call. Subsequent state changes
		// must not leak into the previously returned snapshot.
		wf := newTestWorkflow("wf")
		parent := addTaskGroupParent(wf, "dag.client")
		addTaskGroupChild(wf, parent, "dag.client(0:0)", wfv1.NodePending, "", nil)
		evaluator := NewDAGEvaluator(wf, withSequenceTemplate("client", "1"), "", "dag")

		first := evaluator.EvaluateAll(testCtx())
		require.Equal(t, ActionExecute, first["client(0:0)"].Action)

		// Mark the child Succeeded and re-evaluate. The first snapshot must not
		// reflect the new phase — it's a return value, not a live view.
		for id, node := range wf.Status.Nodes {
			if node.Name == "dag.client(0:0)" {
				node.Phase = wfv1.NodeSucceeded
				wf.Status.Nodes[id] = node
			}
		}
		second := evaluator.EvaluateAll(testCtx())
		assert.Equal(t, ActionNone, second["client(0:0)"].Action, "second call sees the new phase")
		assert.Equal(t, ActionExecute, first["client(0:0)"].Action, "first call must remain unchanged")
	})
}
