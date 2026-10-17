package dag

// Regression tests for evaluator bugs found while unifying DAG and Steps
// execution in the Engine. Each test asserts the correct behavior and locks
// the fix in.

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

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
	result := eval.Evaluate(testCtx(t), "C")

	assert.False(t, result.ShouldRun,
		"C must wait while B is still pending — !B.Failed is not decidable yet")
	assert.True(t, result.Suspended,
		"C should be reported as suspended (waiting on deps)")
	assert.Contains(t, result.WaitingOn, "B",
		"C should report B in its WaitingOn list")
	assert.False(t, result.Skipped,
		"C must not be skipped — B has not failed yet")
}
