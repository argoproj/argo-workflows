package controller

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
)

// fakeReconciler captures DesiredTasks so tests can assert exactly what the
// engine dispatched without going through the K8s reconciliation path.
type fakeReconciler struct {
	calls    [][]DesiredTask
	errOnRun error
}

func (f *fakeReconciler) Reconcile(_ context.Context, desired []DesiredTask) error {
	f.calls = append(f.calls, desired)
	return f.errOnRun
}

// allDesiredTaskNames returns every DesiredTask name that was reconciled, in
// the order it was passed in. Useful for assertions.
func (f *fakeReconciler) allDesiredTaskNames() []string {
	var names []string
	for _, batch := range f.calls {
		for _, dt := range batch {
			names = append(names, dt.TaskName)
		}
	}
	return names
}

// engineWithFakeReconciler runs one real operate cycle to set up the
// TaskGroup and child nodes, then returns a fresh engine pointing at the
// resulting wf state with a fake reconciler swapped in. Subsequent
// engine.* calls don't touch K8s — they just exercise the dispatch
// machinery so we can inspect what would be reconciled.
func engineWithFakeReconciler(ctx context.Context, t *testing.T) (*Engine, *fakeReconciler, *wfOperationCtx, []dag.Task) {
	t.Helper()
	wf := wfv1.MustUnmarshalWorkflow(dagWithSequenceForIntegration)
	cancel, controller := newController(ctx, wf)
	t.Cleanup(cancel)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // primes Status.Nodes (TaskGroup parent + children)

	// Find the DAG/Steps boundary node and template
	mainNode, err := woc.wf.GetNodeByName(wf.Name)
	require.NoError(t, err, "workflow root node must exist after operate")

	// Locate the template by name; tests use entrypoint=main.
	var tmpl *wfv1.Template
	for i := range woc.execWf.Spec.Templates {
		if woc.execWf.Spec.Templates[i].Name == "main" {
			tmpl = &woc.execWf.Spec.Templates[i]
			break
		}
	}
	require.NotNil(t, tmpl, "test workflows must define a 'main' template")

	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	engine := NewEngine(woc, mainNode.Name, tmplCtx, tmpl, mainNode, mainNode.ID, false)
	fake := &fakeReconciler{}
	engine.reconciler = fake

	// Build the static task list from the DAG template.
	var tasks []dag.Task
	if tmpl.DAG != nil {
		for i := range tmpl.DAG.Tasks {
			tasks = append(tasks, &dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[i]})
		}
	}
	// Force the evaluator to be re-built against the (now-populated) wf state.
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(woc.wf, tasks, tmpl, mainNode.ID, mainNode.Name)
	return engine, fake, woc, tasks
}

// markChildPhase rewrites the in-memory phase of an existing child node by
// display name. Returns the node ID that was updated.
func markChildPhase(t *testing.T, woc *wfOperationCtx, displayName string, phase wfv1.NodePhase) string {
	t.Helper()
	for id, node := range woc.wf.Status.Nodes {
		if node.DisplayName == displayName {
			node.Phase = phase
			woc.wf.Status.Nodes[id] = node
			return id
		}
	}
	t.Fatalf("no node with display name %q", displayName)
	return ""
}

const dagWithSequenceForIntegration = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-seq-integ
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: client
        template: client
        withSequence:
          count: "3"
  - name: client
    container:
      image: alpine:3.23
      command: [echo, hi]
`

// --- Integration: evaluator output → engine dispatch ---

func TestIntegration_EvaluatorPerChild_AllPending_DispatchesEachChild(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := engineWithFakeReconciler(ctx, t)

	// After the initial operate, all three children exist as Pending.
	results := engine.evaluator.EvaluateAll(ctx)

	// Every Pending child must produce an ActionExecute result with the parent linkage.
	for _, n := range []string{"client(0:0)", "client(1:1)", "client(2:2)"} {
		r, ok := results[n]
		require.True(t, ok, "evaluator must emit a result for %q", n)
		assert.Equal(t, dag.ActionExecute, r.Action, "%q action", n)
		assert.True(t, r.ShouldRun, "%q ShouldRun", n)
		assert.Equal(t, "client", r.ParentTaskName, "%q ParentTaskName", n)
	}

	// Reset the fake reconciler so we only count dispatches caused by converge.
	fake.calls = nil
	_, err := engine.converge(ctx, tasks, results)
	require.NoError(t, err)

	// Each per-child result should have triggered one dispatch through
	// dispatchTaskGroupChild → reconcileExpanded → fakeReconciler.Reconcile.
	gotNames := fake.allDesiredTaskNames()
	sort.Strings(gotNames)
	want := []string{
		mainNodeName(woc) + ".client(0:0)",
		mainNodeName(woc) + ".client(1:1)",
		mainNodeName(woc) + ".client(2:2)",
	}
	sort.Strings(want)
	assert.Equal(t, want, gotNames,
		"converge must dispatch each Pending child through the reconciler")
}

func TestIntegration_EvaluatorPerChild_OnlyPendingDispatched(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := engineWithFakeReconciler(ctx, t)

	// Pretend client(0:0) has succeeded and client(2:2) is now Running.
	markChildPhase(t, woc, "client(0:0)", wfv1.NodeSucceeded)
	markChildPhase(t, woc, "client(2:2)", wfv1.NodeRunning)
	// client(1:1) stays Pending.

	results := engine.evaluator.EvaluateAll(ctx)
	assert.Equal(t, dag.ActionNone, results["client(0:0)"].Action)
	assert.Equal(t, dag.ActionExecute, results["client(1:1)"].Action)
	assert.Equal(t, dag.ActionNone, results["client(2:2)"].Action)

	fake.calls = nil
	_, err := engine.converge(ctx, tasks, results)
	require.NoError(t, err)

	gotNames := fake.allDesiredTaskNames()
	assert.Equal(t, []string{mainNodeName(woc) + ".client(1:1)"}, gotNames,
		"only the Pending child must be dispatched")
}

func TestIntegration_EvaluatorPerChild_AllSucceeded_NoDispatches(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := engineWithFakeReconciler(ctx, t)

	for _, name := range []string{"client(0:0)", "client(1:1)", "client(2:2)"} {
		markChildPhase(t, woc, name, wfv1.NodeSucceeded)
	}

	results := engine.evaluator.EvaluateAll(ctx)
	for _, name := range []string{"client(0:0)", "client(1:1)", "client(2:2)"} {
		assert.Equal(t, dag.ActionNone, results[name].Action, "%q", name)
	}

	fake.calls = nil
	_, err := engine.converge(ctx, tasks, results)
	require.NoError(t, err)
	assert.Empty(t, fake.calls,
		"all-Succeeded children should not produce per-child dispatches")
}

func TestIntegration_DispatchTaskGroupChild_MissingParent_NoOp(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, _, tasks := engineWithFakeReconciler(ctx, t)

	fake.calls = nil
	err := engine.dispatchTaskGroupChild(ctx, tasks, "no-such-parent", "client(0:0)")
	require.NoError(t, err, "missing parent should be a silent no-op, not an error")
	assert.Empty(t, fake.calls, "no reconciles for missing parent")
}

func TestIntegration_DispatchTaskGroupChild_ChildNotInExpansion_NoOp(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, _, tasks := engineWithFakeReconciler(ctx, t)

	fake.calls = nil
	err := engine.dispatchTaskGroupChild(ctx, tasks, "client", "client(99:99)")
	require.NoError(t, err, "non-existent child should be a silent no-op")
	assert.Empty(t, fake.calls, "no reconciles for an item that isn't in the expansion")
}

func TestIntegration_DispatchTaskGroupChild_ChildAlreadyFulfilled_NoReconcile(t *testing.T) {
	// createDesiredTask returns nil for fulfilled children. dispatchTaskGroupChild
	// should detect the empty desired slice and skip the Reconcile call —
	// otherwise the reconciler would be invoked with an empty batch.
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := engineWithFakeReconciler(ctx, t)

	markChildPhase(t, woc, "client(0:0)", wfv1.NodeSucceeded)

	fake.calls = nil
	err := engine.dispatchTaskGroupChild(ctx, tasks, "client", "client(0:0)")
	require.NoError(t, err)
	assert.Empty(t, fake.calls,
		"fulfilled child should not produce any reconcile calls")
}

func TestIntegration_DispatchTaskGroupChild_ParentLinkageNotSet(t *testing.T) {
	// dispatchTaskGroupChild must NOT stamp ParentNodeNames on the desired
	// task — the TaskGroup parent already exists from initial expansion and
	// the child is already linked. Re-stamping would create duplicate edges.
	ctx := logging.TestContext(t.Context())
	engine, fake, _, tasks := engineWithFakeReconciler(ctx, t)

	fake.calls = nil
	err := engine.dispatchTaskGroupChild(ctx, tasks, "client", "client(0:0)")
	require.NoError(t, err)
	require.Len(t, fake.calls, 1, "expected exactly one reconcile batch")
	require.Len(t, fake.calls[0], 1, "expected exactly one DesiredTask")
	assert.Empty(t, fake.calls[0][0].ParentNodeNames,
		"per-child dispatch must not duplicate parent linkage that already exists")
}

func TestIntegration_Converge_RouteByParentTaskName_StaticVsChild(t *testing.T) {
	// converge's routing decision must hinge on ParentTaskName, not on the
	// coincidence of getTaskByName returning nil. Feed it both shapes and
	// confirm each goes the right way.
	ctx := logging.TestContext(t.Context())
	engine, fake, _, tasks := engineWithFakeReconciler(ctx, t)

	results := map[string]dag.EvaluationResult{
		// Per-child result: routes to dispatchTaskGroupChild.
		"client(0:0)": {
			TaskName:       "client(0:0)",
			ParentTaskName: "client",
			Action:         dag.ActionExecute,
			ShouldRun:      true,
			CurrentPhase:   wfv1.NodePending,
		},
		// Static-task result with no per-child marker: routes to executeTask.
		// (We don't actually want executeTask to do work here — but it shouldn't
		// route via dispatchTaskGroupChild either. We just verify the static
		// task result *isn't* dispatched as a child.)
		"client": {
			TaskName:     "client",
			Action:       dag.ActionNone,
			CurrentPhase: wfv1.NodeRunning,
		},
	}

	fake.calls = nil
	_, err := engine.converge(ctx, tasks, results)
	require.NoError(t, err)

	// Exactly one reconcile, for the per-child result.
	require.Len(t, fake.calls, 1)
	require.Len(t, fake.calls[0], 1)
	assert.Contains(t, fake.calls[0][0].TaskName, "client(0:0)")
}

// mainNodeName returns the boundary node name for the test workflow's "main"
// template (i.e. the workflow root, since entrypoint=main).
func mainNodeName(woc *wfOperationCtx) string {
	return woc.wf.Name
}
