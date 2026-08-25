package variables_test

import (
	"slices"
	"testing"

	v "github.com/argoproj/argo-workflows/v4/util/variables"
)

// samePhaseSet reports whether a and b contain the same phases, ignoring order
// and rejecting duplicates.
func samePhaseSet(a, b []v.LifecyclePhase) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[v.LifecyclePhase]bool, len(a))
	for _, p := range a {
		if seen[p] {
			return false
		}
		seen[p] = true
	}
	for _, p := range b {
		if !seen[p] {
			return false
		}
	}
	return true
}

// TestReachablePhases pins the hand-maintained per-kind reachability matrix so
// that changes to it are deliberate rather than silent. The doc generator gates
// matrix marks on these sets, so a regression here would quietly mislabel the
// published variable catalog.
func TestReachablePhases(t *testing.T) {
	bodyLeaf := []v.LifecyclePhase{
		v.PhWorkflowStart, v.PhPreDispatch, v.PhDuringExecute,
		v.PhInsideLoop, v.PhInsideRetry, v.PhMetricEmission,
	}
	cases := []struct {
		kind v.TemplateKind
		want []v.LifecyclePhase
	}{
		{v.TmplCronWorkflow, []v.LifecyclePhase{v.PhCronEval}},
		{v.TmplExitHandler, []v.LifecyclePhase{
			v.PhWorkflowStart, v.PhPreDispatch, v.PhDuringExecute,
			v.PhAfterNodeInit, v.PhAfterPodStart, v.PhAfterNodeComplete,
			v.PhAfterNodeSucceeded, v.PhAfterLoop, v.PhExitHandler, v.PhMetricEmission,
		}},
		// Pod-producing and agent-driven leaves share one reachable set.
		{v.TmplContainer, bodyLeaf},
		{v.TmplContainerSet, bodyLeaf},
		{v.TmplScript, bodyLeaf},
		{v.TmplResource, bodyLeaf},
		{v.TmplData, bodyLeaf},
		{v.TmplHTTP, bodyLeaf},
		{v.TmplPlugin, bodyLeaf},
		{v.TmplSteps, []v.LifecyclePhase{
			v.PhWorkflowStart, v.PhPreDispatch, v.PhDuringExecute,
			v.PhInsideLoop, v.PhInsideRetry,
			v.PhAfterNodeInit, v.PhAfterPodStart, v.PhAfterNodeComplete,
			v.PhAfterNodeSucceeded, v.PhAfterLoop, v.PhMetricEmission,
		}},
		{v.TmplDAG, []v.LifecyclePhase{
			v.PhWorkflowStart, v.PhPreDispatch, v.PhDuringExecute,
			v.PhInsideLoop, v.PhInsideRetry,
			v.PhAfterNodeInit, v.PhAfterPodStart, v.PhAfterNodeComplete,
			v.PhAfterNodeSucceeded, v.PhAfterLoop, v.PhMetricEmission,
		}},
		{v.TmplSuspend, []v.LifecyclePhase{
			v.PhWorkflowStart, v.PhPreDispatch, v.PhDuringExecute,
			v.PhInsideLoop, v.PhMetricEmission,
		}},
	}
	for _, tc := range cases {
		got := v.ReachablePhases(tc.kind)
		if !samePhaseSet(got, tc.want) {
			t.Errorf("ReachablePhases(%s) = %v, want %v", tc.kind, got, tc.want)
		}
	}
}

// TestReachablePhasesInvariants asserts the semantic properties the comments on
// ReachablePhases promise, independently of the exact sets above.
func TestReachablePhasesInvariants(t *testing.T) {
	// Suspend has no realistic failure path, so retry vars must never bind.
	if slices.Contains(v.ReachablePhases(v.TmplSuspend), v.PhInsideRetry) {
		t.Error("Suspend should not reach PhInsideRetry")
	}
	// Only cron-workflow evaluation reaches the cron-eval phase.
	for _, k := range []v.TemplateKind{
		v.TmplContainer, v.TmplSteps, v.TmplDAG, v.TmplSuspend, v.TmplExitHandler,
	} {
		if slices.Contains(v.ReachablePhases(k), v.PhCronEval) {
			t.Errorf("%s should not reach PhCronEval", k)
		}
	}
	if got := v.ReachablePhases(v.TmplCronWorkflow); len(got) != 1 || got[0] != v.PhCronEval {
		t.Errorf("ReachablePhases(cron-workflow) = %v, want [cron-eval]", got)
	}
	// Loop/retry-only phases must not leak into the exit-handler column.
	exit := v.ReachablePhases(v.TmplExitHandler)
	if slices.Contains(exit, v.PhInsideLoop) || slices.Contains(exit, v.PhInsideRetry) {
		t.Errorf("exit-handler should not reach loop/retry phases: %v", exit)
	}
	// An unspecified kind has no reachable phases.
	if got := v.ReachablePhases(v.TmplAll); len(got) != 0 {
		t.Errorf("ReachablePhases(any) = %v, want empty", got)
	}
}
