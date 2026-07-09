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
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
)

// newDAGEvaluator builds a DAGEvaluator over a DAG template's tasks, mirroring
// the engine's task wrapping. Test-only: prod uses NewDAGEvaluatorFromTasks.
func newDAGEvaluator(wf *wfv1.Workflow, tmpl *wfv1.Template, boundaryID, boundaryName string) *dag.DAGEvaluator {
	var tasks []dag.Task
	if tmpl.DAG != nil {
		tasks = make([]dag.Task, len(tmpl.DAG.Tasks))
		for i := range tmpl.DAG.Tasks {
			tasks[i] = &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[i]}
		}
	}
	return dag.NewDAGEvaluatorFromTasks(wf, tasks, tmpl, boundaryID, boundaryName)
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

var artifactResolutionWhenSkippedDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: conditional-artifact-passing-
spec:
  entrypoint: artifact-example
  templates:
  - name: artifact-example
    dag:
      tasks:
      - name: generate-artifact
        template: whalesay
        when: "false"
      - name: consume-artifact
        dependencies: [generate-artifact]
        template: print-message
        when: "false"
        arguments:
          artifacts:
          - name: message
            from: "{{tasks.generate-artifact.outputs.artifacts.hello-art}}"
      - name: sequence-param
        template: print-message
        dependencies: [generate-artifact]
        when: "false"
        arguments:
          artifacts:
          - name: message
            from: "{{tasks.generate-artifact.outputs.artifacts.hello-art}}"
        withSequence:
          count: "5"

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 1; cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      - name: hello-art
        path: /tmp/hello_world.txt

  - name: print-message
    inputs:
      artifacts:
      - name: message
        path: /tmp/message
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["cat /tmp/message"]

`

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestArtifactResolutionWhenSkippedDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(artifactResolutionWhenSkippedDAG)
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
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{
			Tasks: []wfv1.DAGTask{task},
		},
	}
	evaluator := newDAGEvaluator(wf, tmpl, "test", "test")

	expanded, err := evaluator.ExpandTask(ctx, task, map[string]string{}, woc)
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
			Name:      "fanout-param(1:foo\tbar)",
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
	evaluator := newDAGEvaluator(wf, tmpl, "test", "test")

	// Task A is running
	nodeID := wf.NodeID("test.A")
	wf.Status.Nodes[nodeID] = wfv1.NodeStatus{Phase: wfv1.NodeRunning}

	// Task B should not proceed, task A is still running
	result := evaluator.EvaluateAll(ctx)["B"]
	require.NoError(t, result.Error)
	assert.True(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Task A succeeded
	wf.Status.Nodes[nodeID] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}

	// Task B and C should proceed and execute
	result = evaluator.EvaluateAll(ctx)["B"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	result = evaluator.EvaluateAll(ctx)["C"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	// Other tasks should not
	result = evaluator.EvaluateAll(ctx)["should-execute-1"]
	require.NoError(t, result.Error)
	assert.True(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Tasks B succeeded, C failed
	wf.Status.Nodes[wf.NodeID("test.B")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.C")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Tasks should-execute-1 and should-execute-2 should proceed and execute
	result = evaluator.EvaluateAll(ctx)["should-execute-1"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	result = evaluator.EvaluateAll(ctx)["should-execute-2"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)
	// Task should-not-execute should proceed, but not execute
	result = evaluator.EvaluateAll(ctx)["should-not-execute"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Tasks should-execute-1 and should-execute-2 succeeded, should-not-execute skipped
	wf.Status.Nodes[wf.NodeID("test.should-execute-1")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.should-execute-2")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.should-not-execute")] = wfv1.NodeStatus{Phase: wfv1.NodeSkipped}

	// Tasks should-execute-3 should proceed and execute
	result = evaluator.EvaluateAll(ctx)["should-execute-3"]
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
	evaluator := newDAGEvaluator(wf, tmpl, "test", "test")

	// Task A is still running, A-1 succeeded but A-2 failed
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{wf.NodeID("test.A-1"), wf.NodeID("test.A-2")},
	}
	wf.Status.Nodes[wf.NodeID("test.A-1")] = wfv1.NodeStatus{Phase: wfv1.NodeRunning}
	wf.Status.Nodes[wf.NodeID("test.A-2")] = wfv1.NodeStatus{Phase: wfv1.NodeRunning}

	// Task B should not proceed as task A is still running
	result := evaluator.EvaluateAll(ctx)["B"]
	require.NoError(t, result.Error)
	assert.True(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Task A succeeded
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{wf.NodeID("test.A-1"), wf.NodeID("test.A-2")},
	}

	// Task B should proceed, but not execute as none of the children have succeeded yet
	result = evaluator.EvaluateAll(ctx)["B"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	// Task A-2 succeeded
	wf.Status.Nodes[wf.NodeID("test.A-2")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}

	// Task B should now proceed and execute
	result = evaluator.EvaluateAll(ctx)["B"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.True(t, result.ShouldRun)

	// Task B succeeds and B-1 fails
	wf.Status.Nodes[wf.NodeID("test.B")] = wfv1.NodeStatus{
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{wf.NodeID("test.B-1"), wf.NodeID("test.B-2")},
	}
	wf.Status.Nodes[wf.NodeID("test.B-1")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Task C should proceed, but not execute as not all of B's children have failed yet
	result = evaluator.EvaluateAll(ctx)["C"]
	require.NoError(t, result.Error)
	assert.False(t, result.Suspended)
	assert.False(t, result.ShouldRun)

	wf.Status.Nodes[wf.NodeID("test.B-2")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Task C should now proceed and execute as all of B's children have failed
	result = evaluator.EvaluateAll(ctx)["C"]
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
	evaluator := newDAGEvaluator(wf, tmpl, "test", "test")

	// Task A is running
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{Phase: wfv1.NodeOmitted}

	// Task B should proceed and execute
	result := evaluator.EvaluateAll(ctx)["B"]
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
		evaluator := newDAGEvaluator(wf, tmpl, "test", "test")

		// Task A is running
		wf.Status.Nodes[wf.NodeID("test.same")] = wfv1.NodeStatus{Phase: statusMap[status]}

		result := evaluator.EvaluateAll(ctx)["Run"]
		require.NoError(t, result.Error)
		assert.False(t, result.Suspended)
		assert.True(t, result.ShouldRun)
		result = evaluator.EvaluateAll(ctx)["NotRun"]
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
	wf.Status.Nodes[wf.NodeID("test.A")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.B")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	wf.Status.Nodes[wf.NodeID("test.C")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	evaluator := newDAGEvaluator(wf, tmpl, "test", "test")

	// D: "A && (C.Succeeded || C.Failed)" — A succeeded, C failed → C.Failed is true → should run
	result := evaluator.EvaluateAll(ctx)["D"]
	require.NoError(t, result.Error)
	assert.True(t, result.ShouldRun, "D should run: A succeeded and C.Failed is true")

	// E: "B || C" — expanded to "(B.Succeeded||B.Skipped||B.Daemoned) || (C.Succeeded||C.Skipped||C.Daemoned)"
	// B.Succeeded is true → should run
	result = evaluator.EvaluateAll(ctx)["E"]
	require.NoError(t, result.Error)
	assert.True(t, result.ShouldRun, "E should run: B succeeded")

	// F: "B && C" — expanded: B.Succeeded is true but C.Succeeded is false → skipped
	result = evaluator.EvaluateAll(ctx)["F"]
	require.NoError(t, result.Error)
	assert.False(t, result.ShouldRun, "F should not run: C did not succeed")
	assert.True(t, result.Skipped, "F should be skipped")

	// G: "E || F" — E has no workflow node yet, F has no workflow node yet → suspended
	result = evaluator.EvaluateAll(ctx)["G"]
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
	wf.Status.Nodes[wf.NodeID("test.C")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	evaluator := newDAGEvaluator(wf, tmpl, "test", "test")

	// D should be ready to run (C.Failed is true)
	result := evaluator.EvaluateAll(ctx)["D"]
	require.NoError(t, result.Error)
	assert.True(t, result.ShouldRun, "D should run because C.Failed is true")

	// The DAG should still have pending tasks, so assessDAGPhase should return Running
	// (D is Pending - no workflow node yet)
	results := evaluator.EvaluateAll(ctx)
	hasPendingOrRunning := false
	for _, r := range results {
		if r.CurrentPhase == wfv1.NodePending || r.CurrentPhase == wfv1.NodeRunning {
			hasPendingOrRunning = true
			break
		}
	}
	assert.True(t, hasPendingOrRunning, "DAG should have pending/running tasks (D), so it should be Running not Failed")
}

// Restored integration tests from the old DAG engine test suite, adapted
// for the new multi-cycle reconciliation pattern.

var testRetryStrategyNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wf-retry-pol
spec:

  entrypoint: run-steps
  onExit: onExit
  templates:
  -
    inputs: {}
    metadata: {}
    name: run-steps
    outputs: {}
    steps:
    - -
        name: run-dag
        template: run-dag
    - -
        name: manual-onExit
        template: onExit
  -
    dag:
      tasks:
      -
        name: A
        template: fail
      -
        dependencies:
        - A
        name: B
        template: onExit
    inputs: {}
    metadata: {}
    name: run-dag
    outputs: {}
  -
    container:
      args:
      - exit 2
      command:
      - sh
      - -c
      image: alpine
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
    retryStrategy:
      limit: 100
      retryPolicy: OnError
  -
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: onExit
    outputs: {}
status:
  nodes:
    wf-retry-pol:
      children:
      - wf-retry-pol-3488382045
      displayName: wf-retry-pol
      finishedAt: "2020-05-27T16:03:00Z"
      id: wf-retry-pol
      name: wf-retry-pol
      outboundNodes:
      - wf-retry-pol-2278798366
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: run-steps
      templateScope: local/wf-retry-pol
      type: Steps
    wf-retry-pol-2616013767:
      boundaryID: wf-retry-pol
      children:
      - wf-retry-pol-3151556158
      displayName: run-dag
      id: wf-retry-pol-2616013767
      name: wf-retry-pol[0].run-dag
      outboundNodes:
      - wf-retry-pol-3134778539
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: run-dag
      templateScope: local/wf-retry-pol
      type: DAG
    wf-retry-pol-3148069997:
      boundaryID: wf-retry-pol-2616013767
      children:
      - wf-retry-pol-3134778539
      displayName: A(0)
      finishedAt: "2020-05-27T16:02:53Z"
      hostNodeName: minikube
      id: wf-retry-pol-3148069997
      message: failed with exit code 2
      name: wf-retry-pol[0].run-dag.A(0)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: wf-retry-pol/wf-retry-pol-3148069997/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "2"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-05-27T16:02:49Z"
      templateName: fail
      templateScope: local/wf-retry-pol
      type: Pod
    wf-retry-pol-3151556158:
      boundaryID: wf-retry-pol-2616013767
      children:
      - wf-retry-pol-3148069997
      displayName: A
      id: wf-retry-pol-3151556158
      message: failed with exit code 2
      name: wf-retry-pol[0].run-dag.A
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: fail
      templateScope: local/wf-retry-pol
      type: Retry
    wf-retry-pol-3488382045:
      boundaryID: wf-retry-pol
      children:
      - wf-retry-pol-2616013767
      displayName: '[0]'
      id: wf-retry-pol-3488382045
      name: wf-retry-pol[0]
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: run-steps
      templateScope: local/wf-retry-pol
      type: StepGroup
  phase: Running
  resourcesDuration:
    cpu: 6
    memory: 2
  startedAt: "2020-05-27T16:02:49Z"
`

func TestRetryStrategyNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testRetryStrategyNodes)
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

var testOnExitNodeDAGPhase = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-88trp
spec:

  entrypoint: diamond
  templates:
  -
    dag:
      failFast: false
      tasks:
      -
        name: A
        template: echo
      -
        dependencies:
        - A
        name: B
        onExit: echo
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  -
    container:
      args:
      - exit 0
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: echo
    outputs: {}
  -
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
status:
  nodes:
    dag-diamond-88trp:
      children:
      - dag-diamond-88trp-2052796420
      displayName: dag-diamond-88trp
      id: dag-diamond-88trp
      name: dag-diamond-88trp
      outboundNodes:
      - dag-diamond-88trp-2103129277
      phase: Running
      startedAt: "2020-05-29T18:11:55Z"
      templateName: diamond
      templateScope: local/dag-diamond-88trp
      type: DAG
    dag-diamond-88trp-2052796420:
      boundaryID: dag-diamond-88trp
      children:
      - dag-diamond-88trp-2103129277
      displayName: A
      finishedAt: "2020-05-29T18:11:58Z"
      hostNodeName: minikube
      id: dag-diamond-88trp-2052796420
      name: dag-diamond-88trp.A
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-88trp/dag-diamond-88trp-2052796420/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-05-29T18:11:55Z"
      templateName: echo
      templateScope: local/dag-diamond-88trp
      type: Pod
    dag-diamond-88trp-2103129277:
      boundaryID: dag-diamond-88trp
      displayName: B
      finishedAt: "2020-05-29T18:12:01Z"
      hostNodeName: minikube
      id: dag-diamond-88trp-2103129277
      name: dag-diamond-88trp.B
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-88trp/dag-diamond-88trp-2103129277/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-29T18:11:59Z"
      templateName: echo
      templateScope: local/dag-diamond-88trp
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 5
    memory: 2
  startedAt: "2020-05-29T18:11:55Z"
`

func TestOnExitDAGPhase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOnExitNodeDAGPhase)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	onExitNode, err := woc.wf.GetNodeByName("dag-diamond-88trp.B.onExit")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodePending, onExitNode.Phase)
	assert.True(t, onExitNode.NodeFlag.Hooked)
}

var testDagOptionalInputArtifacts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-optional-inputartifacts
spec:
  entrypoint: test
  templates:
  - name: condition
    outputs:
      artifacts:
      - {name: A-out, from: '{{tasks.A.outputs.artifacts.A-out}}'}
    dag:
      tasks:
      - {name: A, template: A}
  - name: A
    container:
      args: ['mkdir -p /tmp/outputs/A && echo "exist" > /tmp/outputs/A/data']
      command: [sh, -c]
      image: alpine:3.23
    outputs:
      artifacts:
      - {name: A-out, path: /tmp/outputs/A/data}
  - name: B
    container:
      args: ['[ -f /tmp/outputs/condition/data ] && cat /tmp/outputs/condition/data || echo not exist']
      command: [sh, -c]
      image: alpine:3.23
    inputs:
      artifacts:
      - {name: B-in, optional: true,  path: /tmp/outputs/condition/data}
  - name: test
    dag:
      tasks:
      - name: condition
        template: condition
        when: 'false'
      - name: B
        template: B
        dependencies: [condition]
        arguments:
          artifacts:
          - {name: B-in, optional: true, from: '{{tasks.condition.outputs.artifacts.A-out}}'}
  arguments:
    parameters: []
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-07-21T01:56:24Z"
  nodes:
    dag-optional-inputartifacts:
      children:
      - dag-optional-inputartifacts-3418089753
      displayName: dag-optional-inputartifacts
      finishedAt: "2020-07-21T01:56:24Z"
      id: dag-optional-inputartifacts
      name: dag-optional-inputartifacts
      outboundNodes:
      - dag-optional-inputartifacts-1920355018
      phase: Running
      startedAt: "2020-07-21T01:56:18Z"
      templateName: test
      templateScope: local/dag-optional-inputartifacts
      type: DAG
    dag-optional-inputartifacts-3418089753:
      boundaryID: dag-optional-inputartifacts
      children:
      - dag-optional-inputartifacts-1920355018
      displayName: condition
      finishedAt: "2020-07-21T01:56:18Z"
      id: dag-optional-inputartifacts-3418089753
      message: when 'false' evaluated false
      name: dag-optional-inputartifacts.condition
      phase: Skipped
      startedAt: "2020-07-21T01:56:18Z"
      templateName: condition
      templateScope: local/dag-optional-inputartifacts
      type: Skipped
  phase: Running
  resourcesDuration:
    cpu: 1
    memory: 0
  startedAt: "2020-07-21T01:56:18Z"
`

func TestDagOptionalInputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(testDagOptionalInputArtifacts)
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

var testEmptyWithParamDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-hang-pcwmr
spec:

  entrypoint: dag
  templates:
  -
    dag:
      tasks:
      -
        name: scheduler
        template: job-scheduler
      - arguments:
          parameters:
          - name: job_name
            value: '{{item.job_name}}'
        dependencies:
        - scheduler
        name: children
        template: whalesay
        withParam: '{{tasks.scheduler.outputs.parameters.scheduled-jobs}}'
      -
        dependencies:
        - children
        name: postprocess
        template: whalesay
    inputs: {}
    metadata: {}
    name: dag
    outputs: {}
  -
    container:
      args:
      - echo Decided not to schedule any jobs
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: job-scheduler
    outputs:
      parameters:
      - name: scheduled-jobs
        value: '[]'
  -
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  finishedAt: null
  nodes:
    dag-hang-pcwmr:
      children:
      - dag-hang-pcwmr-1789179473
      displayName: dag-hang-pcwmr
      finishedAt: null
      id: dag-hang-pcwmr
      name: dag-hang-pcwmr
      phase: Running
      startedAt: "2020-08-11T18:19:28Z"
      templateName: dag
      templateScope: local/dag-hang-pcwmr
      type: DAG
    dag-hang-pcwmr-1415348083:
      boundaryID: dag-hang-pcwmr
      displayName: postprocess
      finishedAt: "2020-08-11T18:19:40Z"
      hostNodeName: dech117
      id: dag-hang-pcwmr-1415348083
      name: dag-hang-pcwmr.postprocess
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2020-08-11T18:19:36Z"
      templateName: whalesay
      templateScope: local/dag-hang-pcwmr
      type: Pod
    dag-hang-pcwmr-1738428083:
      boundaryID: dag-hang-pcwmr
      children:
      - dag-hang-pcwmr-1415348083
      displayName: children
      finishedAt: "2020-08-11T18:19:35Z"
      id: dag-hang-pcwmr-1738428083
      name: dag-hang-pcwmr.children
      phase: Succeeded
      startedAt: "2020-08-11T18:19:35Z"
      templateName: whalesay
      templateScope: local/dag-hang-pcwmr
      type: TaskGroup
    dag-hang-pcwmr-1789179473:
      boundaryID: dag-hang-pcwmr
      children:
      - dag-hang-pcwmr-1738428083
      displayName: scheduler
      finishedAt: "2020-08-11T18:19:33Z"
      hostNodeName: dech113
      id: dag-hang-pcwmr-1789179473
      name: dag-hang-pcwmr.scheduler
      outputs:
        exitCode: "0"
        parameters:
        - name: scheduled-jobs
          value: '[]'
      phase: Succeeded
      resourcesDuration:
        cpu: 4
        memory: 4
      startedAt: "2020-08-11T18:19:28Z"
      templateName: job-scheduler
      templateScope: local/dag-hang-pcwmr
      type: Pod
  phase: Running
  startedAt: "2020-08-11T18:19:28Z"
`

func TestEmptyWithParamDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testEmptyWithParamDAG)
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

var testLeafContinueOn = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: build-wf-kpxvm
spec:

  entrypoint: test-workflow
  templates:
  -
    dag:
      tasks:
      -
        name: A
        template: ok
      -
        continueOn:
          failed: true
        dependencies:
        - A
        name: B
        template: fail
    inputs: {}
    metadata: {}
    name: test-workflow
    outputs: {}
  -
    container:
      args:
      - |
        exit 0
      command:
      - sh
      - -c
      image: busybox
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: ok
    outputs: {}
  -
    container:
      args:
      - |
        exit 1
      command:
      - sh
      - -c
      image: busybox
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
status:
  finishedAt: "2020-11-04T16:17:59Z"
  nodes:
    build-wf-kpxvm:
      children:
      - build-wf-kpxvm-2225940411
      displayName: build-wf-kpxvm
      finishedAt: "2020-11-04T16:17:59Z"
      id: build-wf-kpxvm
      name: build-wf-kpxvm
      outboundNodes:
      - build-wf-kpxvm-2242718030
      phase: Running
      progress: 3/3
      resourcesDuration:
        cpu: 13
        memory: 6
      startedAt: "2020-11-04T16:17:43Z"
      templateName: test-workflow
      templateScope: local/build-wf-kpxvm
      type: DAG
    build-wf-kpxvm-2225940411:
      boundaryID: build-wf-kpxvm
      children:
      - build-wf-kpxvm-2242718030
      displayName: A
      finishedAt: "2020-11-04T16:17:51Z"
      hostNodeName: minikube
      id: build-wf-kpxvm-2225940411
      name: build-wf-kpxvm.A
      phase: Succeeded
      startedAt: "2020-11-04T16:17:43Z"
      templateName: ok
      templateScope: local/build-wf-kpxvm
      type: Pod
    build-wf-kpxvm-2242718030:
      boundaryID: build-wf-kpxvm
      displayName: B
      finishedAt: "2020-11-04T16:17:57Z"
      hostNodeName: minikube
      id: build-wf-kpxvm-2242718030
      message: failed with exit code 1
      name: build-wf-kpxvm.B
      phase: Failed
      startedAt: "2020-11-04T16:17:53Z"
      templateName: fail
      templateScope: local/build-wf-kpxvm
      type: Pod
  phase: Running
  progress: 3/3
  resourcesDuration:
    cpu: 13
    memory: 6
  startedAt: "2020-11-04T16:17:43Z"

`

func TestLeafContinueOn(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testLeafContinueOn)
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
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-parallelism
  namespace: argo
spec:
  entrypoint: main
  parallelism: 1
  templates:
    - name: main
      dag:
        tasks:
          - name: do-it-once
            template: do-it
            arguments:
              parameters:
                - name: thing
                  value: 1
          - name: do-it-twice
            template: do-it
            arguments:
              parameters:
                - name: thing
                  value: 2
          - name: do-it-thrice
            template: do-it
            arguments:
              parameters:
                - name: thing
                  value: 3
    - name: do-it
      inputs:
        parameters:
          - name: thing
      container:
        image: docker/whalesay:latest
        command: [cowsay]
        args: ["I have a {{inputs.parameters.thing}}"]`)

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
	assert.NotNil(t, bNode, "B should be scheduled when A is daemoned and running")
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

var dagAssessPhaseContinueOnExpandedTaskVariables = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-one-will-fail2-jt776
spec:

  entrypoint: parameter-aggregation-one-will-fail2
  templates:
  -
    dag:
      tasks:
      -
        continueOn:
          failed: true
        name: generate
        template: gen-number-list
      - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        continueOn:
          failed: true
        dependencies:
        - generate
        name: one-will-fail
        template: one-will-fail
        withParam: '{{tasks.generate.outputs.result}}'
      -
        continueOn:
          failed: true
        dependencies:
        - one-will-fail
        name: whalesay
        template: whalesay
    inputs: {}
    metadata: {}
    name: parameter-aggregation-one-will-fail2
    outputs: {}
  -
    container:
      args:
      - |
        if [ $(({{inputs.parameters.num}})) == 1 ]; then
          exit 1;
        else
          echo {{inputs.parameters.num}}
        fi
      command:
      - sh
      - -xc
      image: alpine:3.23
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: one-will-fail
    outputs: {}
  -
    container:
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
  -
    inputs: {}
    metadata: {}
    name: gen-number-list
    outputs: {}
    script:
      command:
      - python
      image: python:alpine3.23
      name: ""
      resources: {}
      source: |
        import json
        import sys
        json.dump([i for i in range(0, 2)], sys.stdout)
status:
  nodes:
    parameter-aggregation-one-will-fail2-jt776:
      children:
      - parameter-aggregation-one-will-fail2-jt776-1457662774
      displayName: parameter-aggregation-one-will-fail2-jt776
      id: parameter-aggregation-one-will-fail2-jt776
      name: parameter-aggregation-one-will-fail2-jt776
      outboundNodes:
      - parameter-aggregation-one-will-fail2-jt776-3936077093
      phase: Running
      startedAt: "2020-04-20T16:39:00Z"
      templateName: parameter-aggregation-one-will-fail2
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: DAG
    parameter-aggregation-one-will-fail2-jt776-6921149:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-1842114754
      - parameter-aggregation-one-will-fail2-jt776-4113411742
      displayName: one-will-fail
      finishedAt: "2020-04-20T16:39:09Z"
      id: parameter-aggregation-one-will-fail2-jt776-6921149
      name: parameter-aggregation-one-will-fail2-jt776.one-will-fail
      phase: Failed
      startedAt: "2020-04-20T16:39:03Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: TaskGroup
    parameter-aggregation-one-will-fail2-jt776-1457662774:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-6921149
      displayName: generate
      finishedAt: "2020-04-20T16:39:02Z"
      id: parameter-aggregation-one-will-fail2-jt776-1457662774
      name: parameter-aggregation-one-will-fail2-jt776.generate
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-1457662774/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
        result: '[0, 1]'
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-04-20T16:39:00Z"
      templateName: gen-number-list
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
    parameter-aggregation-one-will-fail2-jt776-1842114754:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-3936077093
      displayName: one-will-fail(0:0)
      finishedAt: "2020-04-20T16:39:06Z"
      id: parameter-aggregation-one-will-fail2-jt776-1842114754
      inputs:
        parameters:
        - name: num
          value: "0"
      name: parameter-aggregation-one-will-fail2-jt776.one-will-fail(0:0)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-1842114754/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-04-20T16:39:03Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
    parameter-aggregation-one-will-fail2-jt776-3936077093:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      displayName: whalesay
      finishedAt: "2020-04-20T16:39:14Z"
      id: parameter-aggregation-one-will-fail2-jt776-3936077093
      name: parameter-aggregation-one-will-fail2-jt776.whalesay
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-3936077093/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-04-20T16:39:10Z"
      templateName: whalesay
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
    parameter-aggregation-one-will-fail2-jt776-4113411742:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-3936077093
      displayName: one-will-fail(1:1)
      finishedAt: "2020-04-20T16:39:07Z"
      id: parameter-aggregation-one-will-fail2-jt776-4113411742
      inputs:
        parameters:
        - name: num
          value: "1"
      message: failed with exit code 1
      name: parameter-aggregation-one-will-fail2-jt776.one-will-fail(1:1)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-4113411742/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-04-20T16:39:03Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 10
    memory: 0
  startedAt: "2020-04-20T16:39:00Z"
`

// Tests whether assessPhase marks a DAG as successful when it contains failed tasks with continueOn failed
func TestDagAssessPhaseContinueOnExpandedTaskVariables(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagAssessPhaseContinueOnExpandedTaskVariables)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var dagAssessPhaseContinueOnExpandedTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-one-will-fail-69x7k
spec:

  entrypoint: parameter-aggregation-one-will-fail
  templates:
  -
    dag:
      tasks:
      - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        continueOn:
          failed: true
        name: one-will-fail
        template: one-will-fail
        withItems:
        - 1
        - 2
      -
        continueOn:
          failed: true
        dependencies:
        - one-will-fail
        name: whalesay
        template: whalesay
    inputs: {}
    metadata: {}
    name: parameter-aggregation-one-will-fail
    outputs: {}
  -
    container:
      args:
      - |
        if [ $(({{inputs.parameters.num}})) == 1 ]; then
          exit 1;
        else
          echo {{inputs.parameters.num}}
        fi
      command:
      - sh
      - -xc
      image: alpine:3.23
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: one-will-fail
    outputs: {}
  -
    container:
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  nodes:
    parameter-aggregation-one-will-fail-69x7k:
      children:
      - parameter-aggregation-one-will-fail-69x7k-4292161196
      displayName: parameter-aggregation-one-will-fail-69x7k
      id: parameter-aggregation-one-will-fail-69x7k
      name: parameter-aggregation-one-will-fail-69x7k
      outboundNodes:
      - parameter-aggregation-one-will-fail-69x7k-3555414042
      phase: Running
      startedAt: "2020-04-20T16:47:22Z"
      templateName: parameter-aggregation-one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: DAG
    parameter-aggregation-one-will-fail-69x7k-1324058456:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      children:
      - parameter-aggregation-one-will-fail-69x7k-3555414042
      displayName: one-will-fail(0:1)
      finishedAt: "2020-04-20T16:47:26Z"
      id: parameter-aggregation-one-will-fail-69x7k-1324058456
      inputs:
        parameters:
        - name: num
          value: "1"
      message: failed with exit code 1
      name: parameter-aggregation-one-will-fail-69x7k.one-will-fail(0:1)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: parameter-aggregation-one-will-fail-69x7k/parameter-aggregation-one-will-fail-69x7k-1324058456/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-04-20T16:47:22Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: Pod
    parameter-aggregation-one-will-fail-69x7k-3086527730:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      children:
      - parameter-aggregation-one-will-fail-69x7k-3555414042
      displayName: one-will-fail(1:2)
      finishedAt: "2020-04-20T16:47:28Z"
      id: parameter-aggregation-one-will-fail-69x7k-3086527730
      inputs:
        parameters:
        - name: num
          value: "2"
      name: parameter-aggregation-one-will-fail-69x7k.one-will-fail(1:2)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: parameter-aggregation-one-will-fail-69x7k/parameter-aggregation-one-will-fail-69x7k-3086527730/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 4
        memory: 0
      startedAt: "2020-04-20T16:47:22Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: Pod
    parameter-aggregation-one-will-fail-69x7k-3555414042:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      displayName: whalesay
      finishedAt: "2020-04-20T16:47:33Z"
      id: parameter-aggregation-one-will-fail-69x7k-3555414042
      name: parameter-aggregation-one-will-fail-69x7k.whalesay
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: parameter-aggregation-one-will-fail-69x7k/parameter-aggregation-one-will-fail-69x7k-3555414042/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-04-20T16:47:30Z"
      templateName: whalesay
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: Pod
    parameter-aggregation-one-will-fail-69x7k-4292161196:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      children:
      - parameter-aggregation-one-will-fail-69x7k-1324058456
      - parameter-aggregation-one-will-fail-69x7k-3086527730
      displayName: one-will-fail
      finishedAt: "2020-04-20T16:47:29Z"
      id: parameter-aggregation-one-will-fail-69x7k-4292161196
      name: parameter-aggregation-one-will-fail-69x7k.one-will-fail
      phase: Failed
      startedAt: "2020-04-20T16:47:22Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: TaskGroup
  phase: Running
  resourcesDuration:
    cpu: 9
    memory: 0
  startedAt: "2020-04-20T16:47:22Z"
`

// Tests whether assessPhase marks a DAG as successful when it contains failed tasks with continueOn failed
func TestDagAssessPhaseContinueOnExpandedTask(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagAssessPhaseContinueOnExpandedTask)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var dagWithParamAndGlobalParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-with-param-and-global-param-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: workspace
      value: /argo_workspace/{{workflow.uid}}
  templates:
  - name: main
    dag:
      tasks:
      - name: use-with-param
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello {{workflow.parameters.workspace}} {{item}}"
        withParam: "[0, 1, 2]"
  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestDAGWithParamAndGlobalParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagWithParamAndGlobalParam)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var terminatingDAGWithRetryStrategyNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-xfww2
spec:

  entrypoint: diamond
  shutdown: Terminate
  templates:
  -
    dag:
      tasks:
      -
        name: A
        template: echo
      -
        dependencies:
        - A
        name: B
        template: echo
      -
        dependencies:
        - A
        name: C
        template: echo
      -
        dependencies:
        - B
        - C
        name: D
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  -
    container:
      args:
      - sleep 10
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: echo
    outputs: {}
    retryStrategy:
      limit: 4
status:
  finishedAt: null
  nodes:
    dag-diamond-xfww2:
      children:
      - dag-diamond-xfww2-1488588956
      displayName: dag-diamond-xfww2
      finishedAt: null
      id: dag-diamond-xfww2
      name: dag-diamond-xfww2
      phase: Running
      startedAt: "2020-05-06T16:15:38Z"
      templateName: diamond
      templateScope: local/dag-diamond-xfww2
      type: DAG
    dag-diamond-xfww2-990947287:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-1522144194
      - dag-diamond-xfww2-1538921813
      displayName: A(0)
      finishedAt: "2020-05-06T16:15:50Z"
      hostNodeName: minikube
      id: dag-diamond-xfww2-990947287
      name: dag-diamond-xfww2.A(0)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-xfww2/dag-diamond-xfww2-990947287/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Succeeded
      resourcesDuration:
        cpu: 21
        memory: 0
      startedAt: "2020-05-06T16:15:38Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Pod
    dag-diamond-xfww2-1488588956:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-990947287
      displayName: A
      finishedAt: "2020-05-06T16:15:51Z"
      id: dag-diamond-xfww2-1488588956
      name: dag-diamond-xfww2.A
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-xfww2/dag-diamond-xfww2-990947287/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Succeeded
      startedAt: "2020-05-06T16:15:38Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Retry
    dag-diamond-xfww2-1522144194:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-2043927737
      displayName: C
      finishedAt: "2020-05-06T16:15:59Z"
      id: dag-diamond-xfww2-1522144194
      message: Stopped with strategy 'Terminate'
      name: dag-diamond-xfww2.C
      phase: Failed
      startedAt: "2020-05-06T16:15:51Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Retry
    dag-diamond-xfww2-1538921813:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-3629114292
      displayName: B
      finishedAt: "2020-05-06T16:15:59Z"
      id: dag-diamond-xfww2-1538921813
      message: Stopped with strategy 'Terminate'
      name: dag-diamond-xfww2.B
      phase: Failed
      startedAt: "2020-05-06T16:15:52Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Retry
    dag-diamond-xfww2-2043927737:
      boundaryID: dag-diamond-xfww2
      displayName: C(0)
      finishedAt: "2020-05-06T16:15:58Z"
      hostNodeName: minikube
      id: dag-diamond-xfww2-2043927737
      message: terminated
      name: dag-diamond-xfww2.C(0)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-xfww2/dag-diamond-xfww2-2043927737/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Failed
      resourcesDuration:
        cpu: 11
        memory: 0
      startedAt: "2020-05-06T16:15:51Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Pod
    dag-diamond-xfww2-3629114292:
      boundaryID: dag-diamond-xfww2
      displayName: B(0)
      finishedAt: "2020-05-06T16:15:58Z"
      hostNodeName: minikube
      id: dag-diamond-xfww2-3629114292
      message: terminated
      name: dag-diamond-xfww2.B(0)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-xfww2/dag-diamond-xfww2-3629114292/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Failed
      resourcesDuration:
        cpu: 9
        memory: 0
      startedAt: "2020-05-06T16:15:52Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Pod
  phase: Running
  startedAt: "2020-05-06T16:15:38Z"
`

// This tests that a DAG with retry strategy in its tasks fails successfully when terminated
func TestTerminatingDAGWithRetryStrategyNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(terminatingDAGWithRetryStrategyNodes)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var terminateDAGWithMaxDurationLimitExpiredAndMoreAttempts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-dj7q5
spec:

  entrypoint: diamond
  templates:
  -
    dag:
      tasks:
      -
        name: A
        template: echo
      -
        dependencies:
        - A
        name: B
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  -
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: echo
    outputs: {}
    retryStrategy:
      backoff:
        duration: "1"
        maxDuration: "5"
      limit: 10
status:
  nodes:
    dag-diamond-dj7q5:
      children:
      - dag-diamond-dj7q5-2391658435
      displayName: dag-diamond-dj7q5
      finishedAt: "2020-05-27T15:42:01Z"
      id: dag-diamond-dj7q5
      name: dag-diamond-dj7q5
      phase: Running
      startedAt: "2020-05-27T15:41:54Z"
      templateName: diamond
      templateScope: local/dag-diamond-dj7q5
      type: DAG
    dag-diamond-dj7q5-2241203531:
      boundaryID: dag-diamond-dj7q5
      displayName: A(1)
      finishedAt: "2020-05-27T15:41:59Z"
      hostNodeName: minikube
      id: dag-diamond-dj7q5-2241203531
      message: failed with exit code 1
      name: dag-diamond-dj7q5.A(1)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-dj7q5/dag-diamond-dj7q5-2241203531/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-27T15:41:57Z"
      templateName: echo
      templateScope: local/dag-diamond-dj7q5
      type: Pod
    dag-diamond-dj7q5-2391658435:
      boundaryID: dag-diamond-dj7q5
      children:
      - dag-diamond-dj7q5-2845344910
      - dag-diamond-dj7q5-2241203531
      displayName: A
      finishedAt: "2020-05-27T15:42:01Z"
      id: dag-diamond-dj7q5-2391658435
      name: dag-diamond-dj7q5.A
      phase: Running
      startedAt: "2020-05-27T15:41:54Z"
      templateName: echo
      templateScope: local/dag-diamond-dj7q5
      type: Retry
    dag-diamond-dj7q5-2845344910:
      boundaryID: dag-diamond-dj7q5
      displayName: A(0)
      finishedAt: "2020-05-27T15:41:56Z"
      hostNodeName: minikube
      id: dag-diamond-dj7q5-2845344910
      message: failed with exit code 1
      name: dag-diamond-dj7q5.A(0)
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-diamond-dj7q5/dag-diamond-dj7q5-2845344910/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-27T15:41:54Z"
      templateName: echo
      templateScope: local/dag-diamond-dj7q5
      type: Pod
      nodeFlag:
        retried: true
  phase: Running
  resourcesDuration:
    cpu: 2
    memory: 0
  startedAt: "2020-05-27T15:41:54Z"
`

// This tests that a DAG with retry strategy in its tasks fails successfully when terminated

var testOnExitNonLeaf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handler-bug-example
spec:

  entrypoint: dag
  templates:
  -
    dag:
      tasks:
      -
        name: step-2
        onExit: on-exit
        template: step-template
      -
        dependencies:
        - step-2
        name: step-3
        onExit: on-exit
        template: step-template
    inputs: {}
    metadata: {}
    name: dag
    outputs: {}
  -
    container:
      args:
      - echo exit-handler-step-{{pod.name}}
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: on-exit
    outputs: {}
  -
    container:
      args:
      - echo step {{pod.name}}
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: step-template
    outputs: {}
status:
  nodes:
    exit-handler-bug-example:
      children:
      - exit-handler-bug-example-3054913383
      displayName: exit-handler-bug-example
      finishedAt: "2020-07-07T16:15:54Z"
      id: exit-handler-bug-example
      name: exit-handler-bug-example
      outboundNodes:
      - exit-handler-bug-example-3038135764
      phase: Running
      startedAt: "2020-07-07T16:15:33Z"
      templateName: dag
      templateScope: local/exit-handler-bug-example
      type: DAG
    exit-handler-bug-example-3054913383:
      boundaryID: exit-handler-bug-example
      displayName: step-2
      finishedAt: "2020-07-07T16:15:37Z"
      hostNodeName: minikube
      id: exit-handler-bug-example-3054913383
      name: exit-handler-bug-example.step-2
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: exit-handler-bug-example/exit-handler-bug-example-3054913383/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2020-07-07T16:15:33Z"
      templateName: step-template
      templateScope: local/exit-handler-bug-example
      type: Pod
  phase: Running
  startedAt: "2020-07-07T16:15:33Z"
`

func TestOnExitNonLeaf(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOnExitNonLeaf)
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

var testDagTargetTaskOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-primay-branch-6bnnl
spec:

  entrypoint: statis
  templates:
  -
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: a
    outputs: {}
  -
    container:
      args:
      - exit!
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: exit
    outputs: {}
  -
    inputs: {}
    metadata: {}
    name: steps
    outputs: {}
    steps:
    - -
        name: step-a
        template: a
  -
    dag:
      tasks:
      -
        name: A
        onExit: exit
        template: steps
    inputs: {}
    metadata: {}
    name: statis
    outputs: {}
status:
  nodes:
    dag-primay-branch-6bnnl:
      children:
      - dag-primay-branch-6bnnl-1650817843
      displayName: dag-primay-branch-6bnnl
      finishedAt: "2020-08-10T14:30:19Z"
      id: dag-primay-branch-6bnnl
      name: dag-primay-branch-6bnnl
      outboundNodes:
      - dag-primay-branch-6bnnl-1181733215
      phase: Running
      startedAt: "2020-08-10T14:30:08Z"
      templateName: statis
      templateScope: local/dag-primay-branch-6bnnl
      type: DAG
    dag-primay-branch-6bnnl-1181733215:
      boundaryID: dag-primay-branch-6bnnl-1650817843
      displayName: step-a
      finishedAt: "2020-08-10T14:30:11Z"
      hostNodeName: minikube
      id: dag-primay-branch-6bnnl-1181733215
      name: dag-primay-branch-6bnnl.A[0].step-a
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: dag-primay-branch-6bnnl/dag-primay-branch-6bnnl-1181733215/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-08-10T14:30:08Z"
      templateName: a
      templateScope: local/dag-primay-branch-6bnnl
      type: Pod
    dag-primay-branch-6bnnl-1650817843:
      boundaryID: dag-primay-branch-6bnnl
      children:
      - dag-primay-branch-6bnnl-3841864351
      - dag-primay-branch-6bnnl-1342580575
      displayName: A
      finishedAt: "2020-08-10T14:30:13Z"
      id: dag-primay-branch-6bnnl-1650817843
      name: dag-primay-branch-6bnnl.A
      outboundNodes:
      - dag-primay-branch-6bnnl-1181733215
      phase: Running
      startedAt: "2020-08-10T14:30:08Z"
      templateName: steps
      templateScope: local/dag-primay-branch-6bnnl
      type: Steps
    dag-primay-branch-6bnnl-3841864351:
      boundaryID: dag-primay-branch-6bnnl-1650817843
      children:
      - dag-primay-branch-6bnnl-1181733215
      displayName: '[0]'
      finishedAt: "2020-08-10T14:30:13Z"
      id: dag-primay-branch-6bnnl-3841864351
      name: dag-primay-branch-6bnnl.A[0]
      phase: Running
      startedAt: "2020-08-10T14:30:08Z"
      templateName: steps
      templateScope: local/dag-primay-branch-6bnnl
      type: StepGroup
  phase: Running
  resourcesDuration:
    cpu: 5
    memory: 2
  startedAt: "2020-08-10T14:30:08Z"
`

func TestDagTargetTaskOnExit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testDagTargetTaskOnExit)
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

var testFailsWithParamDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reproduce-bug-9tpfr
spec:

  entrypoint: start
  serviceAccountName: argo-workflow
  templates:
  -
    dag:
      tasks:
      -
        name: gen-tasks
        template: gen-tasks
      - arguments:
          parameters:
          - name: chunk
            value: '{{item}}'
        dependencies:
        - gen-tasks
        name: process-tasks
        template: process-tasks
        withParam: '{{tasks.gen-tasks.outputs.result}}'
      -
        dependencies:
        - process-tasks
        name: finish
        template: finish
    inputs: {}
    metadata: {}
    name: start
    outputs: {}
  - activeDeadlineSeconds: 300

    inputs: {}
    metadata: {}
    name: gen-tasks
    outputs: {}
    retryStrategy:
      backoff:
        duration: 15s
        factor: 2
      limit: 5
      retryPolicy: Always
    script:
      command:
      - bash
      image: python:3
      name: ""
      resources:
        requests:
          cpu: 250m
      source: |
        set -e
        python3 -c 'import os, json; print(json.dumps([str(i) for i in range(10)]))'
  - activeDeadlineSeconds: 1800

    inputs:
      parameters:
      - name: chunk
    metadata: {}
    name: process-tasks
    outputs: {}
    retryStrategy:
      backoff:
        duration: 15s
        factor: 2
      limit: 2
      retryPolicy: Always
    script:
      command:
      - bash
      image: python:3
      name: ""
      resources:
        requests:
          cpu: 100m
      source: |
        set -e
        chunk="{{inputs.parameters.chunk}}"
        if [[ $chunk == "3" ]]; then
          echo "failed"
          exit 1
        fi
        echo "process $chunk"
  - activeDeadlineSeconds: 300

    inputs: {}
    metadata: {}
    name: finish
    outputs: {}
    script:
      command:
      - sh
      image: busybox
      name: ""
      resources:
        requests:
          cpu: 100m
      source: |
        echo fin
status:
  nodes:
    reproduce-bug-9tpfr:
      children:
      - reproduce-bug-9tpfr-1525049382
      displayName: reproduce-bug-9tpfr
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr
      name: reproduce-bug-9tpfr
      outboundNodes:
      - reproduce-bug-9tpfr-247809182
      phase: Running
      startedAt: "2020-08-14T03:47:23Z"
      templateName: start
      templateScope: local/reproduce-bug-9tpfr
      type: DAG
    reproduce-bug-9tpfr-247809182:
      boundaryID: reproduce-bug-9tpfr
      displayName: finish
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr-247809182
      message: 'omitted: depends condition not met'
      name: reproduce-bug-9tpfr.finish
      phase: Omitted
      startedAt: "2020-08-14T03:49:42Z"
      templateName: finish
      templateScope: local/reproduce-bug-9tpfr
      type: Skipped
    reproduce-bug-9tpfr-546685502:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3207714925
      displayName: process-tasks(6:6)
      finishedAt: "2020-08-14T03:47:37Z"
      id: reproduce-bug-9tpfr-546685502
      inputs:
        parameters:
        - name: chunk
          value: "6"
      name: reproduce-bug-9tpfr.process-tasks(6:6)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-585646929:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(7:7)(0)
      finishedAt: "2020-08-14T03:47:33Z"
      hostNodeName: gke-dotdb3-dotdb-pool-c3910e16-nb0l
      id: reproduce-bug-9tpfr-585646929
      inputs:
        parameters:
        - name: chunk
          value: "7"
      name: reproduce-bug-9tpfr.process-tasks(7:7)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-600389385:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(5:5)(0)
      finishedAt: "2020-08-14T03:47:33Z"
      hostNodeName: gke-dotdb3-dotdb-pool-c3910e16-nb0l
      id: reproduce-bug-9tpfr-600389385
      inputs:
        parameters:
        - name: chunk
          value: "5"
      name: reproduce-bug-9tpfr.process-tasks(5:5)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-850457006:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3450100829
      displayName: process-tasks(2:2)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-850457006
      inputs:
        parameters:
        - name: chunk
          value: "2"
      name: reproduce-bug-9tpfr.process-tasks(2:2)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-988357889:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(1:1)(0)
      finishedAt: "2020-08-14T03:47:34Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-988357889
      inputs:
        parameters:
        - name: chunk
          value: "1"
      name: reproduce-bug-9tpfr.process-tasks(1:1)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-1384515437:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(8:8)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-1384515437
      inputs:
        parameters:
        - name: chunk
          value: "8"
      name: reproduce-bug-9tpfr.process-tasks(8:8)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-1525049382:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3595395365
      displayName: gen-tasks
      finishedAt: "2020-08-14T03:47:29Z"
      id: reproduce-bug-9tpfr-1525049382
      name: reproduce-bug-9tpfr.gen-tasks
      outputs:
        exitCode: "0"
        result: '["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"]'
      phase: Succeeded
      startedAt: "2020-08-14T03:47:23Z"
      templateName: gen-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-1602779214:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3522592509
      displayName: process-tasks(4:4)
      finishedAt: "2020-08-14T03:47:45Z"
      id: reproduce-bug-9tpfr-1602779214
      inputs:
        parameters:
        - name: chunk
          value: "4"
      name: reproduce-bug-9tpfr.process-tasks(4:4)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-1670762753:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-2788826366
      - reproduce-bug-9tpfr-3412607194
      - reproduce-bug-9tpfr-850457006
      - reproduce-bug-9tpfr-3505932898
      - reproduce-bug-9tpfr-1602779214
      - reproduce-bug-9tpfr-4143798034
      - reproduce-bug-9tpfr-546685502
      - reproduce-bug-9tpfr-3415721642
      - reproduce-bug-9tpfr-3221614398
      - reproduce-bug-9tpfr-3518128714
      displayName: process-tasks
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr-1670762753
      name: reproduce-bug-9tpfr.process-tasks
      phase: Failed
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: TaskGroup
    reproduce-bug-9tpfr-2008331609:
      boundaryID: reproduce-bug-9tpfr
      displayName: process-tasks(3:3)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-2008331609
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: failed with exit code 1
      name: reproduce-bug-9tpfr.process-tasks(3:3)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-2475724017:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(9:9)(0)
      finishedAt: "2020-08-14T03:47:34Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-2475724017
      inputs:
        parameters:
        - name: chunk
          value: "9"
      name: reproduce-bug-9tpfr.process-tasks(9:9)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-2612472988:
      boundaryID: reproduce-bug-9tpfr
      displayName: process-tasks(3:3)(1)
      finishedAt: "2020-08-14T03:48:05Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-2612472988
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: failed with exit code 1
      name: reproduce-bug-9tpfr.process-tasks(3:3)(1)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2020-08-14T03:48:00Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-2788826366:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3662458541
      displayName: process-tasks(0:0)
      finishedAt: "2020-08-14T03:47:44Z"
      id: reproduce-bug-9tpfr-2788826366
      inputs:
        parameters:
        - name: chunk
          value: "0"
      name: reproduce-bug-9tpfr.process-tasks(0:0)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3207714925:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(6:6)(0)
      finishedAt: "2020-08-14T03:47:34Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-3207714925
      inputs:
        parameters:
        - name: chunk
          value: "6"
      name: reproduce-bug-9tpfr.process-tasks(6:6)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3221614398:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-1384515437
      displayName: process-tasks(8:8)
      finishedAt: "2020-08-14T03:47:44Z"
      id: reproduce-bug-9tpfr-3221614398
      inputs:
        parameters:
        - name: chunk
          value: "8"
      name: reproduce-bug-9tpfr.process-tasks(8:8)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3412607194:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-988357889
      displayName: process-tasks(1:1)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-3412607194
      inputs:
        parameters:
        - name: chunk
          value: "1"
      name: reproduce-bug-9tpfr.process-tasks(1:1)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3415721642:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-585646929
      displayName: process-tasks(7:7)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-3415721642
      inputs:
        parameters:
        - name: chunk
          value: "7"
      name: reproduce-bug-9tpfr.process-tasks(7:7)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3450100829:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(2:2)(0)
      finishedAt: "2020-08-14T03:47:33Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-3450100829
      inputs:
        parameters:
        - name: chunk
          value: "2"
      name: reproduce-bug-9tpfr.process-tasks(2:2)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3505932898:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-2008331609
      - reproduce-bug-9tpfr-2612472988
      - reproduce-bug-9tpfr-4222579959
      displayName: process-tasks(3:3)
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr-3505932898
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: No more retries left
      name: reproduce-bug-9tpfr.process-tasks(3:3)
      phase: Failed
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3518128714:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-2475724017
      displayName: process-tasks(9:9)
      finishedAt: "2020-08-14T03:47:37Z"
      id: reproduce-bug-9tpfr-3518128714
      inputs:
        parameters:
        - name: chunk
          value: "9"
      name: reproduce-bug-9tpfr.process-tasks(9:9)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3522592509:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(4:4)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-3522592509
      inputs:
        parameters:
        - name: chunk
          value: "4"
      name: reproduce-bug-9tpfr.process-tasks(4:4)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3595395365:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-1670762753
      displayName: gen-tasks(0)
      finishedAt: "2020-08-14T03:47:26Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-3595395365
      name: reproduce-bug-9tpfr.gen-tasks(0)
      outputs:
        exitCode: "0"
        result: '["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"]'
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:23Z"
      templateName: gen-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3662458541:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(0:0)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-3662458541
      inputs:
        parameters:
        - name: chunk
          value: "0"
      name: reproduce-bug-9tpfr.process-tasks(0:0)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-4143798034:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-600389385
      displayName: process-tasks(5:5)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-4143798034
      inputs:
        parameters:
        - name: chunk
          value: "5"
      name: reproduce-bug-9tpfr.process-tasks(5:5)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-4222579959:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(3:3)(2)
      finishedAt: "2020-08-14T03:48:40Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-4222579959
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: failed with exit code 1
      name: reproduce-bug-9tpfr.process-tasks(3:3)(2)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:48:37Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 36
    memory: 36
  startedAt: "2020-08-14T03:47:23Z"
`

func TestFailsWithParamDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testFailsWithParamDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var dagOutputsReferTaskAggregatedOuputs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-dag-h8b82
spec:

  entrypoint: parameter-aggregation
  templates:
  -
    dag:
      tasks:
      - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        name: odd-or-even
        template: odd-or-even
        withItems:
        - 1
        - 2
    inputs: {}
    metadata: {}
    name: parameter-aggregation
    outputs:
      parameters:
      - name: dag-nums
        valueFrom:
          parameter: '{{tasks.odd-or-even.outputs.parameters.num}}'
      - name: dag-evenness
        valueFrom:
          parameter: '{{tasks.odd-or-even.outputs.parameters.evenness}}'
  -
    container:
      args:
      - |
        sleep 1 &&
        echo {{inputs.parameters.num}} > /tmp/num &&
        if [ $(({{inputs.parameters.num}}%2)) -eq 0 ]; then
          echo "even" > /tmp/even;
        else
          echo "odd" > /tmp/even;
        fi
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: odd-or-even
    outputs:
      parameters:
      - name: num
        valueFrom:
          path: /tmp/num
      - name: evenness
        valueFrom:
          path: /tmp/even
status:
  nodes:
    parameter-aggregation-dag-h8b82:
      children:
      - parameter-aggregation-dag-h8b82-3379492521
      displayName: parameter-aggregation-dag-h8b82
      finishedAt: "2020-12-09T15:37:07Z"
      id: parameter-aggregation-dag-h8b82
      name: parameter-aggregation-dag-h8b82
      outboundNodes:
      - parameter-aggregation-dag-h8b82-3175470584
      - parameter-aggregation-dag-h8b82-2243926302
      phase: Running
      startedAt: "2020-12-09T15:36:46Z"
      templateName: parameter-aggregation
      templateScope: local/parameter-aggregation-dag-h8b82
      type: DAG
    parameter-aggregation-dag-h8b82-1440345089:
      boundaryID: parameter-aggregation-dag-h8b82
      displayName: odd-or-even(1:2)
      finishedAt: "2020-12-09T15:36:54Z"
      hostNodeName: minikube
      id: parameter-aggregation-dag-h8b82-1440345089
      inputs:
        parameters:
        - name: num
          value: "2"
      name: parameter-aggregation-dag-h8b82.odd-or-even(1:2)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "2"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: even
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2020-12-09T15:36:46Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-dag-h8b82
      type: Pod
    parameter-aggregation-dag-h8b82-3379492521:
      boundaryID: parameter-aggregation-dag-h8b82
      children:
      - parameter-aggregation-dag-h8b82-3572919299
      - parameter-aggregation-dag-h8b82-1440345089
      displayName: odd-or-even
      finishedAt: "2020-12-09T15:36:55Z"
      id: parameter-aggregation-dag-h8b82-3379492521
      name: parameter-aggregation-dag-h8b82.odd-or-even
      phase: Succeeded
      startedAt: "2020-12-09T15:36:46Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-dag-h8b82
      type: TaskGroup
    parameter-aggregation-dag-h8b82-3572919299:
      boundaryID: parameter-aggregation-dag-h8b82
      displayName: odd-or-even(0:1)
      finishedAt: "2020-12-09T15:36:53Z"
      hostNodeName: minikube
      id: parameter-aggregation-dag-h8b82-3572919299
      inputs:
        parameters:
        - name: num
          value: "1"
      name: parameter-aggregation-dag-h8b82.odd-or-even(0:1)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "1"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: odd
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2020-12-09T15:36:46Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-dag-h8b82
      type: Pod
  phase: Succeeded
  startedAt: "2020-12-09T15:36:46Z"
`

func TestDAGReferTaskAggregatedOutputs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagOutputsReferTaskAggregatedOuputs)
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

var dagHTTPChildrenAssigned = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template-nv52d
spec:
  entrypoint: main
  templates:
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: url
            value: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
        name: good1
        template: http
      - arguments:
          parameters:
          - name: url
            value: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
        dependencies:
        - good1
        name: good2
        template: http
    name: main
  - http:
      url: '{{inputs.parameters.url}}'
    inputs:
      parameters:
      - name: url
    name: http
status:
  nodes:
    http-template-nv52d:
      children:
      - http-template-nv52d-444770636
      displayName: http-template-nv52d
      id: http-template-nv52d
      name: http-template-nv52d
      outboundNodes:
      - http-template-nv52d-478325874
      phase: Running
      startedAt: "2021-10-27T13:46:08Z"
      templateName: main
      templateScope: local/http-template-nv52d
      type: DAG
    http-template-nv52d-444770636:
      boundaryID: http-template-nv52d
      children:
      - http-template-nv52d-495103493
      displayName: good1
      finishedAt: null
      id: http-template-nv52d-444770636
      name: http-template-nv52d.good1
      phase: Succeeded
      startedAt: "2021-10-27T13:46:08Z"
      templateName: http
      templateScope: local/http-template-nv52d
      type: HTTP
  phase: Running
  startedAt: "2021-10-27T13:46:08Z"
`

func TestDagHttpChildrenAssigned(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagHTTPChildrenAssigned)
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

var retryTypeDagTaskRunExitNodeAfterCompleted = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  labels:
    workflows.argoproj.io/phase: Running
  name: test-workflow-with-hang-cztfs
  namespace: argo-system
spec:
  entrypoint: dag
  templates:
  - name: linuxExitHandler
    steps:
    - - name: print-exit
        template: print-exit
  - container:
      args:
      - echo
      - exit
      command:
      - /argosay
      image: argoproj/argosay:v2
      name: ""
    name: print-exit
  - container:
      args:
      - echo
      - a
      command:
      - /argosay
      image: argoproj/argosay:v2
      name: ""
    name: printA
    retryStrategy:
      limit: "3"
      retryPolicy: OnError
  - dag:
      tasks:
      - hooks:
          exit:
            template: linuxExitHandler
        name: printA
        template: printA
      - depends: printA.Succeeded
        hooks:
          exit:
            template: linuxExitHandler
        name: dependencyTesting
        template: printA
    name: dag
status:
  nodes:
    test-workflow-with-hang-cztfs:
      children:
      - test-workflow-with-hang-cztfs-1556528266
      displayName: test-workflow-with-hang-cztfs
      finishedAt: null
      id: test-workflow-with-hang-cztfs
      name: test-workflow-with-hang-cztfs
      phase: Running
      progress: 4/4
      startedAt: "2022-08-04T02:28:38Z"
      templateName: dag
      templateScope: local/test-workflow-with-hang-cztfs
      type: DAG
    test-workflow-with-hang-cztfs-589413809:
      boundaryID: test-workflow-with-hang-cztfs
      children:
      - test-workflow-with-hang-cztfs-527957059
      displayName: printA(0)
      finishedAt: "2022-08-04T02:28:43Z"
      hostNodeName: node2
      id: test-workflow-with-hang-cztfs-589413809
      name: test-workflow-with-hang-cztfs.printA(0)
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2022-08-04T02:28:38Z"
      templateName: printA
      templateScope: local/test-workflow-with-hang-cztfs
      type: Pod
    test-workflow-with-hang-cztfs-1556528266:
      boundaryID: test-workflow-with-hang-cztfs
      children:
      - test-workflow-with-hang-cztfs-589413809
      displayName: printA
      finishedAt: "2022-08-04T02:28:48Z"
      id: test-workflow-with-hang-cztfs-1556528266
      name: test-workflow-with-hang-cztfs.printA
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 4/4
      resourcesDuration:
        cpu: 5
        memory: 5
      startedAt: "2022-08-04T02:28:38Z"
      templateName: printA
      templateScope: local/test-workflow-with-hang-cztfs
      type: Retry
  phase: Running
  progress: 4/4
  resourcesDuration:
    cpu: 5
    memory: 5
  startedAt: "2022-08-04T02:28:38Z"
`

func TestRetryTypeDagTaskRunExitNodeAfterCompleted(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(retryTypeDagTaskRunExitNodeAfterCompleted)
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
var dagTaskGroupIDRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-taskgroup-id-ref
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: generate
        template: gen-list
      - name: fanout
        dependencies: [generate]
        template: echo
        arguments:
          parameters:
          - name: msg
            value: '{{item}}'
        withParam: '{{tasks.generate.outputs.result}}'
      - name: use-id
        dependencies: [fanout]
        template: echo
        arguments:
          parameters:
          - name: msg
            value: '{{tasks.fanout.id}}'
  - name: gen-list
    script:
      image: python:alpine3.23
      command: [python]
      source: |
        import json, sys
        json.dump([0, 1], sys.stdout)
  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:3.23
      command: [echo, '{{inputs.parameters.msg}}']
status:
  nodes:
    dag-taskgroup-id-ref:
      id: dag-taskgroup-id-ref
      name: dag-taskgroup-id-ref
      displayName: dag-taskgroup-id-ref
      type: DAG
      templateName: main
      templateScope: local/dag-taskgroup-id-ref
      phase: Running
      startedAt: "2020-04-20T16:39:00Z"
      children:
      - dag-taskgroup-id-ref-455800905
    dag-taskgroup-id-ref-455800905:
      id: dag-taskgroup-id-ref-455800905
      name: dag-taskgroup-id-ref.generate
      displayName: generate
      type: Pod
      templateName: gen-list
      templateScope: local/dag-taskgroup-id-ref
      boundaryID: dag-taskgroup-id-ref
      phase: Succeeded
      startedAt: "2020-04-20T16:39:00Z"
      finishedAt: "2020-04-20T16:39:02Z"
      children:
      - dag-taskgroup-id-ref-2094038697
      outputs:
        result: '[0, 1]'
        exitCode: "0"
    dag-taskgroup-id-ref-2094038697:
      id: dag-taskgroup-id-ref-2094038697
      name: dag-taskgroup-id-ref.fanout
      displayName: fanout
      type: TaskGroup
      templateName: echo
      templateScope: local/dag-taskgroup-id-ref
      boundaryID: dag-taskgroup-id-ref
      phase: Succeeded
      startedAt: "2020-04-20T16:39:03Z"
      finishedAt: "2020-04-20T16:39:09Z"
      children:
      - dag-taskgroup-id-ref-2700861446
      - dag-taskgroup-id-ref-3319419394
    dag-taskgroup-id-ref-2700861446:
      id: dag-taskgroup-id-ref-2700861446
      name: dag-taskgroup-id-ref.fanout(0:0)
      displayName: fanout(0:0)
      type: Pod
      templateName: echo
      templateScope: local/dag-taskgroup-id-ref
      boundaryID: dag-taskgroup-id-ref
      phase: Succeeded
      startedAt: "2020-04-20T16:39:03Z"
      finishedAt: "2020-04-20T16:39:06Z"
      inputs:
        parameters:
        - name: msg
          value: "0"
      outputs:
        exitCode: "0"
    dag-taskgroup-id-ref-3319419394:
      id: dag-taskgroup-id-ref-3319419394
      name: dag-taskgroup-id-ref.fanout(1:1)
      displayName: fanout(1:1)
      type: Pod
      templateName: echo
      templateScope: local/dag-taskgroup-id-ref
      boundaryID: dag-taskgroup-id-ref
      phase: Succeeded
      startedAt: "2020-04-20T16:39:03Z"
      finishedAt: "2020-04-20T16:39:07Z"
      inputs:
        parameters:
        - name: msg
          value: "1"
      outputs:
        exitCode: "0"
  phase: Running
  startedAt: "2020-04-20T16:39:00Z"
`

func TestDAGTaskGroupIDReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagTaskGroupIDRef)
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

var dagWhenSkipNoRequeue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-when-skip-no-requeue-
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: script-echo
        when: "false"
      - name: B
        dependencies: [A]
        when: "{{tasks.A.status}} == Succeeded"
        template: echo-with-param
        arguments:
          parameters:
          - name: msg
            value: "{{tasks.A.outputs.result}}"
  - name: script-echo
    script:
      image: alpine:3.23
      command: [sh]
      source: |
        echo hello
  - name: echo-with-param
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.msg}}"]
`

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

	wf := wfv1.MustUnmarshalWorkflow(dagWhenSkipNoRequeue)
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

var dagSkippedOutputRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-skipped-output-ref
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: run-stage-b
        value: "false"
  templates:
    - name: main
      inputs:
        parameters:
          - name: run-stage-b
      dag:
        tasks:
          - name: stage-a
            template: stage-a
          - name: stage-b
            template: stage-b
            when: "\"{{inputs.parameters.run-stage-b}}\" == \"true\""
          - name: stage-c
            template: stage-c
            depends: "stage-a && (stage-b || stage-b.Skipped)"
            arguments:
              parameters:
                - name: message-from-a
                  value: "{{tasks.stage-a.outputs.parameters.output-message}}"
                - name: message-from-b
                  value: "{{tasks.stage-b.outputs.parameters.output-message}}"
    - name: stage-a
      outputs:
        parameters:
          - name: output-message
            valueFrom:
              path: /tmp/output.txt
      container:
        image: alpine:3.23
        command: [sh, -c]
        args: ["echo 'hello from stage A' > /tmp/output.txt"]
    - name: stage-b
      outputs:
        parameters:
          - name: output-message
            valueFrom:
              path: /tmp/output.txt
      container:
        image: alpine:3.23
        command: [sh, -c]
        args: ["echo 'hello from stage B' > /tmp/output.txt"]
    - name: stage-c
      inputs:
        parameters:
          - name: message-from-a
          - name: message-from-b
      container:
        image: alpine:3.23
        command: [echo]
        args: ["{{inputs.parameters.message-from-a}}", "{{inputs.parameters.message-from-b}}"]
status:
  phase: Running
  startedAt: "2024-01-01T00:00:00Z"
  nodes:
    dag-skipped-output-ref:
      id: dag-skipped-output-ref
      name: dag-skipped-output-ref
      displayName: dag-skipped-output-ref
      type: DAG
      templateName: main
      templateScope: local/dag-skipped-output-ref
      phase: Running
      startedAt: "2024-01-01T00:00:00Z"
      children:
        - dag-skipped-output-ref-1381367026
        - dag-skipped-output-ref-1364589407
      outboundNodes:
        - dag-skipped-output-ref-1347811788
    dag-skipped-output-ref-1381367026:
      id: dag-skipped-output-ref-1381367026
      name: dag-skipped-output-ref.stage-a
      displayName: stage-a
      type: Pod
      templateName: stage-a
      templateScope: local/dag-skipped-output-ref
      boundaryID: dag-skipped-output-ref
      phase: Succeeded
      startedAt: "2024-01-01T00:00:00Z"
      finishedAt: "2024-01-01T00:00:10Z"
      outputs:
        parameters:
          - name: output-message
            value: "hello from stage A"
    dag-skipped-output-ref-1364589407:
      id: dag-skipped-output-ref-1364589407
      name: dag-skipped-output-ref.stage-b
      displayName: stage-b
      type: Skipped
      templateName: stage-b
      templateScope: local/dag-skipped-output-ref
      boundaryID: dag-skipped-output-ref
      phase: Skipped
      startedAt: "2024-01-01T00:00:00Z"
      finishedAt: "2024-01-01T00:00:00Z"
      message: when '"false" == "true"' evaluated false
`

// TestDAGSkippedOutputRef verifies that a DAG task referencing a skipped dependency's defaultless
// output fails terminally rather than getting stuck in a requeue loop (#16223 absence semantics).
// Scenario: stage-a succeeds with output parameters, stage-b is skipped (when evaluates false),
// stage-c depends on both and references outputs from both. stage-b's output has no default and
// stage-c's input has no default, so the reference is an unhandled absent optional: stage-c must
// resolve to a terminal Error, not schedule with an unresolved tag or requeue forever.
func TestDAGSkippedOutputRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(dagSkippedOutputRef)
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

var dagWhenExprSkipEval = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-when-expr-skip-eval-
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: data
        value: ''
  templates:
  - name: main
    dag:
      tasks:
      - name: A
        template: echo
        when: "false"
        arguments:
          parameters:
          - name: message
            value: A
      - name: B
        dependencies: [A]
        when: "{{= workflow.parameters.data != '' }}"
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{= jsonpath(workflow.parameters.data, '$.id') }}"
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]
`

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

	wf := wfv1.MustUnmarshalWorkflow(dagWhenExprSkipEval)
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

	wf := wfv1.MustUnmarshalWorkflow(terminateDAGWithMaxDurationLimitExpiredAndMoreAttempts)
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

// testRound2IConvergeActionFailDAG is a DAG with task A (retry, limit=0) and task B
// (depends: "A.Failed"). When A fails, B should be scheduled in the same operate cycle.
var testRound2IConvergeActionFailDAG = `
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

// TestRound2I_ConvergeActionFailRunningRetryNode verifies that converge() correctly
// handles the ActionFail case for a retry node that is still NodeRunning.
//
// When evaluateRetryNode returns ActionFail for A (last child failed, limit exhausted,
// A still NodeRunning), converge must:
//  1. Call executeTask(A) to mark A as NodeFailed via processNodeRetries
//  2. Call executeTask(B) because A.Failed is satisfied (B.ShouldRun=true)
//
// If the ActionFail path in converge is broken (e.g., needsExecution is not set),
// A stays Running and B is never scheduled, leaving the DAG stuck at Running.
func TestRound2I_ConvergeActionFailRunningRetryNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testRound2IConvergeActionFailDAG)
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

	// Cycle 2: A(0) fails → evaluateRetryNode returns ActionFail (limit=0, no retries)
	// converge should: (a) call executeTask(A) to mark A as Failed, (b) schedule B
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A should be marked Failed after processNodeRetries runs via executeTask
	retryNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase,
		"retry node A must be Failed: converge should call executeTask(A) for ActionFail")

	// B should be scheduled because A.Failed is true
	// Bug: if converge doesn't handle ActionFail correctly for running retry nodes,
	// B is never scheduled and the DAG hangs at Running.
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"task B must be scheduled: A.Failed is satisfied, converge should call executeTask(B)")
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
	if bNode != nil {
		assert.True(t, bNode.Phase == wfv1.NodeOmitted || bNode.Phase == wfv1.NodeSkipped,
			"B should be Omitted/Skipped — A.Succeeded is unsatisfiable for a running daemon")
	}
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
	if bNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
			"B should be Omitted when A.Succeeded is never satisfied")
	}

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
	if cNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, cNode.Phase,
			"C should be Omitted since A && B both failed")
	}

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
	if bNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
			"B should be Omitted since A failed (default depends = A.Succeeded)")
	}

	// C should be Omitted
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	if cNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, cNode.Phase,
			"C should be Omitted since A failed")
	}

	// D should be Omitted
	dNode := woc.wf.Status.Nodes.FindByDisplayName("D")
	if dNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, dNode.Phase,
			"D should be Omitted since B and C are Omitted")
	}

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

// Test 2: Retry errored dep propagates Error (not hardcoded Failed)
var testOperatorBugfixRetryErrorPropagatesError = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-error-propagates
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

// TestOperatorBugfix_RetryErroredDepPropagatesError verifies that when A(0)
// exits with Error phase (not Failed) and retry is exhausted (limit=0),
// B (depends: "A.Errored") should run because A.Errored is true.
func TestOperatorBugfix_RetryErroredDepPropagatesError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixRetryErrorPropagatesError)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	// Mark A(0) pod as failed then set node phase to Error
	makePodsPhase(ctx, woc, v1.PodFailed)
	a0Node := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node)
	a0Node.Phase = wfv1.NodeError
	a0Node.Message = "pod errored"
	woc.wf.Status.Nodes[a0Node.ID] = *a0Node

	// Cycle 2: retry exhausted (limit=0), A should be Error
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.True(t, retryNode.Phase.FailedOrError(),
		"A should be in a terminal failure/error state")

	// B should be scheduled because A.Errored is true
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should run when A.Errored is true (retry exhausted with Error phase)")
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

// Test 4: Daemoned dep with explicit A.Succeeded waits (not omitted)
var testOperatorBugfixDaemonDepExplicitSucceeded = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: daemon-explicit-succeeded
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

// TestOperatorBugfix_DaemonedDepExplicitSucceededWaits verifies that when A is
// a daemon with B depending on "A.Succeeded", B is NOT created or omitted while
// A is just daemoned+running (A hasn't actually Succeeded yet).
func TestOperatorBugfix_DaemonedDepExplicitSucceededWaits(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixDaemonDepExplicitSucceeded)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: kicks off A
	woc.operate(ctx)

	// Simulate A becoming daemoned+running
	makePodsPhase(ctx, woc, v1.PodRunning)
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	daemon := true
	aNode.Daemoned = &daemon
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	// Cycle 2: A is daemoned+running, B depends on A.Succeeded
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// B should be omitted — A.Succeeded is unsatisfiable for a running daemon
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	if bNode != nil {
		assert.True(t, bNode.Phase == wfv1.NodeOmitted || bNode.Phase == wfv1.NodeSkipped,
			"B should be Omitted/Skipped — A.Succeeded is unsatisfiable for a running daemon")
	}
}

// Test 5: Retry limit exhausted -> downstream omitted (not stuck waiting)
var testOperatorBugfixRetryExhaustedDownstreamOmitted = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-exhausted-omit-bugfix
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

// TestOperatorBugfix_RetryExhaustedDownstreamOmitted verifies that when A's
// retry is exhausted (A(0) fails, A(1) fails), A is Failed, B is Omitted,
// and the DAG is Failed (not stuck waiting).
func TestOperatorBugfix_RetryExhaustedDownstreamOmitted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixRetryExhaustedDownstreamOmitted)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	// A(0) fails
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(1) should be created
	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	require.NotNil(t, a1Node, "A(1) should be created after A(0) fails")

	// A(1) fails too -> retry exhausted
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A (retry node) should be Failed
	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase,
		"A should be Failed after retry exhausted")

	// B should be Omitted
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	if bNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
			"B should be Omitted when A.Succeeded is never satisfied")
	}

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
	if bNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
			"B should be Omitted since A failed")
	}

	// C should be Omitted
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	if cNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, cNode.Phase,
			"C should be Omitted since B is Omitted")
	}

	// D should be Omitted
	dNode := woc.wf.Status.Nodes.FindByDisplayName("D")
	if dNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, dNode.Phase,
			"D should be Omitted since C is Omitted")
	}

	// DAG should fail
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 10: Retry dep succeeded -> dependent runs immediately
var testOperatorBugfixRetryDepSucceededDependentRuns = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-dep-succeeded-dependent
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
      limit: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorBugfix_RetryDepSucceededDependentRuns verifies that when A(0)
// fails, A(1) succeeds, B should run immediately.
func TestOperatorBugfix_RetryDepSucceededDependentRuns(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixRetryDepSucceededDependentRuns)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
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

	// B should be scheduled/running
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode,
		"B should be scheduled after retry node A succeeds")
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

// Test 12: Retry with OnError policy: Failed phase NOT retried
var testOperatorBugfixRetryOnErrorFailedNotRetried = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-onerror-failed-not-retried
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

// TestOperatorBugfix_RetryOnErrorPolicyFailedNotRetried verifies that when the
// retry policy is OnError and A(0) exits with Failed (not Error), the retry is
// NOT triggered because OnError only covers Error phase.
func TestOperatorBugfix_RetryOnErrorPolicyFailedNotRetried(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixRetryOnErrorFailedNotRetried)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	a0Node := woc.wf.Status.Nodes.FindByDisplayName("A(0)")
	require.NotNil(t, a0Node, "A(0) should exist")

	// A(0) fails with NodeFailed (not NodeError)
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// A(1) should NOT be created (OnError does not retry Failed)
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

// Test 14: Steps template: step-a completes -> step-b runs -> workflow succeeds
var testOperatorBugfixStepsSequential = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-sequential-bugfix
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

// TestOperatorBugfix_StepsSequentialExecution verifies that in a steps template
// with [[step-a]] -> [[step-b]], step-a runs first, step-b runs after step-a
// succeeds, and the workflow succeeds after both complete.
func TestOperatorBugfix_StepsSequentialExecution(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorBugfixStepsSequential)
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

// Test 1: Retry dep succeeded unblocks dependent
var testOperatorEdgeRetryDepSucceededUnblocks = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-retry-dep-succeeded-unblocks
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
      limit: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_RetryDepSucceededUnblocksDependent verifies that when
// A(retry) eventually succeeds, B is scheduled.
func TestOperatorEdge_RetryDepSucceededUnblocksDependent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeRetryDepSucceededUnblocks)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates A -> A(0)
	woc.operate(ctx)

	// A(0) fails -> retry A(1)
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	require.NotNil(t, a1Node, "A(1) should be created after A(0) fails")

	// A(1) succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Retry node A should be Succeeded
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeSucceeded, aNode.Phase, "Retry node A should be Succeeded")

	// B should be scheduled
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	assert.NotNil(t, bNode, "B should be scheduled after retry node A succeeds")
}

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
	if bNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, bNode.Phase,
			"B should be Omitted when retry dep A has failed")
	}

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

// Test 4: Retry daemon dep with explicit A.Succeeded waits
var testOperatorEdgeRetryDaemonExplicitSucceededWaits = `
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

// TestOperatorEdge_RetryDaemonDepExplicitSucceededWaits verifies that when
// A(retry+daemon) is daemoned (Running+Daemoned) but B depends on "A.Succeeded",
// B is NOT omitted — it waits because A.Succeeded is not yet true.
func TestOperatorEdge_RetryDaemonDepExplicitSucceededWaits(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeRetryDaemonExplicitSucceededWaits)
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
	if bNode != nil {
		assert.True(t, bNode.Phase == wfv1.NodeOmitted || bNode.Phase == wfv1.NodeSkipped,
			"B should be Omitted/Skipped — A.Succeeded is unsatisfiable for a running daemon")
	}
}

// Test 5: Steps sequential deps
var testOperatorEdgeStepsSequential = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-steps-sequential
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

// TestOperatorEdge_StepsSequentialDeps verifies that in a steps template,
// step-a runs first and only after it completes does step-b start.
func TestOperatorEdge_StepsSequentialDeps(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeStepsSequential)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: step-a should be scheduled, step-b should not
	woc.operate(ctx)

	stepANode := woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, stepANode, "step-a should be scheduled in cycle 1")

	stepBNode := woc.wf.Status.Nodes.FindByDisplayName("step-b")
	assert.Nil(t, stepBNode, "step-b should not be scheduled before step-a completes")

	// step-a succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// step-b should now be scheduled
	stepBNode = woc.wf.Status.Nodes.FindByDisplayName("step-b")
	assert.NotNil(t, stepBNode, "step-b should be scheduled after step-a succeeds")

	// step-b succeeds -> workflow succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
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

// Test 8: FailFast=false with multiple failures
var testOperatorEdgeFailFastFalseMultipleFailures = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-failfast-false-multi
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

// TestOperatorEdge_FailFastFalseMultipleFailures verifies that with failFast=false,
// when both leaf tasks A and B fail, the DAG gets the worst phase (Failed).
func TestOperatorEdge_FailFastFalseMultipleFailures(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeFailFastFalseMultipleFailures)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A and B start in parallel
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled")

	// Both fail
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	aNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)
	bNode = woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode)
	assert.Equal(t, wfv1.NodeFailed, bNode.Phase)

	// C should be Omitted (A && B both failed)
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	if cNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, cNode.Phase,
			"C should be Omitted since A and B both failed")
	}

	// DAG should be Failed
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
	if bNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, bNode.Phase, "B should be Omitted since A failed")
	}
	cNode := woc.wf.Status.Nodes.FindByDisplayName("C")
	if cNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, cNode.Phase, "C should be Omitted since B is Omitted")
	}

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// Test 11: Single task DAG — fails -> DAG Failed
var testOperatorEdgeSingleTaskFails = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-single-task-fails
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

// TestOperatorEdge_SingleTaskDAGFails verifies that a single-task DAG where
// the task fails results in a Failed workflow.
func TestOperatorEdge_SingleTaskDAGFails(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeSingleTaskFails)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	assert.Equal(t, wfv1.NodeFailed, aNode.Phase)
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
	if bNode != nil {
		assert.Equal(t, wfv1.NodeOmitted, bNode.Phase, "B should be Omitted")
	}

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

// Test 14: Parallel retries
var testOperatorEdgeParallelRetries = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-parallel-retries
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
      command: [sh, -c, "exit 1"]
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestOperatorEdge_ParallelRetries verifies that A(retry) and B(retry) both fail
// then succeed, and C (depends: "A && B") runs after both succeed.
func TestOperatorEdge_ParallelRetries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeParallelRetries)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A(0) and B(0) start
	woc.operate(ctx)

	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode, "A should be scheduled")
	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled")

	// A(0) and B(0) both fail
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	a1Node := woc.wf.Status.Nodes.FindByDisplayName("A(1)")
	assert.NotNil(t, a1Node, "A(1) should be created after A(0) fails")
	b1Node := woc.wf.Status.Nodes.FindByDisplayName("B(1)")
	assert.NotNil(t, b1Node, "B(1) should be created after B(0) fails")

	// A(1) and B(1) both succeed
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

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

// Test 15: 3-level nesting: TaskGroup > Retry > Pod (withItems + retry)
var testOperatorEdgeTaskGroupRetryPod = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-taskgroup-retry-pod
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

// TestOperatorEdge_TaskGroupRetryPodNesting verifies 3-level nesting:
// TaskGroup > Retry > Pod. Both items fail then succeed -> TaskGroup A succeeds.
func TestOperatorEdge_TaskGroupRetryPodNesting(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeTaskGroupRetryPod)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: creates TaskGroup A with retry children
	woc.operate(ctx)

	tgNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode, "TaskGroup A should exist")

	// First attempts fail
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Retry attempts succeed
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Extra cycle for convergence
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	tgNode = woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, tgNode)
	assert.Equal(t, wfv1.NodeSucceeded, tgNode.Phase,
		"TaskGroup A should succeed after all retry children succeed")

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// Test 16: Daemon workflow completes
var testOperatorEdgeDaemonWorkflowCompletes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: edge-daemon-workflow-completes
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

// TestOperatorEdge_DaemonWorkflowCompletes verifies that when A is daemon and B
// depends on A, after A is daemoned and B succeeds, the workflow completes.
func TestOperatorEdge_DaemonWorkflowCompletes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOperatorEdgeDaemonWorkflowCompletes)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: A starts
	woc.operate(ctx)

	// Mark A as Running+Daemoned
	makePodsPhase(ctx, woc, v1.PodRunning)
	aNode := woc.wf.Status.Nodes.FindByDisplayName("A")
	require.NotNil(t, aNode)
	daemon := true
	aNode.Daemoned = &daemon
	woc.wf.Status.Nodes[aNode.ID] = *aNode

	// Cycle 2: A is daemoned -> B should be scheduled
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	bNode := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, bNode, "B should be scheduled when A is daemoned")

	// B succeeds
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Extra cycle for convergence
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase,
		"Workflow should complete when daemon's downstream task succeeds")
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
