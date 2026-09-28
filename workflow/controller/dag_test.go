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
)

// TestDagXfail verifies a DAG can fail properly
func TestDagXfail(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag_xfail.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
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
	statusMap := map[string]v1.PodPhase{"Succeeded": v1.PodSucceeded, "Failed": v1.PodFailed}
	var closer context.CancelFunc
	var controller *WorkflowController
	for _, status := range []string{"Succeeded", "Failed", "Skipped"} {
		fmt.Printf("\n\n\nCurrent status %s\n\n\n", status)
		ctx := logging.TestContext(t.Context())
		closer, controller = newController(ctx)
		wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

		// If the status is "skipped" skip the root node.
		var wfString string
		if status == "Skipped" {
			wfString = fmt.Sprintf(dynamicSingleDag, status, `when: "False == True"`, status)
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

	woc = newWorkflowOperationCtx(ctx, wf, controller)
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

	expanded, err := expandTask(ctx, task, map[string]any{})
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
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
	}

	// Task A is running
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {Name: d.taskNodeName("A"), Phase: wfv1.NodeRunning},
			},
		},
	}

	// Task B should not proceed, task A is still running
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Task A succeeded
	d.wf.Status.Nodes[d.taskNodeID("A")] = wfv1.NodeStatus{Name: d.taskNodeName("A"), Phase: wfv1.NodeSucceeded}

	// Task B and C should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	execute, proceed, err = d.evaluateDependsLogic(ctx, "C")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	// Other tasks should not
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-1")
	require.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Tasks B succeeded, C failed
	d.wf.Status.Nodes[d.taskNodeID("B")] = wfv1.NodeStatus{Name: d.taskNodeName("B"), Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("C")] = wfv1.NodeStatus{Name: d.taskNodeName("C"), Phase: wfv1.NodeFailed}

	// Tasks should-execute-1 and should-execute-2 should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-1")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-2")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	// Task should-not-execute should proceed, but not execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-not-execute")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.False(t, execute)

	// Tasks should-execute-1 and should-execute-2 succeeded, should-not-execute skipped
	d.wf.Status.Nodes[d.taskNodeID("should-execute-1")] = wfv1.NodeStatus{Name: d.taskNodeName("should-execute-1"), Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("should-execute-2")] = wfv1.NodeStatus{Name: d.taskNodeName("should-execute-2"), Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("should-not-execute")] = wfv1.NodeStatus{Name: d.taskNodeName("should-not-execute"), Phase: wfv1.NodeSkipped}

	// Tasks should-execute-3 should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-3")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
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
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
	}

	// Task A is still running, A-1 succeeded but A-2 failed
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {Name: d.taskNodeName("A"),
					Phase:    wfv1.NodeRunning,
					Type:     wfv1.NodeTypeTaskGroup,
					Children: []string{d.taskNodeID("A-1"), d.taskNodeID("A-2")},
				},
				d.taskNodeID("A-1"): {Name: d.taskNodeName("A-1"), Phase: wfv1.NodeRunning},
				d.taskNodeID("A-2"): {Name: d.taskNodeName("A-2"), Phase: wfv1.NodeRunning},
			},
		},
	}

	// Task B should not proceed as task A is still running
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Task A succeeded
	d.wf.Status.Nodes[d.taskNodeID("A")] = wfv1.NodeStatus{Name: d.taskNodeName("A"),
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{d.taskNodeID("A-1"), d.taskNodeID("A-2")},
	}

	// Task B should proceed, but not execute as none of the children have succeeded yet
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.False(t, execute)

	// Task A-2 succeeded
	d.wf.Status.Nodes[d.taskNodeID("A-2")] = wfv1.NodeStatus{Name: d.taskNodeName("A-2"), Phase: wfv1.NodeSucceeded}

	// Task B should now proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)

	// Task B succeeds and B-1 fails
	d.wf.Status.Nodes[d.taskNodeID("B")] = wfv1.NodeStatus{Name: d.taskNodeName("B"),
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{d.taskNodeID("B-1"), d.taskNodeID("B-2")},
	}
	d.wf.Status.Nodes[d.taskNodeID("B-1")] = wfv1.NodeStatus{Name: d.taskNodeName("B-1"), Phase: wfv1.NodeFailed}

	// Task C should proceed, but not execute as not all of B's children have failed yet
	execute, proceed, err = d.evaluateDependsLogic(ctx, "C")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.False(t, execute)

	d.wf.Status.Nodes[d.taskNodeID("B-2")] = wfv1.NodeStatus{Name: d.taskNodeName("B-2"), Phase: wfv1.NodeFailed}

	// Task C should now proceed and execute as all of B's children have failed
	execute, proceed, err = d.evaluateDependsLogic(ctx, "C")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
}

func TestEvaluateDependsLogicWhenDaemonFailed(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name:    "B",
			Depends: "A",
		},
	}

	ctx := logging.TestContext(t.Context())
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
	}

	// Task A is running
	daemon := true
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {Name: d.taskNodeName("A"), Phase: wfv1.NodeRunning, Daemoned: &daemon},
			},
		},
	}

	// Task B should proceed and execute
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)

	// Task B running
	d.wf.Status.Nodes[d.taskNodeID("B")] = wfv1.NodeStatus{Name: d.taskNodeName("B"), Phase: wfv1.NodeRunning}

	// Task A failed or error
	d.wf.Status.Nodes[d.taskNodeID("A")] = wfv1.NodeStatus{Name: d.taskNodeName("A"), Phase: wfv1.NodeFailed}

	// Task B should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
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
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
	}

	// Task A is running
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {Name: d.taskNodeName("A"), Phase: wfv1.NodeOmitted},
			},
		},
	}

	// Task B should proceed and execute
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
}

func TestAllEvaluateDependsLogic(t *testing.T) {
	statusMap := map[common.TaskResult]wfv1.NodePhase{
		common.TaskResultSucceeded: wfv1.NodeSucceeded,
		common.TaskResultFailed:    wfv1.NodeFailed,
		common.TaskResultSkipped:   wfv1.NodeSkipped,
	}
	for _, status := range []common.TaskResult{common.TaskResultSucceeded, common.TaskResultFailed, common.TaskResultSkipped} {
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
		d := &dagContext{
			boundaryName: "test",
			tasks:        testTasks,
			wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
			dependencies: make(map[string][]string),
			dependsLogic: make(map[string]string),
			log:          logging.RequireLoggerFromContext(ctx),
		}

		// Task A is running
		d.wf = &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
			Status: wfv1.WorkflowStatus{
				Nodes: map[string]wfv1.NodeStatus{
					d.taskNodeID("same"): {Name: d.taskNodeName("same"), Phase: statusMap[status]},
				},
			},
		}

		execute, proceed, err := d.evaluateDependsLogic(ctx, "Run")
		require.NoError(t, err)
		assert.True(t, proceed)
		assert.True(t, execute)
		execute, proceed, err = d.evaluateDependsLogic(ctx, "NotRun")
		require.NoError(t, err)
		assert.True(t, proceed)
		assert.False(t, execute)
	}
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

func TestRetryStrategyNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/retry-strategy-nodes.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

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
	retryNode, err := woc.wf.GetNodeByName("dag-diamond-88trp")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeRunning, retryNode.Phase)

	retryNode, err = woc.wf.GetNodeByName("dag-diamond-88trp.B.onExit")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.True(t, retryNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, retryNode.Phase)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

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

func TestDagOptionalInputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-optional-input-artifacts.yaml")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	optionalInputArtifactsNode, err := woc.wf.GetNodeByName("dag-optional-inputartifacts.B")
	require.NoError(t, err)
	assert.NotNil(t, optionalInputArtifactsNode)
	assert.Equal(t, wfv1.NodePending, optionalInputArtifactsNode.Phase)
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

func TestEmptyWithParamDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/empty-with-param-dag.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
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

func TestLeafContinueOn(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/leaf-continue-on.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
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
	woc.operate(ctx)
	onExitNode = woc.wf.Status.Nodes.FindByDisplayName("printA.onExit")
	assert.Equal(t, wfv1.NodeSucceeded, onExitNode.Phase)
	assert.True(t, onExitNode.NodeFlag.Hooked)

	// run next DAGTask
	woc.operate(ctx)
	nextDAGTaskNode := woc.wf.Status.Nodes.FindByDisplayName("dependencyTesting")
	require.NotNil(t, nextDAGTaskNode)
	assert.Equal(t, wfv1.NodeRunning, nextDAGTaskNode.Phase)
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

// TestDAGSkippedOutputRef verifies that a DAG task referencing the output of a skipped dependency
// does not cause a requeue loop: the reference is an absent optional (no producer valueFrom.default,
// no consumer input default, no `??` fallback), so the task fails terminally with a clear message
// instead of either requeuing forever or silently receiving an empty string.
// Scenario: stage-a succeeds with output parameters, stage-b is skipped (when evaluates false),
// stage-c depends on both and references outputs from both.
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

// TestDAGSkippedOutputDefault verifies that when a DAG task is skipped and its template declares
// an output parameter with a valueFrom.default, a downstream task referencing that output in its
// INPUT receives the producer's declared default instead of an empty string.
func TestDAGSkippedOutputDefault(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-output-default.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "default-from-producer", in.Value.String())
}

// TestDAGSkippedOutputDefaultAggregate verifies the precedence decision: when a skipped producer
// declares an output valueFrom.default AND the aggregating template's output parameter declares its
// own valueFrom.default, the producer's default wins (it populates scope as a real value, so the
// aggregator's skipped-fallback never fires).
func TestDAGSkippedOutputDefaultAggregate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-output-default-aggregate.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("dag-skipped-output-default-aggregate")
	require.NotNil(t, dagNode)
	require.NotNil(t, dagNode.Outputs)
	require.Len(t, dagNode.Outputs.Parameters, 1)
	assert.Equal(t, "default-from-producer", dagNode.Outputs.Parameters[0].Value.String())
}

// TestDAGSkippedOutputExprDefaultAggregate verifies that a ValueFrom.Expression referencing a skipped
// defaultless output WITHOUT handling the absent (nil) optional mirrors the inline {{= ...}} semantics:
// the expression fails to resolve, and the output parameter's own valueFrom.default applies via the
// error fallback (instead of silently emitting "").
func TestDAGSkippedOutputExprDefaultAggregate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-output-expr-default-aggregate.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("dag-skipped-output-expr-default-aggregate")
	require.NotNil(t, dagNode)
	require.NotNil(t, dagNode.Outputs)
	require.Len(t, dagNode.Outputs.Parameters, 1)
	assert.Equal(t, "default-from-aggregator", dagNode.Outputs.Parameters[0].Value.String())
}

// TestDAGSkippedRefDynamicTemplateName verifies that a task whose templateRef is itself templated
// ("{{item.*}}", resolved only at expansion) is still rescued by the consumed template's input
// default when an argument references a skipped defaultless output: the argument is marked with the
// absent-optional sentinel before substitution and ProcessArgs interprets it at consumption time,
// when the dynamic templateRef has been resolved.
func TestDAGSkippedRefDynamicTemplateName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-ref-dynamic-template-name.yaml"), wfv1.MustUnmarshalWorkflowTemplate(skippedRefConsumeWorkflowTemplate))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-ref-dynamic-template-name.yaml"), controller)
	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	var consumer *wfv1.NodeStatus
	for _, node := range woc.wf.Status.Nodes {
		assert.NotEqual(t, wfv1.NodeError, node.Phase, "node %q should not error: %s", node.DisplayName, node.Message)
		if strings.HasPrefix(node.DisplayName, "consumer(") {
			n := node
			consumer = &n
		}
	}
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "FALLBACK", in.Value.String())
}

// TestDAGSkippedInputDefaultUsed verifies that when a producer is skipped and its output declares NO
// valueFrom.default, a consumer referencing that output in its input falls back to the consumer's OWN
// input default rather than receiving the empty skipped-marker.
//
// The fixture mirrors default-demo.yaml: the producer is skipped and its output parameter declares
// NO valueFrom.default; the consumer references that output in its input, and the consumer's input
// declares its own default. This is the case where the skipped-marker "" is written into scope,
// substituted into the consumer's argument, and then clobbers the input default.
func TestDAGSkippedInputDefaultUsed(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-input-default-suppressed.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	// The skipped reference must NOT clobber the consumer's own input default.
	assert.Equal(t, "FALLBACK-FROM-INPUT", in.Value.String(),
		"a skipped output reference should fall back to the consumer's input default")
}

// TestDAGWhenExprSkipEval verifies that expression templates in a DAG task's arguments are not
// evaluated when the task's "when" clause evaluates to false.
// Scenario: workflow parameter "data" is empty. Task B has when: "{{= workflow.parameters.data != ” }}"
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

// TestDAGSkippedInlineExpressionFallback verifies that an inline {{= ... ?? ...}} expression in a
// task argument sees a skipped/omitted dependency's defaultless output as nil (absent), so the ??
// fallback applies, instead of the empty-string flattening that previously made ?? a no-op.
func TestDAGSkippedInlineExpressionFallback(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag/dag-skipped-inline-expression-fallback.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "inline-fallback", in.Value.String())
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
