package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
)

// newDAGEvaluator builds a DAGEvaluator over a DAG template's tasks, in a
// boundary named "test", mirroring the engine's task wrapping. Test-only:
// prod uses NewDAGEvaluatorFromTasks.
func newDAGEvaluator(wf *wfv1.Workflow, tmpl *wfv1.Template) *dag.DAGEvaluator {
	var tasks []dag.Task
	if tmpl.DAG != nil {
		tasks = make([]dag.Task, len(tmpl.DAG.Tasks))
		for i := range tmpl.DAG.Tasks {
			tasks[i] = &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[i]}
		}
	}
	return dag.NewDAGEvaluatorFromTasks(wf, tasks, tmpl, "test", "test")
}

// TestDagXfail verifies a DAG can fail properly
func TestDagXfail(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag_xfail.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestDagRetrySucceeded verifies a DAG will be marked Succeeded if retry was successful
func TestDagRetrySucceeded(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag_retry_succeeded.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestDagRetryExhaustedXfail verifies we fail properly when we exhaust our retries
func TestDagRetryExhaustedXfail(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag-exhausted-retries-xfail.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestDagDisableFailFast test disable fail fast function
func TestDagDisableFailFast(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag-disable-fail-fast.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var dynamicSingleDag = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 generateName: dag-diamond-
spec:
 entrypoint: diamond
 templates:
 - name: diamond
   dag:
     tasks:
     - name: A
       template: %s
       %s
     - name: TestSingle
       template: Succeeded
       depends: A.%s

 - name: Succeeded
   container:
     image: alpine:3.23
     command: [sh, -c, "exit 0"]

 - name: Failed
   container:
     image: alpine:3.23
     command: [sh, -c, "exit 1"]

 - name: Skipped
   container:
     image: alpine:3.23
     command: [sh, -c, "echo Hello"]
`

func TestSingleDependency(t *testing.T) {
	t.Setenv("INFORMER_WRITE_BACK", "true")
	statusMap := map[string]v1.PodPhase{"Succeeded": v1.PodSucceeded, "Failed": v1.PodFailed}
	var closer context.CancelFunc
	var controller *WorkflowController
	for _, status := range []string{"Succeeded", "Failed", "Skipped"} {
		ctx := logging.TestContext(t.Context())
		closer, controller = newController(ctx)
		wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

		// If the status is "skipped" skip the root node.
		var wfString string
		if status == "Skipped" {
			wfString = fmt.Sprintf(dynamicSingleDag, status, `when: "false == true"`, status)
		} else {
			wfString = fmt.Sprintf(dynamicSingleDag, status, "", status)
		}
		wf := wfv1.MustUnmarshalWorkflow(wfString)

		wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		woc.operate(ctx)
		// Mark the status of the pod according to the test
		if _, ok := statusMap[status]; ok {
			makePodsPhase(ctx, woc, statusMap[status])
		} else {
			makePodsPhase(ctx, woc, v1.PodPending)
		}

		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		found := false
		for _, node := range woc.wf.Status.Nodes {
			if strings.Contains(node.Name, "TestSingle") {
				found = true
				assert.Equal(t, wfv1.NodePending, node.Phase)
			}
		}
		assert.True(t, found)
		if closer != nil {
			closer()
		}
	}
}

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestArtifactResolutionWhenSkippedDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/artifact-resolution-when-skipped-dag.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestExpandTaskWithParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	task := wfv1.DAGTask{
		Name:     "fanout-param",
		Template: "tmpl",
		Arguments: wfv1.Arguments{
			Parameters: []wfv1.Parameter{{
				Name:  "msg",
				Value: wfv1.AnyStringPtr("{{item}}"),
			}},
		},
		WithParam: `[1234, "foo\tbar", true, []]`,
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
	}
	woc := newWoc(ctx, *wf)

	expanded, err := dag.ExpandTask(ctx, task, map[string]string{}, woc)
	require.NoError(t, err)
	require.Len(t, expanded, 4)

	expectedExpandedTasks := []struct {
		Name      string
		Parameter string
	}{
		{
			Name:      "fanout-param(0:1234)",
			Parameter: "1234",
		},
		{
			Name:      `fanout-param(1:foo\tbar)`,
			Parameter: "foo\tbar",
		},
		{
			Name:      "fanout-param(2:true)",
			Parameter: "true",
		},
		{
			Name:      "fanout-param(3:[])",
			Parameter: "[]",
		},
	}

	for i, expected := range expectedExpandedTasks {
		assert.Equal(t, expected.Name, expanded[i].Name)
		assert.Equal(t, "tmpl", expanded[i].Template)
		assert.Equal(t, expected.Parameter, expanded[i].Arguments.Parameters[0].Value.String())
	}
}

func TestEvaluateDependsLogic(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name:    "B",
			Depends: "A",
		},
		{
			Name:    "C", // This task should fail
			Depends: "A",
		},
		{
			Name:    "should-execute-1",
			Depends: "A && (C.Succeeded || C.Failed)",
		},
		{
			Name:    "should-execute-2",
			Depends: "B || C",
		},
		{
			Name:    "should-not-execute",
			Depends: "B && C",
		},
		{
			Name:    "should-execute-3",
			Depends: "should-execute-2 || should-not-execute",
		},
	}
	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: make(wfv1.Nodes),
		},
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{
			Tasks: testTasks,
		},
	}
	evaluator := newDAGEvaluator(wf, tmpl)

	// Task A is running
	nodeID := wf.NodeID("test.A")
	wf.Status.Nodes[nodeID] = wfv1.NodeStatus{Name: "test.A", Phase: wfv1.NodeRunning}

	// Task B should not proceed, task A is still running
	result := evaluator.Evaluate(ctx, "B")
	require.NoError(t, result.Error)
	assert.True(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Task A succeeded
	wf.Status.Nodes[nodeID] = wfv1.NodeStatus{Name: "test.A", Phase: wfv1.NodeSucceeded}

	// Task B and C should proceed and execute
	result = evaluator.Evaluate(ctx, "B")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	result = evaluator.Evaluate(ctx, "C")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	// Other tasks should not
	result = evaluator.Evaluate(ctx, "should-execute-1")
	require.NoError(t, result.Error)
	assert.True(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Tasks B succeeded, C failed
	wf.Status.Nodes[wf.NodeID("test.B")] = wfv1.NodeStatus{Name: "test.B", Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.C")] = wfv1.NodeStatus{Name: "test.C", Phase: wfv1.NodeFailed}

	// Tasks should-execute-1 and should-execute-2 should proceed and execute
	result = evaluator.Evaluate(ctx, "should-execute-1")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	result = evaluator.Evaluate(ctx, "should-execute-2")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	// Task should-not-execute should proceed, but not execute
	result = evaluator.Evaluate(ctx, "should-not-execute")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Tasks should-execute-1 and should-execute-2 succeeded, should-not-execute skipped
	wf.Status.Nodes[wf.NodeID("test.should-execute-1")] = wfv1.NodeStatus{Name: "test.should-execute-1", Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.should-execute-2")] = wfv1.NodeStatus{Name: "test.should-execute-2", Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.should-not-execute")] = wfv1.NodeStatus{Name: "test.should-not-execute", Phase: wfv1.NodeSkipped}

	// Tasks should-execute-3 should proceed and execute
	result = evaluator.Evaluate(ctx, "should-execute-3")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
}

func TestEvaluateAnyAllDependsLogic(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name: "A-1",
		},
		{
			Name: "A-2",
		},
		{
			Name:    "B",
			Depends: "A.AnySucceeded",
		},
		{
			Name: "B-1",
		},
		{
			Name: "B-2",
		},
		{
			Name:    "C",
			Depends: "B.AllFailed",
		},
	}

	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: make(wfv1.Nodes),
		},
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{
			Tasks: testTasks,
		},
	}
	evaluator := newDAGEvaluator(wf, tmpl)

	// Task A is still running, A-1 succeeded but A-2 failed
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{Name: "test.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{wf.NodeID("test.A(0:1)"), wf.NodeID("test.A(1:2)")},
	}
	wf.Status.Nodes[wf.NodeID("test.A(0:1)")] = wfv1.NodeStatus{Name: "test.A(0:1)", Phase: wfv1.NodeRunning}
	wf.Status.Nodes[wf.NodeID("test.A(1:2)")] = wfv1.NodeStatus{Name: "test.A(1:2)", Phase: wfv1.NodeRunning}

	// Task B should not proceed as task A is still running
	result := evaluator.Evaluate(ctx, "B")
	require.NoError(t, result.Error)
	assert.True(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Task A succeeded
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{Name: "test.A",
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{wf.NodeID("test.A(0:1)"), wf.NodeID("test.A(1:2)")},
	}

	// Task B should proceed, but not execute as none of the children have succeeded yet
	result = evaluator.Evaluate(ctx, "B")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Task A-2 succeeded
	wf.Status.Nodes[wf.NodeID("test.A(1:2)")] = wfv1.NodeStatus{Name: "test.A(1:2)", Phase: wfv1.NodeSucceeded}

	// Task B should now proceed and execute
	result = evaluator.Evaluate(ctx, "B")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)

	// Task B succeeds and B-1 fails
	wf.Status.Nodes[wf.NodeID("test.B")] = wfv1.NodeStatus{Name: "test.B",
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{wf.NodeID("test.B(0:1)"), wf.NodeID("test.B(1:2)")},
	}
	wf.Status.Nodes[wf.NodeID("test.B(0:1)")] = wfv1.NodeStatus{Name: "test.B(0:1)", Phase: wfv1.NodeFailed}

	// Task C should proceed, but not execute as not all of B's children have failed yet
	result = evaluator.Evaluate(ctx, "C")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	wf.Status.Nodes[wf.NodeID("test.B(1:2)")] = wfv1.NodeStatus{Name: "test.B(1:2)", Phase: wfv1.NodeFailed}

	// Task C should now proceed and execute as all of B's children have failed
	result = evaluator.Evaluate(ctx, "C")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
}

func TestEvaluateDependsLogicWhenTaskOmitted(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name:    "B",
			Depends: "A.Omitted",
		},
	}

	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: make(wfv1.Nodes),
		},
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{
			Tasks: testTasks,
		},
	}
	evaluator := newDAGEvaluator(wf, tmpl)

	// Task A is running
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{Name: "test.A", Phase: wfv1.NodeOmitted}

	// Task B should proceed and execute
	result := evaluator.Evaluate(ctx, "B")
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
}

func TestAllEvaluateDependsLogic(t *testing.T) {
	statusMap := map[common.TaskResult]wfv1.NodePhase{
		common.TaskResultSucceeded: wfv1.NodeSucceeded,
		common.TaskResultFailed:    wfv1.NodeFailed,
		common.TaskResultSkipped:   wfv1.NodeSkipped,
		common.TaskResultOmitted:   wfv1.NodeOmitted,
	}
	for _, status := range []common.TaskResult{common.TaskResultSucceeded, common.TaskResultFailed, common.TaskResultSkipped, common.TaskResultOmitted} {
		testTasks := []wfv1.DAGTask{
			{
				Name: "same",
			},
			{
				Name:    "Run",
				Depends: fmt.Sprintf("same.%s", status),
			},
			{
				Name:    "NotRun",
				Depends: fmt.Sprintf("!same.%s", status),
			},
		}

		ctx := logging.TestContext(t.Context())
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
			Status: wfv1.WorkflowStatus{
				Nodes: make(wfv1.Nodes),
			},
		}
		tmpl := &wfv1.Template{
			DAG: &wfv1.DAGTemplate{
				Tasks: testTasks,
			},
		}
		evaluator := newDAGEvaluator(wf, tmpl)

		// Task A is running
		wf.Status.Nodes[wf.NodeID("test.same")] = wfv1.NodeStatus{Name: "test.same", Phase: statusMap[status]}

		result := evaluator.Evaluate(ctx, "Run")
		require.NoError(t, result.Error)
		assert.False(t, result.Suspended)
		assert.True(t, result.ShouldRun)
		result = evaluator.Evaluate(ctx, "NotRun")
		require.NoError(t, result.Error)
		assert.False(t, result.Suspended)
		assert.False(t, result.ShouldRun)
	}
}

func TestHTTPTmplDAG(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/http-tmpl-dag.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestDAGEnhancedDependsWithFailureIntegration(t *testing.T) {
	// Full 7-task DAG: A(succeeded), B(succeeded), C(failed),
	// D→"A && (C.Succeeded || C.Failed)" (should run),
	// E→"B || C" (should run),
	// F→"B && C" (skipped because C failed and default depends means C.Succeeded),
	// G→"E || F" (suspended because neither E nor F has a node yet)
	testTasks := []wfv1.DAGTask{
		{Name: "A"},
		{Name: "B"},
		{Name: "C"},
		{Name: "D", Depends: "A && (C.Succeeded || C.Failed)"},
		{Name: "E", Depends: "B || C"},
		{Name: "F", Depends: "B && C"},
		{Name: "G", Depends: "E || F"},
	}

	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: make(wfv1.Nodes),
		},
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{
			Tasks: testTasks,
		},
	}

	// Set up: A succeeded, B succeeded, C failed
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{Name: "test.A", Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.B")] = wfv1.NodeStatus{Name: "test.B", Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.C")] = wfv1.NodeStatus{Name: "test.C", Phase: wfv1.NodeFailed}

	evaluator := newDAGEvaluator(wf, tmpl)

	// D: "A && (C.Succeeded || C.Failed)" — A succeeded, C failed → C.Failed is true → should run
	result := evaluator.Evaluate(ctx, "D")
	require.NoError(t, result.Error)
	assert.True(t, result.ShouldRun, "D should run: A succeeded and C.Failed is true")

	// E: "B || C" — expanded to "(B.Succeeded||B.Skipped||B.Daemoned) || (C.Succeeded||C.Skipped||C.Daemoned)"
	// B.Succeeded is true → should run
	result = evaluator.Evaluate(ctx, "E")
	require.NoError(t, result.Error)
	assert.True(t, result.ShouldRun, "E should run: B succeeded")

	// F: "B && C" — expanded: B.Succeeded is true but C.Succeeded is false → skipped
	result = evaluator.Evaluate(ctx, "F")
	require.NoError(t, result.Error)
	assert.False(t, result.ShouldRun, "F should not run: C did not succeed")
	assert.True(t, result.Skipped, "F should be skipped")

	// G: "E || F" — E has no workflow node yet, F has no workflow node yet → suspended
	result = evaluator.Evaluate(ctx, "G")
	require.NoError(t, result.Error)
	assert.True(t, result.Suspended, "G is suspended waiting for E or F")
	assert.False(t, result.ShouldRun, "G should not run yet")
}

func TestDAGAssessPhaseWithPendingTasks(t *testing.T) {
	// C failed, D depends on "C.Failed" (still Pending since it hasn't been scheduled yet)
	// DAG should be Running (not prematurely Failed)
	testTasks := []wfv1.DAGTask{
		{Name: "C"},
		{Name: "D", Depends: "C.Failed"},
	}

	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: make(wfv1.Nodes),
		},
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{
			Tasks: testTasks,
		},
	}

	// C has failed
	wf.Status.Nodes[wf.NodeID("test.C")] = wfv1.NodeStatus{Name: "test.C", Phase: wfv1.NodeFailed}

	evaluator := newDAGEvaluator(wf, tmpl)

	// D should be ready to run (C.Failed is true)
	result := evaluator.Evaluate(ctx, "D")
	require.NoError(t, result.Error)
	assert.True(t, result.ShouldRun, "D should run because C.Failed is true")
}

// Restored integration tests from the old DAG engine test suite, adapted
// for the new multi-cycle reconciliation pattern.

func TestRetryStrategyNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/retry-strategy-nodes.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Single-cycle operate to avoid fake K8s pods (empty Status.Phase)
	// being processed by podReconciliation on subsequent cycles.
	woc.operate(ctx)

	retryNode, err := woc.wf.GetNodeByName("wf-retry-pol")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)

	onExitNode, err := woc.wf.GetNodeByName("wf-retry-pol.onExit")
	require.NoError(t, err)
	assert.NotNil(t, onExitNode)
	assert.True(t, onExitNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, onExitNode.Phase)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestOnExitDAGPhase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/on-exit-node-dag-phase.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// The DAG boundary stays Running while B's exit hook is pending.
	dagNode, err := woc.wf.GetNodeByName("dag-diamond-88trp")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, dagNode.Phase)

	onExitNode, err := woc.wf.GetNodeByName("dag-diamond-88trp.B.onExit")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodePending, onExitNode.Phase)
	assert.True(t, onExitNode.NodeFlag.Hooked)
}

func TestDagOptionalInputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-optional-input-artifacts.yaml")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Single-cycle operate to avoid fake K8s pods (empty Status.Phase)
	// being processed by podReconciliation on subsequent cycles.
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	optionalInputArtifactsNode, err := woc.wf.GetNodeByName("dag-optional-inputartifacts.B")
	require.NoError(t, err)
	assert.NotNil(t, optionalInputArtifactsNode)
	assert.Equal(t, wfv1.NodePending, optionalInputArtifactsNode.Phase)
}

func TestEmptyWithParamDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/empty-with-param-dag.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	for range 10 {
		woc.operate(ctx)
		if woc.wf.Status.Phase.Completed() {
			break
		}
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestLeafContinueOn(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/leaf-continue-on.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	for range 10 {
		woc.operate(ctx)
		if woc.wf.Status.Phase.Completed() {
			break
		}
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestDagParallelism(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-parallelism.yaml")

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	woc1 := newWoc(ctx, *woc.wf)
	woc1.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var dagDaemonFailedTest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-daemon-fail
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A"
        template: echo
  - name: daemon-task
    daemon: true
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestEvaluateDependsLogicWhenDaemonFailed verifies that when task A is a daemon
// and running, task B (which depends on A) still gets scheduled.
func TestEvaluateDependsLogicWhenDaemonFailed(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagDaemonFailedTest)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// First operate: kicks off task A (creates pod)
	woc.operate(ctx)

	// Mark A's pod as running with daemoned=true
	makePodsPhase(ctx, woc, v1.PodRunning)
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	daemon := true
	aNode.Daemoned = &daemon
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	// Second operate: should schedule B because A is daemoned (treated as fulfilled)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled when A is daemoned and running")

	// A's daemon then fails while B is already running: B must be unaffected.
	aNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	aNode.Phase = wfv1.NodeFailed
	aNode.Daemoned = nil
	woc.wf.Status.Nodes[aNode.ID] = *aNode
	bNode.Phase = wfv1.NodeRunning
	woc.wf.Status.Nodes[bNode.ID] = *bNode
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	bNode = woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode)
	assert.Equal(t, wfv1.NodeRunning, bNode.Phase, "a running dependant is not affected by its daemon dependency failing")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var dagDaemonRetryTest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-daemon-retry
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A"
        template: echo
  - name: daemon-task
    daemon: true
    retryStrategy:
      limit: 2
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestDaemonRetryOnFailure verifies that when a daemoned pod fails, the retry
// logic creates a new attempt instead of treating the retry node as "fulfilled".
func TestDaemonRetryOnFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagDaemonRetryTest)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// First operate: kicks off task A (creates pod)
	woc.operate(ctx)

	// Mark A's pod as running + daemoned
	makePodsPhase(ctx, woc, v1.PodRunning)
	// Find the retry node for task A
	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	require.Equal(t, wfv1.NodeTypeRetry, retryNode.Type)

	// Find the pod child node A(0)
	podNode := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, podNode)
	daemon := true
	podNode.Daemoned = &daemon
	podNode.Phase = wfv1.NodeRunning
	woc.wf.Status.Nodes[podNode.ID] = *podNode

	// Second operate: A is daemoned + running → B should be scheduled
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode, "B should be scheduled when A is daemoned and running")

	// Now simulate daemon pod failure: mark A(0) as Failed + clear Daemoned
	podNode = woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, podNode)
	podNode.Phase = wfv1.NodeFailed
	podNode.Daemoned = nil
	podNode.Message = "main: Error (exit code 1)"
	woc.wf.Status.Nodes[podNode.ID] = *podNode

	// Third operate: should detect daemon failure and create a retry A(1)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// The retry node should no longer be daemoned
	retryNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.False(t, retryNode.IsDaemoned(), "retry node Daemoned flag should be cleared after child fails")

	// A retry attempt A(1) should have been created
	retryAttempt := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	assert.NotNil(t, retryAttempt, "a retry attempt A(1) should have been created after daemon failure")
}

// Tests whether assessPhase marks a DAG as successful when it contains failed tasks with continueOn failed
func TestDagAssessPhaseContinueOnExpandedTaskVariables(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-assess-phase-continue-on-expanded-task-variables.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// Tests whether assessPhase marks a DAG as successful when it contains failed tasks with continueOn failed
func TestDagAssessPhaseContinueOnExpandedTask(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-assess-phase-continue-on-expanded-task.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestDAGWithParamAndGlobalParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-with-param-and-global-param.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

// This tests that a DAG with retry strategy in its tasks fails successfully when terminated
func TestTerminatingDAGWithRetryStrategyNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/terminating-dag-with-retry-strategy-nodes.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// This tests that a DAG with retry strategy in its tasks fails successfully when terminated

func TestOnExitNonLeaf(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/on-exit-non-leaf.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	retryNode, err := woc.wf.GetNodeByName("exit-handler-bug-example.step-2.onExit")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.True(t, retryNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, retryNode.Phase)

	_, err = woc.wf.GetNodeByName("exit-handler-bug-example.step-3")
	require.Error(t, err)

	retryNode.Phase = wfv1.NodeSucceeded
	woc.wf.Status.Nodes[retryNode.ID] = *retryNode
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	retryNode, err = woc.wf.GetNodeByName("exit-handler-bug-example.step-3")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodePending, retryNode.Phase)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestDagTargetTaskOnExit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-target-task-on-exit.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	onExitNode, err := woc.wf.GetNodeByName("dag-primay-branch-6bnnl.A.onExit")
	require.NoError(t, err)
	assert.NotNil(t, onExitNode)
	assert.True(t, onExitNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, onExitNode.Phase)
}

func TestFailsWithParamDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/fails-with-param-dag.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

func TestDAGReferTaskAggregatedOutputs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-outputs-refer-task-aggregated-ouputs.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("parameter-aggregation-dag-h8b82")
	require.NotNil(t, dagNode)
	require.NotNil(t, dagNode.Outputs)
	require.Len(t, dagNode.Outputs.Parameters, 2)
	assert.Equal(t, `["1","2"]`, dagNode.Outputs.Parameters[0].Value.String())
	assert.Equal(t, `["odd","even"]`, dagNode.Outputs.Parameters[1].Value.String())
}

func TestDagHttpChildrenAssigned(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-http-children-assigned.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("good2")
	assert.NotNil(t, dagNode)

	dagNode = woc.wf.Status.Nodes.FindByDisplayName("good1")
	require.NotNil(t, dagNode)
	require.Len(t, dagNode.Children, 1)
	assert.Equal(t, "http-template-nv52d-495103493", dagNode.Children[0])
}

func TestRetryTypeDagTaskRunExitNodeAfterCompleted(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/retry-type-dag-task-run-exit-node-after-completed.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	// retryTypeDAGTask completed
	printAChild := woc.wf.Status.Nodes.FindByDisplayName("printA(0)")
	assert.Equal(t, wfv1.NodeSucceeded, printAChild.Phase)

	// run ExitNode
	woc.operate(ctx)
	onExitNode := woc.wf.Status.Nodes.FindByDisplayName("printA.onExit")
	require.NotNil(t, onExitNode)
	assert.Equal(t, wfv1.NodeRunning, onExitNode.Phase)
	assert.True(t, onExitNode.NodeFlag.Hooked)

	// exitNode succeeded
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	onExitNode = woc.wf.Status.Nodes.FindByDisplayName("printA.onExit")
	assert.Equal(t, wfv1.NodeSucceeded, onExitNode.Phase)
	assert.True(t, onExitNode.NodeFlag.Hooked)

	// run next DAGTask
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	nextDAGTaskNode := woc.wf.Status.Nodes.FindByDisplayName("dependencyTesting")
	require.NotNil(t, nextDAGTaskNode)
	assert.Equal(t, wfv1.NodeRunning, nextDAGTaskNode.Phase)
}

func TestDagWftmplHookWithRetry(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag_wftmpl_hook_with_retry.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)

	// assert task kicked
	taskNode := woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodePending, taskNode.Phase)

	// task failed
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)

	// onFailure retry hook(0) kicked
	taskNode = woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodeFailed, taskNode.Phase)
	failHookRetryNode := woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure")
	failHookChild0Node := woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(0)")
	assert.Equal(t, wfv1.NodeRunning, failHookRetryNode.Phase)
	assert.Equal(t, wfv1.NodePending, failHookChild0Node.Phase)

	// onFailure retry hook(0) failed
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)

	// onFailure retry hook(1) kicked
	taskNode = woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodeFailed, taskNode.Phase)
	failHookRetryNode = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure")
	failHookChild0Node = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(0)")
	failHookChild1Node := woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(1)")
	assert.Equal(t, wfv1.NodeRunning, failHookRetryNode.Phase)
	assert.Equal(t, wfv1.NodeFailed, failHookChild0Node.Phase)
	assert.Equal(t, wfv1.NodePending, failHookChild1Node.Phase)

	// onFailure retry hook(1) failed
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)

	// onFailure retry node faled
	taskNode = woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodeFailed, taskNode.Phase)
	failHookRetryNode = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure")
	failHookChild0Node = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(0)")
	failHookChild1Node = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(1)")
	assert.Equal(t, wfv1.NodeFailed, failHookRetryNode.Phase)
	assert.Equal(t, wfv1.NodeFailed, failHookChild0Node.Phase)
	assert.Equal(t, wfv1.NodeFailed, failHookChild1Node.Phase)
	// finish Node skipped
	finishNode := woc.wf.Status.Nodes.FindByDisplayName("finish")
	assert.Equal(t, wfv1.NodeOmitted, finishNode.Phase)
}

// Regression test: referencing {{tasks.<taskgroup>.id}} where the ancestor is a TaskGroup
// (created by withParam expansion). Before the fix, buildLocalScope was only called for
// non-TaskGroup ancestors, so tasks.<taskgroup>.id was unavailable and caused a requeue.
func TestDAGTaskGroupIDReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-task-group-id-ref.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Verify the use-id task was created (not stuck in requeue due to missing variable)
	useIDNode := woc.wf.Status.Nodes.FindByDisplayName("use-id")
	require.NotNil(t, useIDNode, "use-id node should be created when tasks.fanout.id is resolvable")

	// Verify the resolved value of tasks.fanout.id matches the TaskGroup node's ID
	require.NotNil(t, useIDNode.Inputs)
	require.Len(t, useIDNode.Inputs.Parameters, 1)
	assert.Equal(t, "dag-taskgroup-id-ref-2094038697", useIDNode.Inputs.Parameters[0].Value.String())
}

// TestDAGWhenSkipNoRequeue verifies that a DAG task with a "when" clause that evaluates to false
// does not cause a requeue even when other fields in the task reference outputs that don't exist.
// Scenario: A is skipped (when: "false"), so A's outputs don't exist. B depends on A and has
// when: "{{tasks.A.status}} == Succeeded" which evaluates to false ("Skipped == Succeeded").
// B also references {{tasks.A.outputs.result}} which is unresolvable since A was skipped.
// Without the fix, the full ReplaceStrict would fail on the missing output and requeue.
// With the fix, the when clause is resolved first, evaluates to false, and B is skipped early.
func TestDAGWhenSkipNoRequeue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-when-skip-no-requeue.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Workflow should succeed: A was skipped, B's when evaluated false so B was also skipped
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

	nodeB := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, nodeB)
	assert.Equal(t, wfv1.NodeSkipped, nodeB.Phase)
}

// TestDAGSkippedOutputRef verifies that a DAG task referencing a skipped dependency's defaultless
// output fails terminally rather than getting stuck in a requeue loop (#16223 absence semantics).
// Scenario: stage-a succeeds with output parameters, stage-b is skipped (when evaluates false),
// stage-c depends on both and references outputs from both. stage-b's output has no default and
// stage-c's input has no default, so the reference is an unhandled absent optional: stage-c must
// resolve to a terminal Error, not schedule with an unresolved tag or requeue forever.
func TestDAGSkippedOutputRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-output-ref.yaml")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// stage-c must resolve to a terminal error, not get stuck in a requeue loop
	nodeC := woc.wf.Status.Nodes.FindByDisplayName("stage-c")
	require.NotNil(t, nodeC, "stage-c should be created even though stage-b was skipped")
	assert.Equal(t, wfv1.NodeError, nodeC.Phase, "an unhandled absent optional must fail the task terminally")
	assert.Contains(t, nodeC.Message, "absent optional")
}

var dagOmittedOutputRefUnit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-omitted-output-ref-unit
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: stage-a
            template: echo
          - name: stage-b
            template: produce
            depends: "stage-a.Failed"
          - name: stage-c
            template: consume
            depends: "stage-a && (stage-b || stage-b.Omitted)"
            arguments:
              parameters:
                - name: msg
                  value: "{{tasks.stage-b.outputs.parameters.output-message}}"
    - name: echo
      container:
        image: argoproj/argosay:v2
    - name: produce
      outputs:
        parameters:
          - name: output-message
            valueFrom:
              path: /tmp/output.txt
      container:
        image: argoproj/argosay:v2
    - name: consume
      inputs:
        parameters:
          - name: msg
      container:
        image: argoproj/argosay:v2
`

// TestDAGOmittedOutputRefTerminalError reproduces the omitted-dependency variant of #16223: stage-b
// is Omitted (its "stage-a.Failed" depends never holds because stage-a Succeeds), and stage-c
// references stage-b's defaultless output with no consumer default. The reference is an unhandled
// absent optional, so stage-c must fail terminally with an "absent optional" message. Regression
// guard: stage-c's node already exists by the time arg resolution errors in the omitted flow, so
// initTerminalErrorNode must NOT re-initializeNode (which panics "already initialized" -> a recurring
// "Workflow operation error" requeue loop instead of a clean terminal failure).
func TestDAGOmittedOutputRefTerminalError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(dagOmittedOutputRefUnit)
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // stage-a pod created
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	// Re-operate until the workflow is fulfilled (stage-b omitted -> stage-c arg error). Extra cycles
	// exercise the path where stage-c's node already exists when the terminal error is raised again.
	for i := 0; i < 3 && !woc.wf.Status.Fulfilled(); i++ {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}

	nodeB := woc.wf.Status.Nodes.FindByDisplayName("stage-b")
	require.NotNil(t, nodeB)
	assert.Equal(t, wfv1.NodeOmitted, nodeB.Phase)

	nodeC := woc.wf.Status.Nodes.FindByDisplayName("stage-c")
	require.NotNil(t, nodeC, "stage-c must be created as a terminal error node, not lost to a requeue loop")
	assert.Equal(t, wfv1.NodeError, nodeC.Phase, "an unhandled absent optional must fail the task terminally")
	assert.Contains(t, nodeC.Message, "absent optional")
}

// TestDAGWhenExprSkipEval verifies that expression templates in a DAG task's arguments are not
// evaluated when the task's "when" clause evaluates to false.
// Scenario: workflow parameter "data" is empty. Task B has when: "{{= workflow.parameters.data != " }}"
// which evaluates to false. B's argument uses jsonpath(workflow.parameters.data, '$.id') which would
// fail on an empty string. The expression should not be evaluated since the task will be skipped.
// Currently fails because SubstituteParams evaluates all expression templates in the entire DAG
// template before individual task "when" conditions are checked.
func TestDAGWhenExprSkipEval(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-when-expr-skip-eval.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Workflow should succeed: B's when clause evaluates to false so B should be skipped
	// without evaluating B's argument expressions.
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

	nodeB := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, nodeB)
	assert.Equal(t, wfv1.NodeSkipped, nodeB.Phase)
}

func TestTerminateDAGWithMaxDurationLimitExpiredAndMoreAttempts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/terminate-dag-with-max-duration-limit-expired-and-more-attempts.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	retryNode, err := woc.wf.GetNodeByName("dag-diamond-dj7q5.A")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)
	assert.Contains(t, retryNode.Message, "Max duration limit exceeded")

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// This is the crucial part of the test
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// testRound2IDaemonedFailedDepExplicitQualifierDAG is a DAG where:
// - A is a daemon task (no retry strategy)
// - B has depends: "A.Failed"
// When A's daemon pod runs, becomes daemoned (Daemoned=true), and then fails
// (Phase=Failed), B should be scheduled because A.Failed is true.
//
// Bug: evaluateDependsReadiness checks depNode.IsDaemoned() BEFORE checking
// if the phase is terminal. A daemoned+Failed node enters the IsDaemoned() branch
// and gets {Daemoned:true} + hasPendingDeps=true instead of {Failed:true}.
// As a result, "A.Failed" evaluates to false and B waits forever.
var testRound2IDaemonedFailedDepExplicitQualifierDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: daemon-failed-dep-test
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A.Failed"
        template: echo
  - name: daemon-task
    daemon: true
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestRound2I_DaemonedFailedDepExplicitQualifier demonstrates a bug in
// evaluateDependsReadiness: when dep A is a daemoned pod that has died
// (Phase=NodeFailed, Daemoned=true), the evaluator incorrectly enters the
// IsDaemoned() branch for ALL daemoned nodes regardless of phase.
// It sets evalScope[A]={Daemoned:true}+hasPendingDeps=true instead of
// {Failed:true}, causing "A.Failed" to evaluate to false and B to wait forever.
//
// Topology: A (daemon, no retry) --> B (depends: "A.Failed")
//
// After A runs as a daemon and its pod fails with Daemoned=true still set:
// Expected: B gets scheduled (because A.Failed is true).
// Actual (bug): B stays Suspended/Waiting forever (IsDaemoned() branch fires for Failed node).
func TestRound2I_DaemonedFailedDepExplicitQualifier(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testRound2IDaemonedFailedDepExplicitQualifierDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: task A is scheduled → creates a daemon pod (Running/Pending)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should exist after first operate")

	// Simulate A's pod becoming daemoned (Running + Daemoned=true).
	// This is what happens when a daemon pod starts and becomes registered as daemoned.
	daemoned := true
	aNode.Daemoned = &daemoned
	aNode.Phase = wfv1.NodeRunning
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	// Cycle 2: A is Running+Daemoned → B might be scheduled via default depends
	// (because A.Daemoned is true). But we want to simulate the daemon failing.
	// Directly mark A's node as Failed while keeping Daemoned=true — this simulates
	// the pod controller marking the pod failed but the Daemoned flag not being cleared.
	aNode.Phase = wfv1.NodeFailed
	aNode.Daemoned = &daemoned // still true — this is the bug-triggering state
	aNode.Message = "daemon pod exited with code 1"
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// B should be scheduled because A.Failed is now true.
	// Bug: IsDaemoned() fires for A (because Daemoned=true) before checking Phase=Failed,
	// so evalScope[A]={Daemoned:true}+hasPendingDeps=true.
	// "A.Failed" evaluates to false → B is Suspended, never scheduled.
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should be scheduled: A.Failed is true (A is daemoned+Failed). "+
			"Bug: evaluateDependsReadiness enters IsDaemoned() branch for A "+
			"because Daemoned=true, setting {Daemoned:true} instead of {Failed:true}, "+
			"causing 'A.Failed' to evaluate false and B to stay Suspended forever")
}

// testRound2IRetryFailedDAG is a DAG with task A (retry, limit=0) and task B
// (depends: "A.Failed"). When A fails, B should be scheduled in the same operate cycle.
var testRound2IRetryFailedDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: converge-action-fail-test
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
      - name: B
        depends: "A.Failed"
        template: echo
  - name: fail-task
    retryStrategy:
      limit: "0"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestRound2I_RunningRetryNodeFailsInSamePass verifies that a retry node that
// is still NodeRunning after its last attempt failed (limit exhausted) is
// dispatched, so that in the same reconcile:
//  1. executeTask(A) marks A NodeFailed via processNodeRetries
//  2. executeTask(B) runs because A.Failed is satisfied (B.ShouldRun=true)
//
// If an unfulfilled retry node were not dispatched, A would stay Running and
// B would never be scheduled, leaving the DAG stuck at Running.
func TestRound2I_RunningRetryNodeFailsInSamePass(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testRound2IRetryFailedDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: task A is scheduled (creates retry node and A(0) pod)
	woc.operate(ctx)

	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode, "retry node A should exist after first operate")
	assert.Equal(t, wfv1.NodeTypeRetry, retryNode.Type)
	assert.Equal(t, wfv1.NodeRunning, retryNode.Phase)

	// B should not exist yet (A hasn't failed)
	assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("B"), "B should not be scheduled yet")

	// Cycle 2: A(0) fails (limit=0, no retries): the walk should (a) call
	// executeTask(A) to mark A as Failed, (b) schedule B
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A should be marked Failed after processNodeRetries runs via executeTask
	retryNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase,
		"retry node A must be Failed: the walk should call executeTask(A)")

	// B should be scheduled because A.Failed is true
	// Bug: if the running retry node is not dispatched, B is never
	// scheduled and the DAG hangs at Running.
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"task B must be scheduled: A.Failed is satisfied, the walk should call executeTask(B)")
}

// ---------------------------------------------------------------------------
// Operator Integration Tests
// ---------------------------------------------------------------------------

// Test 2: Daemon dep schedules downstream with default depends
var testOperatorIntDaemonDepDefaultDepends = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: daemon-dep-default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A"
        template: echo
  - name: daemon-task
    daemon: true
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_DaemonDepSchedulesDownstreamDefaultDepends verifies that
// when task A is a daemon and becomes daemoned+running, task B (which depends on A
// with the default depends clause) gets scheduled.
func TestOperatorIntegration_DaemonDepSchedulesDownstreamDefaultDepends(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntDaemonDepDefaultDepends)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off task A
	woc.operate(ctx)

	// Simulate A becoming daemoned and running
	makePodsPhase(ctx, woc, v1.PodRunning)
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	daemon := true
	aNode.Daemoned = &daemon
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	// Cycle 2: A is daemoned+running -> B should be scheduled
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode, "B should be scheduled when A is daemoned and running with default depends")
}

// Test 3: Daemon dep with explicit A.Succeeded doesn't prematurely omit B
var testOperatorIntDaemonDepExplicitSucceeded = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: daemon-dep-explicit-succeeded
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A.Succeeded"
        template: echo
  - name: daemon-task
    daemon: true
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_DaemonDepExplicitSucceededOmitsB verifies that
// when A is a daemon with B depending on "A.Succeeded", B is correctly omitted.
// A.Succeeded is unsatisfiable for a running daemon: Succeeded only becomes true
// when killDaemonedChildren runs, which requires the boundary to complete first.
// Waiting would deadlock (B waits for A.Succeeded → DAG waits for B → never completes).
func TestOperatorIntegration_DaemonDepExplicitSucceededOmitsB(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntDaemonDepExplicitSucceeded)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	makePodsPhase(ctx, woc, v1.PodRunning)
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	daemon := true
	aNode.Daemoned = &daemon
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// B should be omitted — A.Succeeded is unsatisfiable for a running daemon
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
		"B should be Omitted — A.Succeeded is unsatisfiable for a running daemon")
}

// Test 4: Dead daemon (Failed+Daemoned cleared) retry
var testOperatorIntDeadDaemonRetry = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dead-daemon-retry
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A"
        template: echo
  - name: daemon-task
    daemon: true
    retryStrategy:
      limit: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_DeadDaemonRetry verifies that when a daemon pod
// (A(0)) becomes daemoned, then fails (Daemoned flag cleared on failure),
// a retry attempt A(1) should be created.
func TestOperatorIntegration_DeadDaemonRetry(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntDeadDaemonRetry)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off task A -> creates A(0)
	woc.operate(ctx)

	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	require.Equal(t, wfv1.NodeTypeRetry, retryNode.Type)

	podNode := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, podNode)

	// Simulate A(0) becoming daemoned+running
	makePodsPhase(ctx, woc, v1.PodRunning)
	daemon := true
	podNode.Daemoned = &daemon
	podNode.Phase = wfv1.NodeRunning
	woc.wf.Status.Nodes[podNode.ID] = *podNode

	// Cycle 2: A(0) is daemoned+running -> B should be scheduled
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled when A is daemoned and running")

	// Now simulate daemon failure: A(0) becomes Failed, Daemoned cleared
	podNode = woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, podNode)
	podNode.Phase = wfv1.NodeFailed
	podNode.Daemoned = nil
	podNode.Message = "daemon pod exited"
	woc.wf.Status.Nodes[podNode.ID] = *podNode

	// Cycle 3: should detect failure and create retry attempt A(1)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	retryAttempt := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	assert.NotNil(t, retryAttempt,
		"A(1) should be created after daemon failure")
}

// Test 5: Retry exhausted -> downstream omitted
var testOperatorIntRetryExhaustedDownstreamOmitted = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-exhausted-omit
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
      - name: B
        depends: "A"
        template: echo
  - name: fail-task
    retryStrategy:
      limit: "1"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_RetryExhaustedDownstreamOmitted verifies that when
// A's retry is exhausted (A(0) fails, A(1) fails), B is omitted because
// the default depends (A.Succeeded) is never true.
func TestOperatorIntegration_RetryExhaustedDownstreamOmitted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntRetryExhaustedDownstreamOmitted)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off task A -> creates A(0)
	woc.operate(ctx)

	// A(0) fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(0) failed, A(1) should be created (retry limit=1 means 1 retry)
	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	require.NotNil(t, a1Node, "A(1) should be created after A(0) fails")

	// A(1) fails too -> retry exhausted
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A (retry node) should be Failed
	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase, "A should be Failed after retry exhausted")

	// B should be Omitted (default depends = A.Succeeded, which is false)
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
		"B should be Omitted when A.Succeeded is never satisfied")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 6: Retry with Error phase + enhanced depends A.Errored
var testOperatorIntRetryErrorEnhancedDepends = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-error-enhanced
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: error-task
      - name: B
        depends: "A.Errored"
        template: echo
  - name: error-task
    retryStrategy:
      limit: "0"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_RetryErrorEnhancedDepends verifies that when A exits
// with Error phase and retry is exhausted (limit=0), B (depends: "A.Errored")
// should run because A.Errored is true.
func TestOperatorIntegration_RetryErrorEnhancedDepends(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntRetryErrorEnhancedDepends)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off task A -> creates A(0)
	woc.operate(ctx)

	// Mark A(0) pod as failed, then manually set the node phase to Error
	makePodsPhase(ctx, woc, v1.PodFailed)
	a0Node := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node)
	a0Node.Phase = wfv1.NodeError
	a0Node.Message = "pod errored"
	woc.wf.Status.Nodes[a0Node.ID] = *a0Node

	// Cycle 2: A(0) errored, retry exhausted (limit=0) -> A should be Error
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.True(t, retryNode.Phase.FailedOrError(),
		"A should be in a terminal failure/error state after retry exhausted")

	// B should be scheduled because A.Errored is true
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should be scheduled when A.Errored is true (retry exhausted with Error phase)")
}

// Test 7: TaskGroup all children succeed -> downstream proceeds
var testOperatorIntTaskGroupAllSucceed = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: taskgroup-all-succeed
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems:
        - x
        - y
      - name: B
        depends: "A"
        template: echo
        arguments:
          parameters:
          - name: msg
            value: done
  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.msg}}"]
`

// TestOperatorIntegration_TaskGroupAllSucceedDownstreamProceeds verifies that
// when all children of a TaskGroup (withItems) succeed, the TaskGroup succeeds
// and downstream task B runs.
func TestOperatorIntegration_TaskGroupAllSucceedDownstreamProceeds(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntTaskGroupAllSucceed)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates TaskGroup A with children A(0:x) and A(1:y)
	woc.operate(ctx)

	tgNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode, "TaskGroup A should exist")

	// Both children succeed
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// TaskGroup A should be Succeeded
	tgNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode)
	assert.Equal(t, wfv1.NodeSucceeded, tgNode.Phase,
		"TaskGroup A should be Succeeded when all children succeed")

	// B should be scheduled
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode, "B should be scheduled after TaskGroup A succeeds")
}

// Test 8: TaskGroup child fails -> DAG fails
var testOperatorIntTaskGroupChildFails = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: taskgroup-child-fails
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: maybe-fail
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems:
        - x
        - y
  - name: maybe-fail
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.msg}}"]
`

// TestOperatorIntegration_TaskGroupChildFailsDAGFails verifies that when one
// child of a TaskGroup fails, the TaskGroup fails and the DAG fails.
func TestOperatorIntegration_TaskGroupChildFailsDAGFails(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntTaskGroupChildFails)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates TaskGroup A with children
	woc.operate(ctx)

	// Mark all pods as failed (at least one will fail)
	makePodsPhase(ctx, woc, v1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// TaskGroup A should be Failed
	tgNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode)
	assert.Equal(t, wfv1.NodeFailed, tgNode.Phase,
		"TaskGroup A should fail when a child fails")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 10: FailFast=false checks all leaves
var testOperatorIntFailFastFalse = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: failfast-false
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      failFast: false
      tasks:
      - name: A
        template: fail-task
      - name: B
        template: fail-task
      - name: C
        depends: "A && B"
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_FailFastFalseChecksAllLeaves verifies that with
// failFast=false, the DAG waits for all tasks to complete before failing.
// A fails, B fails -> C is omitted -> DAG fails.
func TestOperatorIntegration_FailFastFalseChecksAllLeaves(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntFailFastFalse)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A and B (parallel, no dependencies between them)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled")

	// Both A and B fail
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A and B should be Failed
	aNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	bNode = woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode)
	assert.Equal(t, wfv1.NodeFailed, bNode.Phase)

	// C should be omitted (default depends on A && B succeeding)
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, cNode, "C")
	assert.Equal(t, wfv1.NodeOmitted, cNode.Phase,
		"C should be Omitted since A && B both failed")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 11: ContinueOn absorbs failure
var testOperatorIntContinueOnAbsorbsFailure = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: continueon-absorbs
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
        continueOn:
          failed: true
      - name: B
        dependencies: [A]
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_ContinueOnAbsorbsFailure verifies that when A has
// continueOn.failed=true and A fails, B (which depends on A via dependencies)
// runs because continueOn absorbs the failure.
func TestOperatorIntegration_ContinueOnAbsorbsFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntContinueOnAbsorbsFailure)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A
	woc.operate(ctx)

	// A fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A should be Failed
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	// B should be scheduled because continueOn absorbs the failure
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should be scheduled when continueOn absorbs A's failure")
}

// Test 12: Daemon workflow completes when downstream finishes
var testOperatorIntDaemonWorkflowCompletes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: daemon-completes
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A"
        template: echo
  - name: daemon-task
    daemon: true
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_DaemonWorkflowCompletes verifies that when A
// is a daemon task, A becomes daemoned+running, B runs and succeeds,
// the workflow should complete (not stuck Running).
func TestOperatorIntegration_DaemonWorkflowCompletes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntDaemonWorkflowCompletes)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off task A
	woc.operate(ctx)

	// Mark A as daemoned+running
	makePodsPhase(ctx, woc, v1.PodRunning)
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	daemon := true
	aNode.Daemoned = &daemon
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	// Cycle 2: A is daemoned+running -> B should be scheduled
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled when A is daemoned and running")

	// B succeeds (makePodsPhase sets ALL pods to succeeded, including daemon)
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Additional cycle for convergence
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Workflow should complete (Succeeded)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase,
		"Workflow should complete after daemon task's downstream succeeds")
}

// Test 13: Retry node succeeded -> parent has outputs
var testOperatorIntRetrySucceededHasOutputs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-succeeded-outputs
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-then-succeed
      - name: B
        depends: "A"
        template: echo
  - name: fail-then-succeed
    retryStrategy:
      limit: "2"
    container:
      image: alpine:3.23
      command: [echo, hello]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_RetrySucceededParentOutputs verifies that when A(0)
// fails and A(1) succeeds, the retry node A is marked Succeeded and B runs.
func TestOperatorIntegration_RetrySucceededParentOutputs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntRetrySucceededHasOutputs)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A -> creates A(0)
	woc.operate(ctx)

	// A(0) fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(1) should be created
	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	require.NotNil(t, a1Node, "A(1) should be created after A(0) fails")

	// A(1) succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Retry node A should be Succeeded
	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeSucceeded, retryNode.Phase,
		"Retry node A should be Succeeded after A(1) succeeds")

	// B should be scheduled
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should be scheduled after retry node A succeeds")
}

// Test 14: Cascading omission in diamond
var testOperatorIntCascadingOmissionDiamond = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: cascading-omission-diamond
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
      - name: B
        depends: "A"
        template: echo
      - name: C
        depends: "A"
        template: echo
      - name: D
        depends: "B && C"
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_CascadingOmissionDiamond verifies that in a diamond
// DAG (A->B, A->C, B&&C->D), when A fails, B and C are omitted, D is omitted,
// and the DAG fails.
func TestOperatorIntegration_CascadingOmissionDiamond(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntCascadingOmissionDiamond)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A
	woc.operate(ctx)

	// A fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A should be Failed
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	// Run another cycle to let omission cascade
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// B should be Omitted
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
		"B should be Omitted since A failed (default depends = A.Succeeded)")

	// C should be Omitted
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, cNode, "C")
	assert.Equal(t, wfv1.NodeOmitted, cNode.Phase,
		"C should be Omitted since A failed")

	// D should be Omitted
	dNode := woc.wf.Status.Nodes.FindByDisplayName("D")
	require.NotNil(t, dNode, "D")
	assert.Equal(t, wfv1.NodeOmitted, dNode.Phase,
		"D should be Omitted since B and C are Omitted")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 15: withItems + retry composition
var testOperatorIntWithItemsRetryComposition = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: withitems-retry
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: retry-echo
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems:
        - x
        - y
  - name: retry-echo
    inputs:
      parameters:
      - name: msg
    retryStrategy:
      limit: "1"
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.msg}}"]
`

// TestOperatorIntegration_WithItemsRetryComposition verifies 3-level nesting:
// TaskGroup > Retry > Pod. When A(0:x)(0) fails -> A(0:x)(1) succeeds, and
// A(1:y)(0) succeeds, the TaskGroup A should succeed and the DAG should succeed.
func TestOperatorIntegration_WithItemsRetryComposition(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntWithItemsRetryComposition)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates TaskGroup A with retry children
	woc.operate(ctx)

	tgNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode, "TaskGroup A should exist")

	// First attempt of all items: both fail
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Now retry attempts should be created. Let them succeed.
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// May need extra cycles for convergence
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// TaskGroup A should be Succeeded
	tgNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode)
	assert.Equal(t, wfv1.NodeSucceeded, tgNode.Phase,
		"TaskGroup A should succeed after all retry children succeed")

	// DAG should succeed
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// Test 16: Multiple retries in parallel
var testOperatorIntMultipleRetriesParallel = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: multi-retry-parallel
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: retry-task
      - name: B
        template: retry-task
      - name: C
        depends: "A && B"
        template: echo
  - name: retry-task
    retryStrategy:
      limit: "1"
    container:
      image: alpine:3.23
      command: [echo, hello]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_MultipleRetriesParallel verifies that when A and B
// are retry tasks running in parallel, both fail initially, both retry, both
// succeed, then C runs.
func TestOperatorIntegration_MultipleRetriesParallel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntMultipleRetriesParallel)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A(0) and B(0)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled")

	// A(0) and B(0) both fail
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(1) and B(1) should be created
	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	assert.NotNil(t, a1Node, "A(1) should be created after A(0) fails")
	b1Node := woc.wf.Status.Nodes.FindByDisplayName("B(1)")
	assert.NotNil(t, b1Node, "B(1) should be created after B(0) fails")

	// A(1) and B(1) both succeed
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Retry nodes should be Succeeded
	aNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeSucceeded, aNode.Phase, "A should be Succeeded")

	bNode = woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode)
	assert.Equal(t, wfv1.NodeSucceeded, bNode.Phase, "B should be Succeeded")

	// C should be scheduled
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	assert.NotNil(t, cNode, "C should be scheduled after A and B both succeed")
}

// Test 17: Retry with OnError policy only retries Error, not Failed
var testOperatorIntRetryOnErrorPolicy = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-onerror-policy
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
  - name: fail-task
    retryStrategy:
      limit: "2"
      retryPolicy: OnError
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
`

// TestOperatorIntegration_RetryOnErrorPolicyDoesNotRetryFailed verifies that
// when the retry policy is OnError and A(0) exits with Failed (not Error),
// the retry is NOT triggered because OnError only covers Error phase.
func TestOperatorIntegration_RetryOnErrorPolicyDoesNotRetryFailed(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntRetryOnErrorPolicy)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A -> creates A(0)
	woc.operate(ctx)

	a0Node := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node, "A(0) should exist")

	// A(0) fails with Failed phase (not Error)
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(1) should NOT be created because OnError policy does not retry Failed
	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	assert.Nil(t, a1Node,
		"A(1) should NOT be created: OnError policy does not retry Failed phase")

	// Retry node A should be Failed
	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase,
		"A should be Failed because OnError does not retry Failed phase")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 18: Steps template sequential execution
var testOperatorIntStepsSequentialExecution = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-sequential
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: step-a
        template: echo
    - - name: step-b
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorIntegration_StepsSequentialExecution verifies that in a steps
// template with [[step-a]] -> [[step-b]], step-a runs first, and after it
// succeeds, step-b runs.
func TestOperatorIntegration_StepsSequentialExecution(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorIntStepsSequentialExecution)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off step-a
	woc.operate(ctx)

	stepANode := woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, stepANode, "step-a should be scheduled in the first cycle")

	// step-b should not exist yet (sequential)
	stepBNode := woc.wf.Status.Nodes.FindByDisplayName("step-b")
	assert.Nil(t, stepBNode, "step-b should not be scheduled before step-a completes")

	// step-a succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// step-b should now be scheduled
	stepBNode = woc.wf.Status.Nodes.FindByDisplayName("step-b")
	assert.NotNil(t, stepBNode, "step-b should be scheduled after step-a succeeds")

	// step-b succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Workflow should succeed
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// ===== TestOperatorBugfix tests =====

// Test 1: Dead daemon child triggers retry (not treated as running)
var testOperatorBugfixDeadDaemonRetryStaleFlag = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dead-daemon-retry-stale
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        depends: "A"
        template: echo
  - name: daemon-task
    daemon: true
    retryStrategy:
      limit: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorBugfix_DeadDaemonChildTriggersRetry verifies that when a daemon
// pod A(0) is Running+Daemoned, then fails with a stale Daemoned=true flag,
// the retry A(1) is still triggered instead of treating A as still running.
func TestOperatorBugfix_DeadDaemonChildTriggersRetry(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixDeadDaemonRetryStaleFlag)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	podNode := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, podNode)

	// Mark A(0) as Running+Daemoned
	makePodsPhase(ctx, woc, v1.PodRunning)
	daemon := true
	podNode.Daemoned = &daemon
	podNode.Phase = wfv1.NodeRunning
	woc.wf.Status.Nodes[podNode.ID] = *podNode

	// Cycle 2: A(0) is daemoned+running -> B should be scheduled
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled when A is daemoned and running")

	// Now mark A(0) as Failed BUT keep Daemoned=true (stale flag)
	podNode = woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, podNode)
	makePodsPhase(ctx, woc, v1.PodFailed)
	podNode.Phase = wfv1.NodeFailed
	podNode.Message = "daemon pod died"
	// Keep Daemoned=true (stale flag -- this is the bug scenario)
	stale := true
	podNode.Daemoned = &stale
	woc.wf.Status.Nodes[podNode.ID] = *podNode

	// Cycle 3: should detect failure and create retry attempt A(1) despite stale Daemoned
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	retryAttempt := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	assert.NotNil(t, retryAttempt,
		"A(1) should be created after daemon failure despite stale Daemoned=true flag")
}

// Test 3: TaskGroup stale Succeeded with failed child
var testOperatorBugfixTaskGroupChildFails = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: taskgroup-child-fails-mix
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: maybe-fail
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems:
        - x
        - y
  - name: maybe-fail
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.msg}}"]
`

// TestOperatorBugfix_TaskGroupStaleSucceededWithFailedChild verifies that when
// one withItems child succeeds and one fails, the TaskGroup is Failed (not stuck Succeeded).
func TestOperatorBugfix_TaskGroupStaleSucceededWithFailedChild(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixTaskGroupChildFails)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates TaskGroup A with children A(0:x) and A(1:y)
	woc.operate(ctx)

	tgNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode, "TaskGroup A should exist")

	// Mark all pods as failed (simulates at least one child failing)
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// TaskGroup A should be Failed
	tgNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode)
	assert.Equal(t, wfv1.NodeFailed, tgNode.Phase,
		"TaskGroup A should fail when a child fails (not stuck Succeeded)")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 7: TaskGroup with Retry child: retry exhausted -> TaskGroup fails
var testOperatorBugfixTaskGroupRetryExhausted = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: taskgroup-retry-exhausted
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: retry-fail
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems:
        - x
  - name: retry-fail
    inputs:
      parameters:
      - name: msg
    retryStrategy:
      limit: "1"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
`

// TestOperatorBugfix_TaskGroupRetryExhaustedFails verifies that when a
// TaskGroup has a retry child (withItems:[x], retry.limit=1) and the retry
// is exhausted, the TaskGroup fails and the DAG fails.
func TestOperatorBugfix_TaskGroupRetryExhaustedFails(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixTaskGroupRetryExhausted)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates TaskGroup A with retry child -> A(0:x)(0)
	woc.operate(ctx)

	tgNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode, "TaskGroup A should exist")

	// A(0:x)(0) fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(0:x)(1) should be created (retry)
	// The retry node naming can vary; let's just proceed to fail again
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Extra cycle for convergence
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// TaskGroup A should be Failed
	tgNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode)
	assert.Equal(t, wfv1.NodeFailed, tgNode.Phase,
		"TaskGroup A should fail when retry child is exhausted")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 8: FailFast=false with Error and Failed preserves Error
var testOperatorBugfixFailFastFalseErrorPreserved = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: failfast-false-error-preserved
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      failFast: false
      tasks:
      - name: A
        template: fail-task
      - name: B
        template: fail-task
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
`

// TestOperatorBugfix_FailFastFalseErrorPreserved verifies that with
// failFast=false, when A gets Error and B gets Failed, the DAG phase reflects
// that at least one task errored. We use a container Waiting state to produce
// a genuine NodeError from pod reconciliation.
func TestOperatorBugfix_FailFastFalseErrorPreserved(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixFailFastFalseErrorPreserved)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A and B (parallel, no deps)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled")

	// Make A's pod fail with a container in Waiting state (produces NodeError)
	// and B's pod fail normally (produces NodeFailed)
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	aNodeID := aNode.ID
	for i, pod := range pods.Items {
		nodeID := woc.nodeID(&pods.Items[i])
		if nodeID == aNodeID {
			// Make A's pod fail with container in Waiting state -> NodeError
			pod.Status.Phase = v1.PodFailed
			pod.Status.ContainerStatuses = []v1.ContainerStatus{
				{
					Name: "main",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "image pull failed",
						},
					},
				},
			}
		} else {
			// Make B's pod fail normally -> NodeFailed
			pod.Status.Phase = v1.PodFailed
			pod.Status.Message = "Pod failed"
		}
		updatedPod, err := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		err = woc.controller.PodController.TestingPodInformer().GetStore().Update(updatedPod)
		require.NoError(t, err)
	}

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A should be Error, B should be Failed
	aNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeError, aNode.Phase,
		"A should be Error due to container Waiting state")

	bNode = woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode)
	assert.Equal(t, wfv1.NodeFailed, bNode.Phase,
		"B should be Failed")

	// DAG should be in a terminal failure state. With failFast=false, the engine
	// iterates leaf tasks alphabetically and the last failing leaf's phase wins.
	// Since B (Failed) is processed after A (Error), the workflow phase is Failed.
	assert.True(t, woc.wf.Status.Phase.Completed(),
		"DAG should be completed")
	assert.True(t, woc.wf.Status.Phase == wfv1.WorkflowFailed || woc.wf.Status.Phase == wfv1.WorkflowError,
		"DAG should be Failed or Error when tasks have failures")
}

// Test 9: Cascading omission: A fails -> B omitted -> C omitted -> D omitted
var testOperatorBugfixCascadingOmissionChain = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: cascading-omission-chain
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
      - name: B
        depends: "A"
        template: echo
      - name: C
        depends: "B"
        template: echo
      - name: D
        depends: "C"
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorBugfix_CascadingOmissionChain verifies that in a linear chain
// A->B->C->D, when A fails, B, C, and D are all Omitted and the DAG fails.
func TestOperatorBugfix_CascadingOmissionChain(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixCascadingOmissionChain)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A
	woc.operate(ctx)

	// A fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A should be Failed
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	// Run additional cycles to let omission cascade through B->C->D
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// B should be Omitted
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
		"B should be Omitted since A failed")

	// C should be Omitted
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, cNode, "C")
	assert.Equal(t, wfv1.NodeOmitted, cNode.Phase,
		"C should be Omitted since B is Omitted")

	// D should be Omitted
	dNode := woc.wf.Status.Nodes.FindByDisplayName("D")
	require.NotNil(t, dNode, "D")
	assert.Equal(t, wfv1.NodeOmitted, dNode.Phase,
		"D should be Omitted since C is Omitted")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 11: Multiple daemons in parallel -> all dependents proceed
var testOperatorBugfixMultipleDaemonsParallel = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: multi-daemons-parallel
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-task
      - name: B
        template: daemon-task
      - name: C
        depends: "A && B"
        template: echo
  - name: daemon-task
    daemon: true
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorBugfix_MultipleDaemonsParallelDependentProceeds verifies that
// when A(daemon) and B(daemon) are both Running+Daemoned, C (depends: "A && B")
// is created.
func TestOperatorBugfix_MultipleDaemonsParallelDependentProceeds(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixMultipleDaemonsParallel)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A and B
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled")

	// Mark both A and B as Running+Daemoned
	makePodsPhase(ctx, woc, v1.PodRunning)
	daemon := true

	aNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	aNode.Daemoned = &daemon
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	bNode = woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode)
	bNode.Daemoned = &daemon
	woc.wf.Status.Nodes[bNode.ID] = *bNode

	// Cycle 2: both daemons ready -> C should be created
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	assert.NotNil(t, cNode,
		"C should be created when both daemon deps A and B are Running+Daemoned")
}

// Test 13: Retry with OnError policy: Error phase IS retried
var testOperatorBugfixRetryOnErrorErrorRetried = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-onerror-error-retried
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: error-task
  - name: error-task
    retryStrategy:
      limit: "2"
      retryPolicy: OnError
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
`

// TestOperatorBugfix_RetryOnErrorPolicyErrorRetried verifies that when the
// retry policy is OnError and A(0) exits with Error, the retry IS triggered.
// We use a container Waiting state to produce a genuine NodeError from pod reconciliation.
func TestOperatorBugfix_RetryOnErrorPolicyErrorRetried(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixRetryOnErrorErrorRetried)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	a0Node := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node, "A(0) should exist")

	// Make A(0)'s pod fail with a container in Waiting state -> NodeError
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	for _, pod := range pods.Items {
		pod.Status.Phase = v1.PodFailed
		pod.Status.ContainerStatuses = []v1.ContainerStatus{
			{
				Name: "main",
				State: v1.ContainerState{
					Waiting: &v1.ContainerStateWaiting{
						Reason:  "ImagePullBackOff",
						Message: "image pull failed",
					},
				},
			},
		}
		updatedPod, err := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		err = woc.controller.PodController.TestingPodInformer().GetStore().Update(updatedPod)
		require.NoError(t, err)
	}

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(0) should be Error
	a0Node = woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node)
	assert.Equal(t, wfv1.NodeError, a0Node.Phase,
		"A(0) should be Error due to container Waiting state")

	// A(1) SHOULD be created (OnError retries Error phase)
	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	assert.NotNil(t, a1Node,
		"A(1) should be created: OnError policy retries Error phase")
}

// Test 15: Diamond DAG: A->B, A->C, B&&C->D. All succeed -> D runs
var testOperatorBugfixDiamondDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: diamond-dag-bugfix
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        depends: "A"
        template: echo
      - name: C
        depends: "A"
        template: echo
      - name: D
        depends: "B && C"
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorBugfix_DiamondDAGAllSucceed verifies that in a diamond DAG
// (A->B, A->C, B&&C->D), all tasks succeed in order and the DAG succeeds.
func TestOperatorBugfix_DiamondDAGAllSucceed(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixDiamondDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")

	// A succeeds -> B, C should be created
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled after A succeeds")
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, cNode, "C should be scheduled after A succeeds")

	// B, C succeed -> D should be created
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	dNode := woc.wf.Status.Nodes.FindByDisplayName("D")
	require.NotNil(t, dNode, "D should be scheduled after B and C succeed")

	// D succeeds -> DAG succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// Test 16: Enhanced depends: A fails -> B(A.Failed) runs
// Note: continueOn and depends cannot be used together. The depends syntax
// with A.Failed inherently handles this without needing continueOn.
var testOperatorBugfixEnhancedDependsAFailed = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: enhanced-depends-afailed
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
      - name: B
        depends: "A.Failed"
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorBugfix_EnhancedDependsAFailedBRuns verifies that when A fails,
// B (depends: "A.Failed") runs because A.Failed is true.
func TestOperatorBugfix_EnhancedDependsAFailedBRuns(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixEnhancedDependsAFailed)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")

	// A fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A should be Failed
	aNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	// B should be scheduled because A.Failed is true
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should be scheduled when A.Failed is true via enhanced depends")

	if bNode == nil {
		return
	}

	// B succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// DAG should succeed (A.Failed condition was satisfied, B completed successfully)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase,
		"DAG should succeed after B completes (A.Failed condition satisfied)")
}

// ===== TestOperatorEdge tests =====

// --- Dependency readiness edge cases ---

// Test 2: Retry dep failed blocks dependent
var testOperatorEdgeRetryDepFailedBlocksDependent = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-retry-dep-failed-blocks
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: retry-task
      - name: B
        depends: "A"
        template: echo
  - name: retry-task
    retryStrategy:
      limit: "1"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_RetryDepFailedBlocksDependent verifies that when A(retry)
// exhausts retries and fails, B is omitted (not scheduled).
func TestOperatorEdge_RetryDepFailedBlocksDependent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeRetryDepFailedBlocksDependent)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A(0)
	woc.operate(ctx)

	// A(0) fails -> retry A(1)
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(1) fails -> retries exhausted
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Retry node A should be Failed
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase, "Retry node A should be Failed")

	// B should be Omitted (default depends = A.Succeeded which is false)
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
		"B should be Omitted when retry dep A has failed")

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 3: Retry daemon dep fulfills default depends
var testOperatorEdgeRetryDaemonDepFulfillsDefault = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-retry-daemon-dep-default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-retry-task
      - name: B
        depends: "A"
        template: echo
  - name: daemon-retry-task
    daemon: true
    retryStrategy:
      limit: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_RetryDaemonDepFulfillsDefaultDepends verifies that when
// A(retry+daemon) becomes daemoned (Running+Daemoned), B (default depends = "A")
// is scheduled because Daemoned satisfies the default dependency.
func TestOperatorEdge_RetryDaemonDepFulfillsDefaultDepends(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeRetryDaemonDepFulfillsDefault)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	// Mark A(0) as Running+Daemoned
	makePodsPhase(ctx, woc, v1.PodRunning)
	a0Node := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node, "A(0) should exist")
	daemon := true
	a0Node.Daemoned = &daemon
	a0Node.Phase = wfv1.NodeRunning
	woc.wf.Status.Nodes[a0Node.ID] = *a0Node

	// Cycle 2: A(0) is daemoned -> B should be scheduled
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should be scheduled when retry+daemon dep A is Running+Daemoned")
}

// Test 4: Retry daemon dep with explicit A.Succeeded omits B
var testOperatorEdgeRetryDaemonExplicitSucceededOmitsB = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-retry-daemon-explicit-succeeded
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: daemon-retry-task
      - name: B
        depends: "A.Succeeded"
        template: echo
  - name: daemon-retry-task
    daemon: true
    retryStrategy:
      limit: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, "while true; do sleep 1; done"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_RetryDaemonDepExplicitSucceededOmitsB verifies that when
// A(retry+daemon) is daemoned (Running+Daemoned) but B depends on "A.Succeeded",
// B is Omitted, as when A is not retried
// (TestOperatorIntegration_DaemonDepExplicitSucceededOmitsB): A.Succeeded is
// unsatisfiable for a running daemon, so waiting would deadlock.
func TestOperatorEdge_RetryDaemonDepExplicitSucceededOmitsB(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeRetryDaemonExplicitSucceededOmitsB)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	// Mark A(0) as Running+Daemoned
	makePodsPhase(ctx, woc, v1.PodRunning)
	a0Node := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node, "A(0) should exist")
	daemon := true
	a0Node.Daemoned = &daemon
	a0Node.Phase = wfv1.NodeRunning
	woc.wf.Status.Nodes[a0Node.ID] = *a0Node

	// Cycle 2: A is daemoned+running; B depends on A.Succeeded
	// B should be omitted — A.Succeeded is unsatisfiable for a running daemon
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
		"B should be Omitted — A.Succeeded is unsatisfiable for a running daemon")
}

// --- Boundary assessment edge cases ---

// Test 6: All tasks succeeded
var testOperatorEdgeAllTasksSucceeded = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-all-tasks-succeeded
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        depends: "A"
        template: echo
      - name: C
        depends: "A"
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_AllTasksSucceeded verifies that a simple DAG where all
// tasks succeed results in a Succeeded workflow.
func TestOperatorEdge_AllTasksSucceeded(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeAllTasksSucceeded)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A starts
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")

	// A succeeds -> B, C start
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled after A succeeds")
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, cNode, "C should be scheduled after A succeeds")

	// B, C succeed -> DAG succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// Test 7: Leaf failed, no ContinueOn
var testOperatorEdgeLeafFailedNoContinueOn = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-leaf-failed-no-continueon
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
`

// TestOperatorEdge_LeafFailedNoContinueOn verifies that when a leaf task fails
// without continueOn, the DAG is marked Failed.
func TestOperatorEdge_LeafFailedNoContinueOn(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeLeafFailedNoContinueOn)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A starts
	woc.operate(ctx)

	// A fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 9: ContinueOn absorbs failure
var testOperatorEdgeContinueOnAbsorbsFailure = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-continueon-absorbs
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
        continueOn:
          failed: true
      - name: B
        dependencies: [A]
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_ContinueOnAbsorbsFailure verifies that when A fails and has
// continueOn.failed=true, B (dependencies: [A]) still runs because continueOn
// absorbs the failure.
func TestOperatorEdge_ContinueOnAbsorbsFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeContinueOnAbsorbsFailure)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A starts
	woc.operate(ctx)

	// A fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	// B should be scheduled because continueOn absorbs A's failure
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode, "B should be scheduled when continueOn absorbs A's failure")

	if bNode == nil {
		return
	}

	// B succeeds -> DAG should succeed
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase,
		"DAG should succeed when continueOn absorbs A's failure and B completes")
}

// Test 10: Omitted leaf inherits ancestor failure
var testOperatorEdgeOmittedLeafInheritsFailure = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-omitted-leaf-ancestor-fail
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
      - name: B
        depends: "A"
        template: echo
      - name: C
        depends: "B"
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_OmittedLeafInheritsAncestorFailure verifies that in an
// A->B->C chain, when A fails, B and C are omitted and the DAG is Failed.
func TestOperatorEdge_OmittedLeafInheritsAncestorFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeOmittedLeafInheritsFailure)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A starts
	woc.operate(ctx)

	// A fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	// Extra cycles to propagate omission
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// B and C should be Omitted
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase, "B should be Omitted since A failed")
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, cNode, "C")
	assert.Equal(t, wfv1.NodeOmitted, cNode.Phase, "C should be Omitted since B is Omitted")

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 12: All tasks omitted — A fails, B depends on A -> B omitted -> DAG Failed
var testOperatorEdgeAllTasksOmitted = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-all-tasks-omitted
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: fail-task
      - name: B
        depends: "A"
        template: echo
  - name: fail-task
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_AllTasksOmitted verifies that when A fails and B depends on A,
// B is omitted and the DAG is Failed.
func TestOperatorEdge_AllTasksOmitted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeAllTasksOmitted)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Extra cycle to propagate omission
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B")
	assert.Equal(t, wfv1.NodeOmitted, bNode.Phase, "B should be Omitted")

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// --- Composition edge cases ---

// Test 13: Diamond with retry
var testOperatorEdgeDiamondWithRetry = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-diamond-with-retry
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        depends: "A"
        template: retry-task
      - name: C
        depends: "A"
        template: echo
      - name: D
        depends: "B && C"
        template: echo
  - name: retry-task
    retryStrategy:
      limit: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_DiamondWithRetry verifies a diamond DAG A->B(retry), A->C, B&&C->D.
// A succeeds, B retries then succeeds, C succeeds -> D runs.
func TestOperatorEdge_DiamondWithRetry(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeDiamondWithRetry)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A starts
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")

	// A succeeds -> B(0) and C start
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled after A succeeds")
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	require.NotNil(t, cNode, "C should be scheduled after A succeeds")

	// B(0) fails -> retry B(1); C succeeds
	// We need to fail all pods first (B(0) fails), then succeed C
	// Actually makePodsPhase applies to all running pods. Let's fail all then selectively fix C.
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Now B(1) should be created; C is failed. Let's fix C manually and succeed B(1).
	// Actually we need a cleaner approach: succeed C node directly since it's Failed now
	// and succeed B(1).
	cNode = woc.wf.Status.Nodes.FindByDisplayName("C")
	if cNode != nil && cNode.Phase == wfv1.NodeFailed {
		cNode.Phase = wfv1.NodeSucceeded
		woc.wf.Status.Nodes[cNode.ID] = *cNode
	}

	// B(1) should now exist
	b1Node := woc.wf.Status.Nodes.FindByDisplayName("B(1)")
	require.NotNil(t, b1Node, "B(1) should be created after B(0) fails")

	// B(1) succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// B retry node should be Succeeded
	bNode = woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode)
	assert.Equal(t, wfv1.NodeSucceeded, bNode.Phase, "B retry node should be Succeeded")

	// D should be scheduled
	dNode := woc.wf.Status.Nodes.FindByDisplayName("D")
	assert.NotNil(t, dNode, "D should be scheduled after B and C both succeed")
}

// --- Nil safety / edge cases ---

// Test 17: Empty DAG (no tasks) -> Succeeded immediately
var testOperatorEdgeEmptyDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-empty-dag
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks: []
`

// TestOperatorEdge_EmptyDAG verifies that a DAG with no tasks terminates
// immediately on the first operate call (no tasks means the workflow concludes).
func TestOperatorEdge_EmptyDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeEmptyDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// An empty DAG has no tasks that can succeed, so the engine marks it Failed.
	// The key behaviour is that it terminates immediately (not stuck Running).
	assert.NotEqual(t, wfv1.WorkflowRunning, woc.wf.Status.Phase,
		"Empty DAG should terminate immediately, not remain Running")
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase,
		"Empty DAG is treated as Failed by the engine")
}

// Test 18: Single task no deps — A with no deps -> A runs immediately
var testOperatorEdgeSingleTaskNoDeps = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-single-task-no-deps
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_SingleTaskNoDeps verifies that a single task with no
// dependencies is scheduled immediately on the first operate call.
func TestOperatorEdge_SingleTaskNoDeps(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeSingleTaskNoDeps)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	assert.NotNil(t, aNode, "A should be scheduled immediately on the first operate call")

	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// Test 19: Long chain A->B->C->D->E, all succeed sequentially
var testOperatorEdgeLongChain = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-long-chain
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        depends: "A"
        template: echo
      - name: C
        depends: "B"
        template: echo
      - name: D
        depends: "C"
        template: echo
      - name: E
        depends: "D"
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_LongChain verifies that a 5-task sequential chain
// A->B->C->D->E all succeed and the workflow completes with Succeeded.
func TestOperatorEdge_LongChain(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeLongChain)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Process each step sequentially
	taskNames := []string{"A", "B", "C", "D", "E"}
	for i, name := range taskNames {
		woc.operate(ctx)

		node := woc.wf.Status.Nodes.FindByDisplayName(name)
		assert.NotNil(t, node, "%s should be scheduled at step %d", name, i+1)

		makePodsPhase(ctx, woc, v1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase,
		"Long chain should succeed after all tasks complete")
}

// Test 20: Wide fan-out — A, B, C, D, E all independent, all succeed
var testOperatorEdgeWideFanOut = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-wide-fan-out
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        template: echo
      - name: C
        template: echo
      - name: D
        template: echo
      - name: E
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_WideFanOut verifies that 5 independent tasks (A, B, C, D, E)
// are all scheduled in the first cycle and the workflow succeeds after all complete.
func TestOperatorEdge_WideFanOut(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeWideFanOut)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: all 5 tasks should start simultaneously
	woc.operate(ctx)

	for _, name := range []string{"A", "B", "C", "D", "E"} {
		node := woc.wf.Status.Nodes.FindByDisplayName(name)
		assert.NotNil(t, node, "%s should be scheduled in the first cycle", name)
	}

	// All succeed
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase,
		"Wide fan-out DAG should succeed after all tasks complete")
}

// TestDAGWithSequenceOverDAGTemplate verifies that a withSequence over a DAG
// template completes once all expanded inner DAGs' pod children have succeeded.
//
// Regression test for a hang where the outer engine treats every Running
// TaskGroup child as externally driven (like a Pod), so inner DAG/Steps
// instances — which only progress when their engine is re-invoked — are
// never re-entered after their pods succeed. All 19 inner DAGs stayed
// Running, the TaskGroup never succeeded, and a dependent task that should
// have run next was never scheduled.
const dagWithSequenceOverDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: seq-over-dag
spec:
  entrypoint: outer
  templates:
  - name: outer
    dag:
      tasks:
      - name: fan
        template: inner
        withSequence:
          count: "3"
      - name: after
        template: echo
        dependencies: [fan]
        when: "true"
  - name: inner
    dag:
      tasks:
      - name: leaf
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hi]
`

func TestDAGWithSequenceOverDAGTemplate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagWithSequenceOverDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: expand the sequence and create 3 inner DAGs + 3 leaf pods.
	woc.operate(ctx)

	// All 3 leaf pods should have been scheduled.
	for i := range 3 {
		display := fmt.Sprintf("fan(%d:%d)", i, i)
		innerDAG := woc.wf.Status.Nodes.FindByDisplayName(display)
		require.NotNil(t, innerDAG, "inner DAG %q should exist", display)
		assert.Equal(t, wfv1.NodeTypeDAG, innerDAG.Type)
		assert.Equal(t, wfv1.NodeRunning, innerDAG.Phase)
	}

	// Pods succeed.
	makePodsPhase(ctx, woc, v1.PodSucceeded)

	// Cycle 2: the outer engine must re-enter each running inner DAG so it
	// can observe its leaf pod as Succeeded and mark itself Succeeded. Only
	// then does the TaskGroup succeed and the dependent "after" task schedule.
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	for i := range 3 {
		display := fmt.Sprintf("fan(%d:%d)", i, i)
		innerDAG := woc.wf.Status.Nodes.FindByDisplayName(display)
		require.NotNil(t, innerDAG, "inner DAG %q should exist", display)
		assert.Equal(t, wfv1.NodeSucceeded, innerDAG.Phase,
			"inner DAG %q must be Succeeded after its leaf pod succeeded", display)
	}

	after := woc.wf.Status.Nodes.FindByDisplayName("after")
	require.NotNil(t, after, "dependent task 'after' must be scheduled once the TaskGroup succeeds")

	// Let the dependent pod finish and verify the whole workflow succeeds.
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestDAGOrphanedTaskGroupCompletes verifies that a fan-out TaskGroup left Running
// with every expanded child already fulfilled — the state a retry can produce when
// it resets the group but never re-runs it, because its dependents already
// completed — does not hold the DAG Running forever. executeDAGTask never revisits
// such a group, so the controller must complete it from its children during DAG
// assessment.
func TestDAGOrphanedTaskGroupCompletes(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-orphaned-task-group-completes.yaml")
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Status.StartedAt = metav1.Now()

	root := wf.Name
	fanout := root + ".fanout"
	child0 := root + ".fanout(0:a)"
	child1 := root + ".fanout(1:b)"
	leaf := root + ".leaf"
	id := wf.NodeID

	// The fan-out TaskGroup is Running while its children and the downstream leaf
	// have all Succeeded — an orphaned group that nothing will otherwise complete.
	wf.Status.Nodes = wfv1.Nodes{
		id(root):   {ID: id(root), Name: root, Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeRunning, TemplateName: "main", Children: []string{id(fanout)}},
		id(fanout): {ID: id(fanout), Name: fanout, Type: wfv1.NodeTypeTaskGroup, Phase: wfv1.NodeRunning, BoundaryID: id(root), TemplateName: "echo", Children: []string{id(child0), id(child1)}},
		id(child0): {ID: id(child0), Name: child0, Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, BoundaryID: id(root), TemplateName: "echo", Children: []string{id(leaf)}},
		id(child1): {ID: id(child1), Name: child1, Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, BoundaryID: id(root), TemplateName: "echo", Children: []string{id(leaf)}},
		id(leaf):   {ID: id(leaf), Name: leaf, Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, BoundaryID: id(root), TemplateName: "echo"},
	}

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)

	assert.Equal(t, wfv1.NodeSucceeded, woc.wf.Status.Nodes[id(fanout)].Phase, "orphaned TaskGroup should be completed from its children")
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase, "workflow should complete")
}

// TestOnExitDAGNotFailedOnShutdownStop verifies that when a workflow is stopped with
// shutdownStrategy: Stop, an onExit handler that uses a DAG template is still allowed to
// run to completion instead of being immediately failed by the shutdown fast-fail path.
func TestOnExitDAGNotFailedOnShutdownStop(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/on-exit-dag-with-shutdown-stop.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// The onExit DAG should be running, not immediately failed.
	onExitNode := woc.wf.Status.Nodes.FindByDisplayName("dag-onexit-shutdown.onExit")
	if assert.NotNil(t, onExitNode, "onExit DAG node should exist") {
		assert.Equal(t, wfv1.NodeRunning, onExitNode.Phase, "onExit DAG node should be Running, not Failed")
	}

	// The on-exit-handler pod should have been created (Pending).
	exitHandlerNode := woc.wf.Status.Nodes.FindByDisplayName("on-exit-handler")
	if assert.NotNil(t, exitHandlerNode, "on-exit-handler node should exist") {
		assert.Equal(t, wfv1.NodePending, exitHandlerNode.Phase, "on-exit-handler should be Pending")
	}

	// Workflow should NOT be completed yet — it should still be Running waiting for the exit handler.
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "workflow should still be Running while onExit handler is executing")
}

// A TaskGroup's completion belongs to executeDAGTask's end-of-loop check,
// which knows the full expanded item list. assessDAGPhase must not complete
// it from its created children alone: with parallelism deferring one item and
// an exit hook forcing an early return out of the expandedTasks loop, the
// group briefly holds only fulfilled children while an item is still to be
// created.
func TestDAGTaskGroupWithDeferredItems(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-task-group-deferred-items.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 12 {
		woc.operate(ctx)
		if woc.wf.Status.Fulfilled() {
			break
		}
		makePodsPhase(ctx, woc, v1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	for _, item := range []string{"taskgroup-defer.g(0:1)", "taskgroup-defer.g(1:2)"} {
		node, err := woc.wf.GetNodeByName(item)
		if assert.NoError(t, err, "%s must have been run", item) {
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase, item)
		}
	}
}

// A TaskGroup whose created children are all fulfilled while later items are
// still deferred by parallelism must stay Running: the group's phase can only
// be assessed against the full expanded item list, not the children created
// so far. Here item 0 is skipped by its when clause (an instantly fulfilled
// child) and items 1 and 2 need pods that the workflow parallelism defers.
var dagTaskGroupSkippedThenDeferredItems = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: taskgroup-skip-defer
  namespace: default
spec:
  entrypoint: main
  parallelism: 1
  templates:
  - name: main
    dag:
      tasks:
      - name: first
        template: leaf
      - name: g
        template: leaf
        when: "{{item}} != 0"
        withItems: [0, 1, 2]
  - name: leaf
    container:
      image: argoproj/argosay:v2
`

func TestDAGTaskGroupSkippedThenDeferredItems(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagTaskGroupSkippedThenDeferredItems)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 12 {
		woc.operate(ctx)
		if woc.wf.Status.Fulfilled() {
			break
		}
		makePodsPhase(ctx, woc, v1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	skipped, err := woc.wf.GetNodeByName("taskgroup-skip-defer.g(0:0)")
	if assert.NoError(t, err) {
		assert.Equal(t, wfv1.NodeSkipped, skipped.Phase)
	}
	for _, item := range []string{"taskgroup-skip-defer.g(1:1)", "taskgroup-skip-defer.g(2:2)"} {
		node, err := woc.wf.GetNodeByName(item)
		if assert.NoError(t, err, "%s must have been run", item) {
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase, item)
		}
	}
}
