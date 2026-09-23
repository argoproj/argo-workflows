package dag

// The depends expressions recommended in docs/upgrading.md and
// docs/enhanced-depends-logic.md must behave as documented under settled
// evaluation. These tests pin those examples.

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// evaluateDocsExpression evaluates task C, which depends on A and B with the
// given expression, after A and B have reached the given phases. An empty
// phase means the task has no node yet (still pending).
func evaluateDocsExpression(t *testing.T, depends string, phaseA, phaseB wfv1.NodePhase) EvaluationResult {
	t.Helper()
	wf := newTestWorkflow("test-wf")
	if phaseA != "" {
		addNodeToWorkflow(testCtx(t), wf, "dag.A", phaseA)
	}
	if phaseB != "" {
		addNodeToWorkflow(testCtx(t), wf, "dag.B", phaseB)
	}
	tmpl := createDAGTemplate([]wfv1.DAGTask{
		{Name: "A", Template: "t"},
		{Name: "B", Template: "t"},
		{Name: "C", Template: "t", Depends: depends},
	})
	return NewDAGEvaluator(wf, tmpl, "", "dag").EvaluateTask(testCtx(t), "C")
}

// Before settled evaluation, "A || B" waited for both tasks to finish and
// then ran if either bare reference was satisfied. The upgrade note's
// migration expression must reproduce exactly that.
func TestDocs_DependsMigrationWaitsForBothTasks(t *testing.T) {
	const migrated = "(A || B) && (A || A.Failed || A.Errored || A.Omitted) && (B || B.Failed || B.Errored || B.Omitted)"

	t.Run("waits while the other task is pending", func(t *testing.T) {
		r := evaluateDocsExpression(t, migrated, wfv1.NodeSucceeded, "")
		assert.False(t, r.ShouldRun)
		assert.False(t, r.Skipped)
	})
	t.Run("runs once both finished and one succeeded", func(t *testing.T) {
		for _, other := range []wfv1.NodePhase{wfv1.NodeFailed, wfv1.NodeError, wfv1.NodeOmitted, wfv1.NodeSkipped} {
			r := evaluateDocsExpression(t, migrated, other, wfv1.NodeSucceeded)
			assert.True(t, r.ShouldRun, "A %s, B Succeeded", other)
		}
	})
	t.Run("omitted once neither can satisfy a bare reference", func(t *testing.T) {
		r := evaluateDocsExpression(t, migrated, wfv1.NodeFailed, wfv1.NodeError)
		assert.False(t, r.ShouldRun)
		assert.True(t, r.Skipped)
	})
	t.Run("previous advice ran when both tasks failed", func(t *testing.T) {
		r := evaluateDocsExpression(t, "(A || A.Failed) && (B || B.Failed)", wfv1.NodeFailed, wfv1.NodeFailed)
		assert.True(t, r.ShouldRun, "the expression the note used to recommend does not preserve the old behaviour")
	})
}
