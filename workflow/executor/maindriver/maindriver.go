package maindriver

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Task is one execution of a template's main command.
type Task struct {
	// NodeID identifies the task; results are keyed by it.
	NodeID string
	// Template is the task's template. Required.
	Template *wfv1.Template
	// Command is the executable followed by its arguments.
	Command []string
	// Env is the command's environment; nil inherits the process's (os/exec).
	// theory-debt: nil-inherits is os/exec's rule, not a stated decision.
	Env []string
	// IncludeScriptOutput captures stdout so it can become the script result.
	IncludeScriptOutput bool
	// WorkDir holds captured stdout/combined logs; required when capturing.
	// theory-debt: not in the stated inputs.
	WorkDir string
	// OnStart, if set, receives the command's pid once started.
	// theory-debt: not in the stated inputs.
	OnStart func(pid int)
}

// OutputKind says what an Output file is.
// theory-debt: the stated design says "output files"; the kinds are a choice.
type OutputKind string

const (
	OutputParameter OutputKind = "parameter"
	OutputArtifact  OutputKind = "artifact"
	OutputStdout    OutputKind = "stdout"
	OutputCombined  OutputKind = "combined"
)

// Output is a file produced by a task's main command.
type Output struct {
	Kind OutputKind
	// Path is where the file is now. It may not exist: optional outputs are
	// the sink's decision.
	Path string
}

// ResultSink receives a task's output files.
type ResultSink interface {
	Put(ctx context.Context, nodeID string, out Output) error
}

// TaskSource yields the tasks a worker runs, one at a time.
type TaskSource interface {
	// Next returns the next task; check err first, ok=false means exhausted.
	// theory-debt: the pull-based shape is a choice.
	Next(ctx context.Context) (task Task, ok bool, err error)
}

// MainDriver runs a task's main command exactly once. Retries, liveness,
// signal delivery and every other concern of the pod layout belong to the
// caller.
type MainDriver interface {
	// Run blocks until the command exits and its outputs are handed to sink.
	// exitCode is the command's regardless of err; a handover error replaces
	// the exit error and stops further handover.
	Run(ctx context.Context, task Task, sink ResultSink) (exitCode int, err error)
}
