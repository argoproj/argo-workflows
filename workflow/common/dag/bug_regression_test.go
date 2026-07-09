package dag

// Regression tests documenting known bugs from the code review of
// branch dag-refactor-engine-v4.  Each test asserts the CORRECT (post-fix)
// behavior and is expected to fail against the current code.
//
// Tests without t.Skip are active regressions. Tests that still need fault
// injection or document design debt are kept skipped until the required fix
// or injection point exists.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	intstrutil "github.com/argoproj/argo-workflows/v4/util/intstr"
)

// TestBug_RetryBackoff_NotHonored documents Critical #1.
//
// argo.go:evaluateRetryNode decides whether a retry node should schedule a
// new child attempt but never consults wfv1.RetryStrategy.Backoff and never
// sets EvaluationResult.RequeueAfter.  A task whose last child just failed
// and whose backoff duration has not elapsed should return
//   - Action == ActionNone          (don't schedule another attempt yet)
//   - RequeueAfter ~= backoff left  (tell the engine to requeue)
//
// Current behavior: Action == ActionExecute and RequeueAfter == 0.
//
// Note: end-to-end backoff is still enforced by the legacy
// processNodeRetries() in operator.go (called via handleRetries), which
// masks this bug at the workflow level.  The evaluator's contract is
// nevertheless broken and will silently regress if processNodeRetries is
// ever removed.
func TestBug_RetryBackoff_NotHonored(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{}},
	}

	retryNodeID := wf.NodeID("dag.A")
	childID := wf.NodeID("dag.A(0)")

	// Child failed 1s ago — backoff of 30s has not elapsed.
	justNow := metav1.NewTime(time.Now().Add(-1 * time.Second))
	wf.Status.Nodes.Set(testCtx(), childID, wfv1.NodeStatus{
		ID:         childID,
		Name:       "dag.A(0)",
		Phase:      wfv1.NodeFailed,
		Type:       wfv1.NodeTypePod,
		StartedAt:  justNow,
		FinishedAt: justNow,
		NodeFlag:   &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(), retryNodeID, wfv1.NodeStatus{
		ID:       retryNodeID,
		Name:     "dag.A",
		Phase:    wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{childID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	eval.SetRetryStrategy("A", &wfv1.RetryStrategy{
		Limit: intstrutil.ParsePtr("3"),
		Backoff: &wfv1.Backoff{
			Duration: "30s",
		},
	})

	result := eval.EvaluateTask(testCtx(), "A")

	assert.NotEqual(t, ActionExecute, result.Action,
		"evaluator must not schedule a new retry while backoff is active")
	assert.Greater(t, result.RequeueAfter, time.Duration(0),
		"evaluator must set RequeueAfter to the remaining backoff window")
	assert.LessOrEqual(t, result.RequeueAfter, 30*time.Second,
		"RequeueAfter should be bounded by the backoff duration")
}

// TestBug_Depends_NegationWithPendingDep is a regression test for Critical #2.
//
// For a depends expression that negates a non-fulfilled dep (e.g.
// "A.Succeeded && !B.Failed") where A is Succeeded and B is still pending,
// the evaluator must return `waiting` — B could still fail, so the
// expression is not yet decidable.
//
// Before the fix, the evaluator injected pending deps as taskResult{} and
// returned `ready` immediately because `true && !false = true`.  The fix
// enumerates plausible future outcomes for pending deps and only fires the
// task when the expression is true under every outcome.
func TestBug_Depends_NegationWithPendingDep(t *testing.T) {
	wf := newTestWorkflow("test-wf")

	// A has succeeded.
	addNodeToWorkflow(testCtx(), wf, "dag.A", wfv1.NodeSucceeded)

	// B has no node yet — it's pending.  C depends on A and on B NOT having
	// failed.  Since B could still fail, C must wait, not fire.

	dagTasks := []wfv1.DAGTask{
		{Name: "A", Template: "t"},
		{Name: "B", Template: "t"},
		{Name: "C", Template: "t", Depends: "A.Succeeded && !B.Failed"},
	}
	tmpl := createDAGTemplate(dagTasks)

	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	result := eval.EvaluateTask(testCtx(), "C")

	assert.False(t, result.ShouldRun,
		"C must wait while B is still pending — !B.Failed is not decidable yet")
	assert.True(t, result.Suspended,
		"C should be reported as suspended (waiting on deps)")
	assert.Contains(t, result.WaitingOn, "B",
		"C should report B in its WaitingOn list")
	assert.False(t, result.Skipped,
		"C must not be skipped — B has not failed yet")
}

// TestBug_TaskGroupRetry_KeyStripsPrefix verifies that the evaluator's
// retry-strategy lookup for a TaskGroup-expanded retry child uses the
// boundary-stripped task name (the same key the sibling site
// appendTaskGroupChildResults uses), not the full prefixed node name.
//
// Buggy behavior: evaluateTaskGroupNode passes child.Name (e.g.
// "dag.A-retry") to evaluateRetryNode. retryStrategies is registered under
// the stripped key (here "A-retry"), so the lookup returns nil and any
// withItems/withParam task with retryStrategy is forced to ActionFail
// with reason "no retry strategy configured" on the first child failure —
// long before the retry limit is exhausted.
//
// Post-fix behavior: the call site strips the boundary prefix via
// e.store.taskNameFromNodeName(child.Name), the strategy lookup succeeds,
// and the retry is allowed to proceed.
func TestBug_TaskGroupRetry_KeyStripsPrefix(t *testing.T) {
	wf := newTestWorkflow("test")
	ctx := testCtx()

	// Static task "A" is a TaskGroup. Its child "dag.A-retry" is a Retry
	// node that has had exactly one failed attempt so far. With limit=3,
	// more attempts remain and the evaluator should request ActionExecute.
	tgID := wf.NodeID("dag.A")
	retryChildID := wf.NodeID("dag.A-retry")
	attempt0ID := wf.NodeID("dag.A-retry(0)")

	wf.Status.Nodes.Set(ctx, attempt0ID, wfv1.NodeStatus{
		ID: attempt0ID, Name: "dag.A-retry(0)", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod,
	})
	wf.Status.Nodes.Set(ctx, retryChildID, wfv1.NodeStatus{
		ID: retryChildID, Name: "dag.A-retry", Phase: wfv1.NodeRunning,
		Type:     wfv1.NodeTypeRetry,
		Children: []string{attempt0ID},
	})
	wf.Status.Nodes.Set(ctx, tgID, wfv1.NodeStatus{
		ID: tgID, Name: "dag.A", Phase: wfv1.NodeRunning,
		Type: wfv1.NodeTypeTaskGroup, Children: []string{retryChildID},
	})

	tmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{
		Tasks: []wfv1.DAGTask{{Name: "A", Template: "t"}},
	}}
	eval := NewDAGEvaluator(wf, tmpl, "", "dag")

	// Register the retry strategy under the boundary-stripped key — the
	// same key produced by taskNameFromNodeName and used by the sibling
	// call site in appendTaskGroupChildResults. The bug passes the
	// unstripped child.Name, so this lookup fails and we get ActionFail.
	eval.SetRetryStrategy("A-retry", &wfv1.RetryStrategy{Limit: intstrutil.ParsePtr("3")})

	result := eval.EvaluateTask(ctx, "A")

	assert.NotEqual(t, ActionFail, result.Action,
		"TaskGroup with retry child below limit must not fail: got ActionFail with reason %q", result.ActionReason)
	assert.NotEqual(t, wfv1.NodeFailed, result.CurrentPhase,
		"TaskGroup phase must not be Failed while retries remain")
}
