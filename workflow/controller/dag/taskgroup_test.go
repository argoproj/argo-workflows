package dag

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// taskGroupWorkflow builds a workflow with the TaskGroup node "dag.A" in
// groupPhase and one item node per entry of items, linked under it.
func taskGroupWorkflow(t *testing.T, groupPhase wfv1.NodePhase, items ...wfv1.NodeStatus) *wfv1.Workflow {
	wf := newTestWorkflow("test")
	group := addTaskGroupParent(t, wf, "dag.A")
	group.Phase = groupPhase
	wf.Status.Nodes.Set(testCtx(t), group.ID, *group)
	for _, item := range items {
		addTaskGroupChild(t, wf, group, item.Name, item.Phase, nil)
		if item.Daemoned != nil {
			n := wf.Status.Nodes[wf.NodeID(item.Name)]
			n.Daemoned = item.Daemoned
			wf.Status.Nodes.Set(testCtx(t), n.ID, n)
		}
	}
	return wf
}

func evaluateTaskGroup(t *testing.T, wf *wfv1.Workflow) EvaluationResult {
	tmpl := createDAGTemplate([]wfv1.DAGTask{{Name: "A", Template: "t", WithItems: []wfv1.Item{{Value: []byte(`"x"`)}}}})
	return NewDAGEvaluator(wf, tmpl, "", "dag").Evaluate(testCtx(t), "A")
}

// An unfinished TaskGroup is dispatched on every pass, whatever its items'
// state (none yet, some running, all finished): the Engine completes it.
func TestEvaluateTaskGroupNode_ExecuteWhileUnfulfilled(t *testing.T) {
	for name, items := range map[string][]wfv1.NodeStatus{
		"no items yet": nil,
		"item running": {{Name: "dag.A(0:x)", Phase: wfv1.NodeSucceeded}, {Name: "dag.A(1:y)", Phase: wfv1.NodeRunning}},
		"all finished": {{Name: "dag.A(0:x)", Phase: wfv1.NodeSucceeded}, {Name: "dag.A(1:y)", Phase: wfv1.NodeFailed}},
		"item pending": {{Name: "dag.A(0:x)", Phase: wfv1.NodePending}},
	} {
		t.Run(name, func(t *testing.T) {
			result := evaluateTaskGroup(t, taskGroupWorkflow(t, wfv1.NodeRunning, items...))
			assert.Equal(t, ActionExecute, result.Action)
			assert.True(t, result.ShouldRun)
			assert.False(t, result.FulfilledForDeps)
			assert.Equal(t, wfv1.NodeRunning, result.CurrentPhase)
		})
	}
}

// A completed TaskGroup is fulfilled for its dependants and not dispatched.
// If one of its daemoned items has died since the group Succeeded, it
// reports that failure (worst phase wins, a live daemon counts as
// finished), without changing the node.
func TestEvaluateTaskGroupNode_Fulfilled(t *testing.T) {
	daemoned := true
	for name, tc := range map[string]struct {
		group wfv1.NodePhase
		items []wfv1.NodeStatus
		want  wfv1.NodePhase
	}{
		"succeeded": {wfv1.NodeSucceeded, []wfv1.NodeStatus{{Name: "dag.A(0:x)", Phase: wfv1.NodeSucceeded}}, wfv1.NodeSucceeded},
		"failed":    {wfv1.NodeFailed, []wfv1.NodeStatus{{Name: "dag.A(0:x)", Phase: wfv1.NodeFailed}}, wfv1.NodeFailed},
		"live daemons": {wfv1.NodeSucceeded, []wfv1.NodeStatus{
			{Name: "dag.A(0:x)", Phase: wfv1.NodeRunning, Daemoned: &daemoned},
			{Name: "dag.A(1:y)", Phase: wfv1.NodeRunning, Daemoned: &daemoned},
		}, wfv1.NodeSucceeded},
		"one daemon crashed, one live": {wfv1.NodeSucceeded, []wfv1.NodeStatus{
			{Name: "dag.A(0:x)", Phase: wfv1.NodeFailed, Daemoned: &daemoned},
			{Name: "dag.A(1:y)", Phase: wfv1.NodeRunning, Daemoned: &daemoned},
		}, wfv1.NodeFailed},
	} {
		t.Run(name, func(t *testing.T) {
			result := evaluateTaskGroup(t, taskGroupWorkflow(t, tc.group, tc.items...))
			assert.Equal(t, ActionNone, result.Action)
			assert.False(t, result.ShouldRun)
			assert.True(t, result.FulfilledForDeps)
			assert.Equal(t, tc.want, result.CurrentPhase)
		})
	}
}

func TestTaskGroupPhase(t *testing.T) {
	daemoned := true
	node := func(phase wfv1.NodePhase) *wfv1.NodeStatus { return &wfv1.NodeStatus{Phase: phase} }
	unsynced := false
	for name, tc := range map[string]struct {
		items []*wfv1.NodeStatus
		phase wfv1.NodePhase
		done  bool
	}{
		"all succeeded":               {[]*wfv1.NodeStatus{node(wfv1.NodeSucceeded), node(wfv1.NodeSkipped)}, wfv1.NodeSucceeded, true},
		"missing item":                {[]*wfv1.NodeStatus{node(wfv1.NodeSucceeded), nil}, "", false},
		"running item":                {[]*wfv1.NodeStatus{node(wfv1.NodeFailed), node(wfv1.NodeRunning)}, "", false},
		"pending item":                {[]*wfv1.NodeStatus{node(wfv1.NodePending)}, "", false},
		"result not synced":           {[]*wfv1.NodeStatus{{Phase: wfv1.NodeSucceeded, TaskResultSynced: &unsynced}}, "", false},
		"running daemon finished":     {[]*wfv1.NodeStatus{node(wfv1.NodeSucceeded), {Phase: wfv1.NodeRunning, Daemoned: &daemoned}}, wfv1.NodeSucceeded, true},
		"pending daemon unfinished":   {[]*wfv1.NodeStatus{{Phase: wfv1.NodePending, Daemoned: &daemoned}}, "", false},
		"failed":                      {[]*wfv1.NodeStatus{node(wfv1.NodeSucceeded), node(wfv1.NodeFailed)}, wfv1.NodeFailed, true},
		"error over failed":           {[]*wfv1.NodeStatus{node(wfv1.NodeError), node(wfv1.NodeFailed)}, wfv1.NodeError, true},
		"error over failed, reversed": {[]*wfv1.NodeStatus{node(wfv1.NodeFailed), node(wfv1.NodeError), node(wfv1.NodeSucceeded)}, wfv1.NodeError, true},
	} {
		t.Run(name, func(t *testing.T) {
			phase, done := TaskGroupPhase(tc.items)
			assert.Equal(t, tc.done, done)
			if tc.done {
				assert.Equal(t, tc.phase, phase)
			}
		})
	}
}
