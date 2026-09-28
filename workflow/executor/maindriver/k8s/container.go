package k8s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// exitCodeUnknown is reported when the command never produced an exit code,
// matching the emissary's default.
const exitCodeUnknown = 64

// Container runs a task's main command as a child process.
type Container struct{}

var _ maindriver.MainDriver = Container{}

func (Container) Run(ctx context.Context, task maindriver.Task, sink maindriver.ResultSink) (int, error) {
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

	var logs []maindriver.Output
	if task.IncludeScriptOutput || task.Template.SaveLogsAsArtifact() {
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
		logs = []maindriver.Output{
			{Kind: maindriver.OutputStdout, Path: stdoutPath},
			{Kind: maindriver.OutputCombined, Path: combinedPath},
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

	// Outputs are handed over whatever the exit code, as the emissary does.
	// theory-debt: the driver pushes outputs for every container; the emissary
	// only saves them for main (the parked containerName gate), so the caller's
	// sink must drop them for other containers.
	if err := putOutputs(ctx, task, sink, logs); err != nil {
		return exitCode, err
	}
	return exitCode, waitErr
}

func putOutputs(ctx context.Context, task maindriver.Task, sink maindriver.ResultSink, logs []maindriver.Output) error {
	outs := logs
	for _, p := range task.Template.Outputs.Parameters {
		if p.ValueFrom != nil && p.ValueFrom.Path != "" {
			outs = append(outs, maindriver.Output{Kind: maindriver.OutputParameter, Path: p.ValueFrom.Path})
		}
	}
	for _, a := range task.Template.Outputs.Artifacts {
		if a.Path != "" {
			outs = append(outs, maindriver.Output{Kind: maindriver.OutputArtifact, Path: a.Path})
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
