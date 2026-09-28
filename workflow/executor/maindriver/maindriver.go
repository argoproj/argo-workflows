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
	// Env is the environment the command runs with. A nil Env inherits the
	// calling process's environment (os/exec semantics); an empty, non-nil
	// Env runs the command with no environment at all.
	// theory-debt: nil-inherits is os/exec's rule, not a stated decision.
	Env []string
	// IncludeScriptOutput captures stdout so it can become the script result.
	IncludeScriptOutput bool
	// WorkDir is where captured stdout/combined logs are written. Required
	// when logs are captured (IncludeScriptOutput or archived logs).
	// theory-debt: not in the stated inputs; added so log capture has somewhere to write.
	WorkDir string
	// OnStart, if set, is called with the command's pid once it has started.
	// theory-debt: not in the stated inputs; added so the caller can forward signals to the process.
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
	// Next returns the next task. Callers check err first; ok is false with
	// a nil err once the source has no more tasks.
	// theory-debt: the method shape (pull-based Next with an ok flag) is a choice.
	Next(ctx context.Context) (task Task, ok bool, err error)
}

// MainDriver runs a task's main command exactly once. Retries, liveness,
// signal delivery and every other concern of the pod layout belong to the
// caller.
type MainDriver interface {
	// Run blocks until the command exits and its outputs have been handed to
	// sink. exitCode reports the command's exit whatever err is. err is
	// non-nil if the command could not be run, exited non-zero, or an output
	// could not be handed over; in the last case err is the handover error
	// and the remaining outputs are not handed over.
	Run(ctx context.Context, task Task, sink ResultSink) (exitCode int, err error)
}
