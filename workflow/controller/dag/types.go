package dag

import (
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Action is the evaluator's decision for a retry or task-group node, which
// the Engine logs; ShouldRun, set with ActionExecute, is what it acts on.
type Action int

const (
	// ActionNone means there is nothing to dispatch this reconcile: the node
	// is fulfilled.
	ActionNone Action = iota
	// ActionExecute means the unfulfilled node is dispatched, so the
	// operator's retry handling or the TaskGroup's items can progress.
	ActionExecute
)

func (a Action) String() string {
	switch a {
	case ActionNone:
		return "None"
	case ActionExecute:
		return "Execute"
	default:
		return fmt.Sprintf("Action(%d)", int(a))
	}
}

// Key uniquely identifies a task in the DAG.
type Key = string

// readinessResult indicates the readiness state of a task.
type readinessResult int

const (
	// waiting means the task is waiting for dependencies to complete.
	waiting readinessResult = iota
	// ready means the task is ready to execute (all deps satisfied).
	ready
	// omit means the task should be omitted (depends condition can never be met).
	omit
)

// taskResult represents the result state of a dependency task.
// Used as the evaluation scope for depends expressions.
type taskResult struct {
	Succeeded    bool `json:"Succeeded"`
	Failed       bool `json:"Failed"`
	Errored      bool `json:"Errored"`
	Skipped      bool `json:"Skipped"`
	Omitted      bool `json:"Omitted"`
	Daemoned     bool `json:"Daemoned"`
	AnySucceeded bool `json:"AnySucceeded"`
	AllFailed    bool `json:"AllFailed"`
}

// EvaluationResult contains the evaluation result for a single task.
type EvaluationResult struct {
	TaskName string
	// ShouldRun is set when the task's dependencies allow it to run now. For
	// a retry or task-group node it accompanies ActionExecute.
	ShouldRun bool
	// Suspended is set while the task waits on dependencies that have not
	// finished, and WaitingOn names them. Both are diagnostic: the Engine
	// logs them and does not act on them.
	Suspended bool
	WaitingOn []string
	// Skipped is set when the task can never run; the Engine creates its
	// Omitted node with SkipReason.
	Skipped      bool
	SkipReason   string
	Error        error
	CurrentPhase wfv1.NodePhase

	// Action is the evaluator's decision for a retry or task-group node; see
	// the Action constants for what the Engine does with each.
	Action Action
	// ActionReason explains the chosen Action. The Engine logs it at debug
	// level with the task's other diagnostics.
	ActionReason string
	// FulfilledForDeps is set once a retry or task-group node is fulfilled (a
	// running daemon counts), so its dependants may proceed.
	FulfilledForDeps bool
}
