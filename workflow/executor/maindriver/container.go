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

// exitCodeUnknown is reported when the command never produced an exit code,
// matching the emissary's default.
const exitCodeUnknown = 64

// Container runs a task's main command as a child process. Not safe for
// concurrent Run calls in one process (osspecific.Wait reaps any child), and
// ctx cancellation does not stop the command (released before wait).
type Container struct{}

var _ MainDriver = Container{}

func (Container) Run(ctx context.Context, task Task, sink ResultSink) (int, error) {
	if task.Template == nil {
		return exitCodeUnknown, errors.New("task has no template")
	}
	if len(task.Command) == 0 {
		return exitCodeUnknown, errors.New("no command to run")
	}
	name, err := exec.LookPath(task.Command[0])
	if err != nil {
		return exitCodeUnknown, fmt.Errorf("failed to find name in PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, name, task.Command[1:]...)
	cmd.Env = task.Env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	var logs []Output
	if task.IncludeScriptOutput || task.Template.SaveLogsAsArtifact() {
		if task.WorkDir == "" {
			return exitCodeUnknown, errors.New("task has no workdir to capture logs in")
		}
		logging.RequireLoggerFromContext(ctx).Info(ctx, "capturing logs")
		stdoutPath := filepath.Join(task.WorkDir, "stdout")
		combinedPath := filepath.Join(task.WorkDir, "combined")
		// O_APPEND: a caller that retries keeps every attempt's logs.
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
	exitCode := exitCodeFromErr(waitErr)

	// Outputs are handed over whatever the exit code.
	if err := putOutputs(ctx, task, sink, logs); err != nil {
		return exitCode, err
	}
	// Unwrapped: main() reads the exit code with a type assertion.
	return exitCode, waitErr
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

// exitCodeFromErr mirrors the emissary's mapping: 0 on success, the process's
// own code when it exited, 137 when it was signalled.
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
