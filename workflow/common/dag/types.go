package dag

import (
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Action describes the side effect the engine must perform for a task.
type Action int

const (
	// ActionNone means no side effect is needed; the engine should do nothing.
	ActionNone Action = iota
	// ActionExecute means the engine should create/schedule a pod (or create the next retry attempt).
	ActionExecute
	// ActionSkip means the engine should mark the task as Skipped.
	ActionSkip
	// ActionOmit means the engine should mark the task as Omitted.
	ActionOmit
	// ActionSucceed means the engine should mark the task as Succeeded.
	ActionSucceed
	// ActionFail means the engine should mark the task as Failed.
	ActionFail
)

// Key uniquely identifies a task in the DAG.
type Key = string

// isTerminalPhase returns true if the phase is a terminal state.
func isTerminalPhase(phase wfv1.NodePhase) bool {
	switch phase {
	case wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeSkipped, wfv1.NodeOmitted, wfv1.NodeError:
		return true
	default:
		return false
	}
}

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
	TaskName     string
	ShouldRun    bool
	Suspended    bool
	WaitingOn    []string
	Skipped      bool
	SkipReason   string
	Error        error
	CurrentPhase wfv1.NodePhase

	// Action is the side effect the engine must perform for this task.
	Action Action
	// ActionReason is a human-readable explanation for the chosen Action.
	ActionReason string
	// RequeueAfter is the retry backoff duration the engine should wait before
	// re-evaluating this task. A zero value means no requeue is needed.
	RequeueAfter time.Duration
	// FulfilledForDeps is true when downstream tasks may proceed even if this
	// task is not yet terminal. For example, a running daemon node is fulfilled
	// for its dependants but the overall retry group is not yet done.
	FulfilledForDeps bool
	// ParentTaskName is set when this result represents an expanded TaskGroup
	// child (e.g. TaskName="client(0:0)", ParentTaskName="client"); empty for
	// regular static-DAG tasks. Lets the engine dispatch per-child without
	// reverse-parsing the name.
	ParentTaskName string
}
