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
	// Dir is the working directory of the command. If Dir is empty, the
	// command inherits the working directory of the process.
	Dir string
	// WorkDir holds the captured stdout and combined logs. Run requires it
	// when it captures logs.
	WorkDir string
	// OnStart, if set, receives the command's pid once started.
	OnStart func(pid int)
	// OnExit, if set, is called once the command has exited and been
	// reaped, before its outputs are staged: after it the pid may belong
	// to another process.
	OnExit func()
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
	// the stager's decision.
	Path string
}

// OutputStager receives a task's output files as main produces them.
type OutputStager interface {
	Stage(ctx context.Context, nodeID string, out Output) error
}

// MainDriver runs a task's main command exactly once. Retries, liveness,
// signal delivery and every other concern of the pod layout belong to the
// caller.
type MainDriver interface {
	// Run blocks until the command exits and the stager receives its outputs.
	// exitCode is the exit code of the command, whatever err is. A handover
	// error replaces the exit error and stops further handover.
	Run(ctx context.Context, task Task, stager OutputStager) (exitCode int, err error)
}
