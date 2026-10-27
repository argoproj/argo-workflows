package maindriver

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Task is one execution of a template's main command.
type Task struct {
	// NodeID identifies the task. The sink uses it as the key for results.
	NodeID string
	// Template is the task's template. Required.
	Template *wfv1.Template
	// Command is the executable followed by its arguments.
	Command []string
	// Env is the environment of the command. If Env is nil, the command
	// inherits the environment of the process (os/exec).
	Env []string
	// IncludeScriptOutput captures stdout so it can become the script result.
	IncludeScriptOutput bool
	// WorkDir holds the captured stdout and combined logs. Run requires it
	// when it captures logs.
	WorkDir string
	// OnStart, if set, receives the command's pid once started.
	OnStart func(pid int)
}

// OutputKind says what an Output file is.
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
	// Next returns the next task. Check err first. ok=false means that the
	// source has no more tasks.
	Next(ctx context.Context) (task Task, ok bool, err error)
}

// MainDriver runs a task's main command exactly once. Retries, liveness,
// signal delivery and every other concern of the pod layout belong to the
// caller.
type MainDriver interface {
	// Run blocks until the command exits and the sink receives its outputs.
	// exitCode is the exit code of the command, whatever err is. A handover
	// error replaces the exit error and stops further handover.
	Run(ctx context.Context, task Task, sink ResultSink) (exitCode int, err error)
}
