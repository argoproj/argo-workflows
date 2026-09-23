package dag

import (
	"fmt"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Action is the evaluator's decision for a retry or task-group node. The
// Engine dispatches the task through executeTask for Execute, Succeed and
// Fail alike; the distinction records what the evaluator concluded, and is
// what the evaluator's tests check.
type Action int

const (
	// ActionNone means there is nothing to dispatch this pass: the task is
	// waiting, already recorded, or backing off (see RequeueAfter).
	ActionNone Action = iota
	// ActionExecute means the task should be dispatched: its node created, or
	// the next retry attempt scheduled.
	ActionExecute
	// ActionSucceed means the evaluator judges the node Succeeded. The Engine
	// does not mark it; dispatching it lets the operator's retry handling or
	// the TaskGroup assessment record the outcome, and post-execution handling
	// (lock release, metrics) run.
	ActionSucceed
	// ActionFail means the evaluator judges the node Failed or Errored
	// (CurrentPhase says which). As for ActionSucceed, the Engine dispatches it
	// and the operator records the outcome.
	ActionFail
)

func (a Action) String() string {
	switch a {
	case ActionNone:
		return "None"
	case ActionExecute:
		return "Execute"
	case ActionSucceed:
		return "Succeed"
	case ActionFail:
		return "Fail"
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
	// Suspended is set while the task waits on dependencies whose outcome
	// could still change its result, and WaitingOn names them. Both are
	// diagnostic: the Engine logs them and does not act on them.
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
