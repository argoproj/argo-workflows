package controller

import (
	"slices"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// validTransitions governs the controller-driven node phase changes made
// through markNodePhase, which refuses (and logs at Error) any transition not
// listed here. It does not cover every phase write:
//
//   - markNodePhase itself lets a node that is not yet fulfilled (its task
//     result has not arrived) change to Error, even from a phase whose row
//     does not list Error, such as Succeeded.
//   - Pod reconciliation (podReconciliation, from assessNodeStatus) records
//     a pod's own result directly with Nodes.Set, so a finishing pod can
//     replace a phase the controller recorded.
//   - clearStaleDaemonedRetries writes a Retry node directly with Nodes.Set
//     (it clears the Daemoned flag; the phase is unchanged).
//
// State legend:
//
//	""        = uninitialized (before the first call to initializeNode)
//	Pending   = waiting to be scheduled, or holding a synchronization lock
//	Running   = pod is executing (or a DAG/Steps composite node is orchestrating)
//	Succeeded = terminal: completed successfully
//	Failed    = terminal: non-zero exit code, or daemon pod exited unexpectedly
//	Error     = terminal: controller-level error unrelated to process exit
//	Skipped   = terminal: when-clause evaluated to false
//	Omitted   = terminal: DAG depends condition was not met, a StepGroup whose every step was omitted, or a TaskGroup a hook error stopped before any item started
var validTransitions = map[wfv1.NodePhase][]wfv1.NodePhase{
	"": {
		wfv1.NodePending,
		wfv1.NodeRunning, // DAG/Steps nodes initialize directly to Running
		wfv1.NodeSkipped,
		wfv1.NodeError,
		// Composite (DAG/Steps) nodes can be loaded from persisted status with
		// an empty phase and then re-assessed against already-fulfilled
		// children, producing a direct empty → terminal transition.
		wfv1.NodeSucceeded,
		wfv1.NodeFailed,
		wfv1.NodeOmitted,
	},
	wfv1.NodePending: {
		wfv1.NodePending,   // retry loop / pod auto-restart
		wfv1.NodeRunning,   // pod scheduled and started
		wfv1.NodeSucceeded, // immediate success (e.g. memoization cache hit)
		wfv1.NodeFailed,    // pod failed before reaching Running (e.g. image pull error)
		wfv1.NodeError,     // controller-level error
		wfv1.NodeSkipped,   // when-clause re-evaluated after pending
		wfv1.NodeOmitted,   // DAG depends condition unmet after scheduling
	},
	wfv1.NodeRunning: {
		wfv1.NodeRunning, // daemon pod heartbeat / daemoned state update
		wfv1.NodePending, // container set: container Running → Waiting (re-pending)
		wfv1.NodeSucceeded,
		wfv1.NodeFailed,
		wfv1.NodeError,
		wfv1.NodeOmitted, // StepGroup whose every step was omitted after an earlier group failed, or TaskGroup a hook error stopped before any item started
	},
	// Terminal states have no valid outbound transitions.
	// The existing `if node.Phase != phase` guard in markNodePhase makes
	// same-phase updates no-ops, so they are not listed here.
	wfv1.NodeSucceeded: {},
	wfv1.NodeFailed:    {},
	wfv1.NodeError:     {},
	wfv1.NodeSkipped:   {},
	wfv1.NodeOmitted:   {},
}

// isValidPhaseTransition reports whether transitioning from → to is a documented valid transition.
// Same-phase transitions are always valid (idempotent updates).
func isValidPhaseTransition(from, to wfv1.NodePhase) bool {
	if from == to {
		return true
	}
	return slices.Contains(validTransitions[from], to)
}
