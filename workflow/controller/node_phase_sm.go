package controller

import (
	"slices"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// NodePhaseSM documents and enforces valid node phase transitions.
// The zero value is usable.
type NodePhaseSM struct{}

// validTransitions is the single authoritative table of all valid node phase transitions.
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
//	Omitted   = terminal: DAG depends condition was not met
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

// IsValidTransition reports whether transitioning from → to is a documented valid transition.
// Same-phase transitions are always valid (idempotent updates).
func (NodePhaseSM) IsValidTransition(from, to wfv1.NodePhase) bool {
	if from == to {
		return true
	}
	return slices.Contains(validTransitions[from], to)
}

// nodePhaseSM is the package-level singleton used by markNodePhase.
var nodePhaseSM = NodePhaseSM{}
