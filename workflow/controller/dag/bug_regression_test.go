package dag

// Regression tests for evaluator bugs found while unifying DAG and Steps
// execution in the Engine. Each test asserts the correct behavior and locks
// the fix in.

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
	wf.Status.Nodes.Set(testCtx(t), childID, wfv1.NodeStatus{
		ID:         childID,
		Name:       "dag.A(0)",
		Phase:      wfv1.NodeFailed,
		Type:       wfv1.NodeTypePod,
		StartedAt:  justNow,
		FinishedAt: justNow,
		NodeFlag:   &wfv1.NodeFlag{Retried: true},
	})
	wf.Status.Nodes.Set(testCtx(t), retryNodeID, wfv1.NodeStatus{
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

	result := eval.EvaluateTask(testCtx(t), "A")

	assert.NotEqual(t, ActionExecute, result.Action,
		"evaluator must not schedule a new retry while backoff is active")
	assert.Greater(t, result.RequeueAfter, time.Duration(0),
		"evaluator must set RequeueAfter to the remaining backoff window")
	assert.LessOrEqual(t, result.RequeueAfter, 30*time.Second,
		"RequeueAfter should be bounded by the backoff duration")
}

// TestBug_Depends_NegationWithPendingDep: for "A.Succeeded && !B.Failed"
// with A Succeeded and B still pending, C waits. Evaluating the expression
// with B's fields all false would read !B.Failed as true and run C before B
// could fail; a depends expression is only evaluated once every task it
// references has finished.
func TestBug_Depends_NegationWithPendingDep(t *testing.T) {
	wf := newTestWorkflow("test-wf")

	// A has succeeded.
	addNodeToWorkflow(testCtx(t), wf, "dag.A", wfv1.NodeSucceeded)

	// B has no node yet — it's pending.  C depends on A and on B NOT having
	// failed.  Since B could still fail, C must wait, not fire.

	dagTasks := []wfv1.DAGTask{
		{Name: "A", Template: "t"},
		{Name: "B", Template: "t"},
		{Name: "C", Template: "t", Depends: "A.Succeeded && !B.Failed"},
	}
	tmpl := createDAGTemplate(dagTasks)

	eval := NewDAGEvaluator(wf, tmpl, "", "dag")
	result := eval.EvaluateTask(testCtx(t), "C")

	assert.False(t, result.ShouldRun,
		"C must wait while B is still pending — !B.Failed is not decidable yet")
	assert.True(t, result.Suspended,
		"C should be reported as suspended (waiting on deps)")
	assert.Contains(t, result.WaitingOn, "B",
		"C should report B in its WaitingOn list")
	assert.False(t, result.Skipped,
		"C must not be skipped — B has not failed yet")
}
