package maindriver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// exitCodeUnknown is the exit code when the command produced no exit code.
// The emissary uses the same default.
const exitCodeUnknown = 64

// Container runs the main command of a task as a child process. Do not make
// concurrent Run calls in one process, because osspecific.Wait reaps any child.
// Cancellation of ctx does not stop the command, because Run releases the
// command before it waits.
type Container struct{}

var _ MainDriver = Container{}

func (Container) Run(ctx context.Context, task Task, sink ResultSink) (exitCode int, err error) {
	if task.Template == nil {
		return exitCodeUnknown, errors.New("task has no template")
	}
	if len(task.Command) == 0 {
		return exitCodeUnknown, errors.New("no command to run")
	}
	captureLogs := task.IncludeScriptOutput || task.Template.SaveLogsAsArtifact()
	if captureLogs && task.WorkDir == "" {
		return exitCodeUnknown, errors.New("task has no workdir to capture logs in")
	}
	var logs []Output
	// A failed startup can leave outputs of an earlier retry in place. Give
	// their paths to the sink even when this attempt cannot start. The sink
	// decides if each file exists and needs staging.
	defer func() {
		if outputErr := putOutputs(ctx, task, sink, logs); outputErr != nil {
			err = outputErr
		}
	}()
	name, err := exec.LookPath(task.Command[0])
	if err != nil {
		return exitCodeUnknown, fmt.Errorf("failed to find name in PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, name, task.Command[1:]...)
	cmd.Env = task.Env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if captureLogs {
		logging.RequireLoggerFromContext(ctx).Info(ctx, "capturing logs")
		stdoutPath := filepath.Join(task.WorkDir, "stdout")
		combinedPath := filepath.Join(task.WorkDir, "combined")
		// O_APPEND keeps the logs of every attempt when the caller retries.
		var stdoutf, combinedf *os.File
		stdoutf, err = os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return exitCodeUnknown, fmt.Errorf("failed to open stdout: %w", err)
		}
		defer func() { _ = stdoutf.Close() }()
		combinedf, err = os.OpenFile(combinedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return exitCodeUnknown, fmt.Errorf("failed to open combined: %w", err)
		}
		defer func() { _ = combinedf.Close() }()
		cmd.Stdout = io.MultiWriter(os.Stdout, stdoutf, combinedf)
		cmd.Stderr = io.MultiWriter(os.Stderr, combinedf)
		logs = []Output{
			{Kind: OutputStdout, Path: stdoutPath},
			{Kind: OutputCombined, Path: combinedPath},
		}
	}

	closer, err := osspecific.StartCommand(ctx, cmd)
	if err != nil {
		return exitCodeUnknown, fmt.Errorf("failed to start command: %w", err)
	}
	if task.OnStart != nil {
		task.OnStart(cmd.Process.Pid)
	}
	waitErr := osspecific.Wait(cmd.Process)
	closer()
	// Do not wrap the error: main() reads the exit code with a type assertion.
	return exitCodeFromErr(waitErr), waitErr
}

func putOutputs(ctx context.Context, task Task, sink ResultSink, logs []Output) error {
	outs := logs
	for _, p := range task.Template.Outputs.Parameters {
		if p.ValueFrom != nil && p.ValueFrom.Path != "" {
			outs = append(outs, Output{Kind: OutputParameter, Path: p.ValueFrom.Path})
		}
	}
	for _, a := range task.Template.Outputs.Artifacts {
		if a.Path != "" {
			outs = append(outs, Output{Kind: OutputArtifact, Path: a.Path})
		}
	}
	for _, out := range outs {
		if err := sink.Put(ctx, task.NodeID, out); err != nil {
			return fmt.Errorf("failed to hand over %s %s: %w", out.Kind, out.Path, err)
		}
	}
	return nil
}

// exitCodeFromErr uses the mapping of the emissary. It returns 0 on success,
// the exit code of the process when it exited, and 137 when a signal stopped it.
func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	var exited argoerrors.Exited
	if errors.As(err, &exited) {
		if exited.ExitCode() >= 0 {
			return exited.ExitCode()
		}
		return 137
	}
	return exitCodeUnknown
}
