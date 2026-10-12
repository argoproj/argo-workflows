package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
	native "github.com/argoproj/argo-workflows/v4/workflow/util"
)

// A reopened native DAG consumes freshly processed arguments, but must not
// rewrite the input evidence of its already completed children.
func TestReopenedDAGRecordsCurrentInputsAndPreservesCompletedChild(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-inputs
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: value
      value: old
  templates:
  - name: main
    inputs:
      parameters:
      - name: value
    dag:
      tasks:
      - name: keep
        template: approve
        arguments:
          parameters:
          - name: value
            value: retained
      - name: wait
        dependencies: [keep]
        template: approve
        arguments:
          parameters:
          - name: value
            value: '{{inputs.parameters.value}}'
  - name: approve
    inputs:
      parameters:
      - name: value
    suspend: {}
`)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	root, err := woc.wf.GetNodeByName(wf.Name)
	require.NoError(t, err)
	require.Equal(t, "old", root.Inputs.Parameters[0].Value.String())
	keep, err := woc.wf.GetNodeByName(wf.Name + ".keep")
	require.NoError(t, err)
	keep.Phase = wfv1.NodeSucceeded
	woc.wf.Status.Nodes[keep.ID] = *keep
	value := wfv1.AnyString("new")
	next := woc.wf.DeepCopy()
	next.Spec.Arguments.Parameters[0].Value = &value
	updated := newWoc(ctx, *next)
	updated.operate(ctx)
	root, err = updated.wf.GetNodeByName(wf.Name)
	require.NoError(t, err)
	require.Equal(t, "new", root.Inputs.Parameters[0].Value.String())
	kept, err := updated.wf.GetNodeByName(wf.Name + ".keep")
	require.NoError(t, err)
	require.Equal(t, keep.Inputs, kept.Inputs)
	require.Equal(t, wfv1.NodeSucceeded, kept.Phase)
	wait, err := updated.wf.GetNodeByName(wf.Name + ".wait")
	require.NoError(t, err)
	require.Equal(t, "new", wait.Inputs.Parameters[0].Value.String())
}

func TestCheckedRetryRetainedDecisionWaitsForUndispatchedAncestor(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: retained-undispatched
 annotations:
  parameters.workflows.example.io/retry: retry
spec:
 entrypoint: main
 templates:
 - name: main
   dag:
    target: decision
    tasks:
    - name: work
      template: approve
    - name: decision
      template: approve
      depends: work.Succeeded
 - name: approve
   suspend: {}
`)
	woc := newWoc(ctx, *wf)
	decision := wfv1.NodeStatus{
		ID: wf.NodeID(wf.Name + ".decision"), Name: wf.Name + ".decision",
		BoundaryID: wf.Name, Type: wfv1.NodeTypeSuspend, Phase: wfv1.NodeSucceeded,
		FinishedAt: metav1.Now(),
	}
	woc.wf.Status.Nodes = wfv1.Nodes{decision.ID: decision}
	tmpl := &wf.Spec.Templates[0]
	tasks := []dag.Task{
		&dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[0]},
		&dag.DAGTask{DAGTask: &tmpl.DAG.Tasks[1]},
	}
	engine := NewEngine(woc, wf.Name, nil, tmpl, wf.Name, false)
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(woc.wf, tasks, tmpl, wf.Name, wf.Name)
	// A throttled/deadline-limited walk may not create the ancestor yet.
	// The retained target cannot complete the new attempt on its own.
	phase, _ := engine.assessDAGPhase(ctx, tasks, false)
	require.Equal(t, wfv1.NodeRunning, phase)
	require.Equal(t, decision, woc.wf.Status.Nodes[decision.ID])
}

func TestCheckedRetrySchedulesAncestorOfRetainedDecision(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[failed], func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: retained-decision
 annotations:
  parameters.workflows.example.io/retry: retry
spec:
 entrypoint: main
 templates:
 - name: main
   dag:
    failFast: false
    tasks:
    - name: work
      template: approve
    - name: later
      template: approve
      dependencies: [work]
 - name: approve
   suspend: {}
`)
			ctx := logging.TestContext(t.Context())
			first := newWoc(ctx, *wf)
			first.operate(ctx)
			work, err := first.wf.GetNodeByName(wf.Name + ".work")
			require.NoError(t, err)
			work.Phase = wfv1.NodeSucceeded
			first.wf.Status.Nodes[work.ID] = *work
			second := newWoc(ctx, *first.wf)
			second.operate(ctx)
			later, err := second.wf.GetNodeByName(wf.Name + ".later")
			require.NoError(t, err)
			later.Phase = wfv1.NodeSucceeded
			later.FinishedAt = metav1.Now()
			second.wf.Status.Nodes[later.ID] = *later
			// The upstream attempt is removed, while the later decision is retained.
			delete(second.wf.Status.Nodes, work.ID)
			root := second.wf.Status.Nodes[wf.Name]
			root.Children = nil
			second.wf.Status.Nodes[wf.Name] = root
			reopened := newWoc(ctx, *second.wf)
			reopened.operate(ctx)
			current, err := reopened.wf.GetNodeByName(wf.Name + ".work")
			require.NoError(t, err)
			require.Equal(t, wfv1.NodeRunning, current.Phase)
			retained, err := reopened.wf.GetNodeByName(wf.Name + ".later")
			require.NoError(t, err)
			require.Equal(t, later.Phase, retained.Phase)
			require.Equal(t, later.FinishedAt, retained.FinishedAt)
			current.Phase = wfv1.NodeSucceeded
			if failed {
				current.Phase = wfv1.NodeFailed
			}
			reopened.wf.Status.Nodes[current.ID] = *current
			final := newWoc(ctx, *reopened.wf)
			final.operate(ctx)
			expected := wfv1.WorkflowSucceeded
			if failed {
				expected = wfv1.WorkflowFailed
			}
			require.Equal(t, expected, final.wf.Status.Phase)
			retained, err = final.wf.GetNodeByName(wf.Name + ".later")
			require.NoError(t, err)
			require.Equal(t, wfv1.NodeSucceeded, retained.Phase)
			require.Equal(t, later.FinishedAt, retained.FinishedAt)
		})
	}
}

func TestCheckedRetryCompletesReopenedDAGWithRetainedChildren(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: retained-dag
 annotations:
  parameters.workflows.example.io/retry: retry
spec:
 entrypoint: main
 templates:
 - name: main
   dag:
    tasks:
    - name: group
      template: group
 - name: group
   dag:
    tasks:
    - name: first
      template: approve
    - name: last
      template: approve
      dependencies: [first]
 - name: approve
   suspend: {}
`)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	retained := map[string]wfv1.NodeStatus{}
	for _, name := range []string{"first", "last"} {
		woc.operate(ctx)
		node, err := woc.wf.GetNodeByName(wf.Name + ".group." + name)
		require.NoError(t, err)
		node.Phase = wfv1.NodeSucceeded
		node.FinishedAt = metav1.Now()
		woc.wf.Status.Nodes[node.ID] = *node
		retained[node.ID] = *node
		woc = newWoc(ctx, *woc.wf)
	}
	// Parameter retry may reopen ordering-only DAG boundaries while keeping
	// every completed action and approval below them. No action is rerun.
	group, err := woc.wf.GetNodeByName(wf.Name + ".group")
	require.NoError(t, err)
	delete(woc.wf.Status.Nodes, group.ID)
	root := woc.wf.Status.Nodes[wf.Name]
	root.Children = nil
	woc.wf.Status.Nodes[wf.Name] = root
	resumed := newWoc(ctx, *woc.wf)
	resumed.operate(ctx)
	require.Equal(t, wfv1.WorkflowSucceeded, resumed.wf.Status.Phase)
	group, err = resumed.wf.GetNodeByName(wf.Name + ".group")
	require.NoError(t, err)
	require.NotEmpty(t, group.Children)
	for id, old := range retained {
		current := resumed.wf.Status.Nodes[id]
		require.Equal(t, old.Phase, current.Phase)
		require.Equal(t, old.FinishedAt, current.FinishedAt)
		require.Equal(t, old.Inputs, current.Inputs)
		require.Equal(t, old.Outputs, current.Outputs)
	}
}

func TestCheckedRetryCompletesReopenedStepsWithRetainedChildren(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: retained-steps
 annotations:
  parameters.workflows.example.io/retry: retry
spec:
 entrypoint: main
 templates:
 - name: main
   steps:
    - - name: group
        template: group
 - name: group
   steps:
    - - name: first
        template: approve
    - - name: last
        template: approve
 - name: approve
   suspend: {}
`)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	retained := map[string]wfv1.NodeStatus{}
	for _, name := range []string{"first", "last"} {
		woc.operate(ctx)
		node, err := woc.wf.GetNodeByName(wf.Name + "[0].group[" + map[string]string{"first": "0", "last": "1"}[name] + "]." + name)
		require.NoError(t, err)
		node.Phase = wfv1.NodeSucceeded
		node.FinishedAt = metav1.Now()
		woc.wf.Status.Nodes[node.ID] = *node
		retained[node.ID] = *node
		woc = newWoc(ctx, *woc.wf)
	}
	// Parameter retry may reopen ordering-only Steps boundaries while keeping
	// every completed action and approval below them. No action is rerun.
	group, err := woc.wf.GetNodeByName(wf.Name + "[0].group")
	require.NoError(t, err)
	delete(woc.wf.Status.Nodes, group.ID)
	root := woc.wf.Status.Nodes[wf.Name]
	root.Children = nil
	woc.wf.Status.Nodes[wf.Name] = root
	resumed := newWoc(ctx, *woc.wf)
	resumed.operate(ctx)
	require.Equal(t, wfv1.WorkflowSucceeded, resumed.wf.Status.Phase)
	group, err = resumed.wf.GetNodeByName(wf.Name + "[0].group")
	require.NoError(t, err)
	require.NotEmpty(t, group.Children)
	firstGroup, err := resumed.wf.GetNodeByName(wf.Name + "[0].group[0]")
	require.NoError(t, err)
	require.Contains(t, group.Children, firstGroup.ID)
	// The next retry walks these edges to find every selected node's parent.
	_, _, err = native.FormulateRetryWorkflow(ctx, resumed.wf, true, "name="+wf.Name+"[0].group[1].last", nil)
	require.NoError(t, err)
	for id, old := range retained {
		current := resumed.wf.Status.Nodes[id]
		require.Equal(t, old.Phase, current.Phase)
		require.Equal(t, old.FinishedAt, current.FinishedAt)
		require.Equal(t, old.Inputs, current.Inputs)
		require.Equal(t, old.Outputs, current.Outputs)
	}
}
