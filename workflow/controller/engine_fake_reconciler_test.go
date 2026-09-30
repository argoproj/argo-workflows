package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
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
	tmpl := woc.execWf.GetTemplateByName("main")
	require.NotNil(t, tmpl, "test workflows must define a 'main' template")

	tmplCtx, err := woc.createTemplateContext(ctx, wfv1.ResourceScopeLocal, "")
	require.NoError(t, err)

	engine := NewEngine(woc, mainNode.Name, tmplCtx, tmpl, mainNode.ID, false)
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

// expandCountingTask counts how often an expanded task is expanded.
type expandCountingTask struct {
	dag.Task
	expands *int
}

func (c expandCountingTask) Expand(ctx context.Context, scope map[string]string, substitutor dag.Substitutor) ([]dag.Task, error) {
	*c.expands++
	return c.Task.Expand(ctx, scope, substitutor)
}

// Resolve keeps the resolved task counted: the Engine expands the task
// resolveTask returns.
func (c expandCountingTask) Resolve(resolve func(wfv1.DAGTask) (wfv1.DAGTask, error)) (dag.Task, error) {
	resolved, err := c.Task.Resolve(resolve)
	if err != nil {
		return nil, err
	}
	return expandCountingTask{Task: resolved, expands: c.expands}, nil
}

// TestRegressionR4_C19_ExpandedTaskExpandsOncePerDispatch: one visit of a
// fan-out whose three items are Pending expands the task once
// and hands each item to the reconciler once, in item order. Each
// Pending item used to be dispatched on its own, re-expanding the whole
// task every time: O(n²) per reconcile for an n-item fan-out.
func TestRegressionR4_C19_ExpandedTaskExpandsOncePerDispatch(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	engine, fake, woc, tasks := engineWithFakeReconciler(ctx, t)
	expands := 0
	for i := range tasks {
		tasks[i] = expandCountingTask{Task: tasks[i], expands: &expands}
	}

	fake.calls = nil
	engine.visit(ctx, engine.getTaskByName(tasks, "client"), engine.evaluator.Evaluate(ctx, "client"), true)

	assert.Equal(t, 1, expands, "the fan-out should be expanded once per dispatch")
	assert.Equal(t, []string{
		mainNodeName(woc) + ".client(0:0)",
		mainNodeName(woc) + ".client(1:1)",
		mainNodeName(woc) + ".client(2:2)",
	}, fake.allDesiredTaskNames(), "each item should be reconciled once, in item order")
}

// mainNodeName returns the boundary node name for the test workflow's "main"
// template (i.e. the workflow root, since entrypoint=main).
func mainNodeName(woc *wfOperationCtx) string {
	return woc.wf.Name
}
