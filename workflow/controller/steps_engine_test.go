package controller

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
)

// Steps counterparts of the Engine tests that only exercise DAG templates.
// Steps run on the same Engine through StepAdapter tasks (synthetic
// dependencies on the previous group) and StepGroup wiring, so each Engine
// behaviour needs a Steps witness too.

var stepsWithSequenceForEngine = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-engine
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: client
        template: echo
        withSequence:
          count: "3"
  - name: echo
    container:
      image: argoproj/argosay:v2
`

// stepsEngineWithFakeReconciler mirrors engineWithFakeReconciler for a Steps
// template: one operate primes the StepGroup, TaskGroup and child nodes, then
// a fresh Engine over StepAdapter tasks (built as executeSteps builds them) is
// returned with a fake reconciler.
func stepsEngineWithFakeReconciler(ctx context.Context, t *testing.T) (*Engine, *fakeReconciler, *wfOperationCtx, []dag.Task) {
	t.Helper()
	wf := wfv1.MustUnmarshalWorkflow(stepsWithSequenceForEngine)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	mainNode, err := woc.wf.GetNodeByName(wf.Name)
	require.NoError(t, err)
	tmpl := woc.execWf.GetTemplateByName("main")
	require.NotNil(t, tmpl)
	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	var tasks []dag.Task
	var prev []string
	for i, group := range tmpl.Steps {
		var current []string
		for j := range group.Steps {
			task := &StepAdapter{step: &group.Steps[j], dependencies: prev, groupIndex: i}
			tasks = append(tasks, task)
			current = append(current, task.GetName())
		}
		prev = current
	}

	engine := NewEngine(woc, mainNode.Name, tmplCtx, tmpl, mainNode, mainNode.ID, false)
	fake := &fakeReconciler{}
	engine.reconciler = fake
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(woc.wf, tasks, tmpl, mainNode.ID, mainNode.Name)
	return engine, fake, woc, tasks
}

// Per-child dispatch: every Pending expanded child of a Steps step is emitted
// by the evaluator with its parent linkage and dispatched through the reconciler.
func TestStepsEngine_PerChildDispatch(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := stepsEngineWithFakeReconciler(ctx, t)

	results := engine.evaluator.EvaluateAll(ctx)
	for _, n := range []string{"[0].client(0:0)", "[0].client(1:1)", "[0].client(2:2)"} {
		r, ok := results[n]
		require.True(t, ok, "evaluator must emit a result for %q", n)
		assert.Equal(t, dag.ActionExecute, r.Action, "%q action", n)
		assert.Equal(t, "[0].client", r.ParentTaskName, "%q ParentTaskName", n)
	}

	fake.calls = nil
	_, err := engine.converge(ctx, tasks, results)
	require.NoError(t, err)
	got := fake.allDesiredTaskNames()
	sort.Strings(got)
	want := []string{
		woc.wf.Name + "[0].client(0:0)",
		woc.wf.Name + "[0].client(1:1)",
		woc.wf.Name + "[0].client(2:2)",
	}
	assert.Equal(t, want, got)
}

// An evaluator error for a step is recorded as a terminal Error node, as for
// a DAG task (TestConverge_EvaluatorErrorBecomesErrorNode).
func TestStepsEngine_EvaluatorErrorBecomesErrorNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := stepsEngineWithFakeReconciler(ctx, t)

	results := map[string]dag.EvaluationResult{
		"[0].client": {TaskName: "[0].client", CurrentPhase: wfv1.NodeRunning, Error: errors.New("depends expression failed to evaluate")},
	}
	fake.calls = nil
	executed, err := engine.converge(ctx, tasks, results)
	require.NoError(t, err)
	assert.Empty(t, fake.calls)
	assert.True(t, executed["[0].client"])

	node, err := woc.wf.GetNodeByName(engine.taskNodeName("[0].client"))
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Equal(t, "depends expression failed to evaluate", node.Message)
}

var stepsWithItemsRetryBackoff = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-withitems-retry-backoff
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: fail
        arguments:
          parameters:
          - name: msg
            value: "{{item}}"
        withItems: [x]
  - name: fail
    inputs:
      parameters:
      - name: msg
    retryStrategy:
      limit: "2"
      retryPolicy: Always
      backoff:
        duration: "10"
        factor: "2"
    container:
      image: alpine:3.23
      command: [sh, -c, exit 1]
`

// A withItems step whose retry node is waiting out its backoff keeps the
// StepGroup, the Steps boundary and the workflow Running
// (TestWithItemsRetryBackoffKeepsDAGRunning for Steps).
func TestStepsEngine_WithItemsRetryBackoffKeepsRunning(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(stepsWithItemsRetryBackoff)
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)

	makePodsPhase(ctx, woc, apiv1.PodFailed)
	attempt := woc.wf.Status.Nodes.FindByDisplayName("A(0:x)(0)")
	require.NotNil(t, attempt)
	attempt.Phase = wfv1.NodeFailed
	attempt.StartedAt = metav1.Time{Time: time.Now().Add(-2 * time.Second)}
	attempt.FinishedAt = metav1.Time{Time: time.Now()}
	woc.wf.Status.Nodes[attempt.ID] = *attempt

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	retryNode := woc.wf.Status.Nodes.FindByDisplayName("A(0:x)")
	require.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeRunning, retryNode.Phase)
	assert.Equal(t, "Backoff for 10 seconds", retryNode.Message)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1, "no new attempt during the backoff window")

	sgNode, err := woc.wf.GetNodeByName(wf.Name + "[0]")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, sgNode.Phase, "StepGroup must not fail while a retry is backing off")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var stepsWhenClauses = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-when
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: skip-me
        template: echo
        when: "false"
      - name: fan
        template: echo
        when: "{{item}} == 1"
        withItems: [1, 2]
  - name: echo
    container:
      image: argoproj/argosay:v2
`

// when clauses in Steps: a static step with when:false is Skipped, and an
// {{item}}-dependent when on an expanded step is evaluated per child.
func TestStepsEngine_WhenClauses(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(stepsWhenClauses)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	skipped := woc.wf.Status.Nodes.FindByDisplayName("skip-me")
	require.NotNil(t, skipped)
	assert.Equal(t, wfv1.NodeSkipped, skipped.Phase)
	assert.Contains(t, skipped.Message, "when 'false' evaluated false")

	run := woc.wf.Status.Nodes.FindByDisplayName("fan(0:1)")
	require.NotNil(t, run)
	assert.Equal(t, wfv1.NodeTypePod, run.Type)
	assert.Equal(t, wfv1.NodePending, run.Phase)

	skippedChild := woc.wf.Status.Nodes.FindByDisplayName("fan(1:2)")
	require.NotNil(t, skippedChild)
	assert.Equal(t, wfv1.NodeSkipped, skippedChild.Phase)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

var stepsContinueOnFailed = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-continue-on
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: step-a
        template: echo
        continueOn:
          failed: true
    - - name: step-b
        template: echo
  - name: echo
    container:
      image: argoproj/argosay:v2
`

// continueOn.failed on a step: the failed step does not fail its group, so
// the next group is scheduled and the workflow keeps running.
func TestStepsEngine_ContinueOnFailedRollsUp(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(stepsContinueOnFailed)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	stepA := woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, stepA)
	assert.Equal(t, wfv1.NodeFailed, stepA.Phase)

	sg0, err := woc.wf.GetNodeByName(wf.Name + "[0]")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, sg0.Phase, "a continueOn.failed step must not fail its group")

	stepB := woc.wf.Status.Nodes.FindByDisplayName("step-b")
	require.NotNil(t, stepB, "the next group must be scheduled")
	assert.Equal(t, wfv1.NodePending, stepB.Phase)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}
