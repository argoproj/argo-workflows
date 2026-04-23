package commands

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

func NewArtifactPluginSidecarCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "artifact-plugin-sidecar",
		Short: "Run an artifact plugin as a sidecar",
		RunE: func(cmd *cobra.Command, args []string) error {
			exitCode := 64
			ctx := cmd.Context()
			logger := logging.RequireLoggerFromContext(ctx)

			osspecific.AllowGrantingAccessToEveryone()

			// Dir permission set to rwxrwxrwx, so that non-root containers can write exitcode and signal files.
			if err := os.MkdirAll(filepath.Join(varRunArgo, "ctr", containerName), 0o777); err != nil {
				return err
			}

			name, args := args[0], args[1:]
			logger.WithFields(logging.Fields{"name": name, "args": args}).Debug(ctx, "starting command")

			command, closer, err := startCommand(ctx, name, args, template)
			if err != nil {
				logger.WithError(err).Error(ctx, "failed to start command")
				return err
			}
			defer closer()
			// setup signal handlers
			signals := make(chan os.Signal, 1)
			defer close(signals)
			signal.Notify(signals)
			defer signal.Reset()

			defer func() {
				err := os.WriteFile(filepath.Join(varRunArgo, "ctr", containerName, "exitcode"), []byte(strconv.Itoa(exitCode)), 0o644)
				if err != nil {
					logger.WithError(err).Error(ctx, "failed to write exit code")
				}
			}()

			// Artifact sidecars ignore SIGTERM (ignoreTerm=true), and only honor
			// that signal via file-based termination from the aux container. We hang
			// around to assist the aux container even when kubernetes is SIGTERMing us.
			forwardSignals(ctx, signals, command.Process.Pid, true)
			// Use background context for signal handler so it responds to wait
			// even after the plugin server process exits
			signalCtx := logger.NewBackgroundContext()
			startFileSignalHandler(signalCtx, command.Process.Pid)

			// Safety net: watch the aux container (supervisor in init-less,
			// wait in legacy) and SIGTERM the plugin process if the aux
			// container exits without sending us a signal file. Without
			// this, an aux-container crash (OOM, panic) during PostMain
			// would orphan the plugin sidecar until pod-level TGP.
			startAuxExitWatcher(signalCtx, command.Process.Pid)

			cmdErr := osspecific.Wait(command.Process)
			exitCode = exitCodeFromErr(cmdErr, exitCode)

			logger.Info(ctx, "artifact plugin sidecar command exited")
			return nil
		},
	}
	return &command
}

// startAuxExitWatcher waits for the aux container (supervisor in init-less
// mode, wait otherwise) to exit. If it does without first signalling us via
// the file-signal mechanism, SIGTERM the plugin process so we don't sit
// alive past the aux container's death and force pod-level TGP cleanup.
// On normal shutdown the aux container's KillArtifactSidecars writes the
// signal file, the plugin exits via startFileSignalHandler, and this
// goroutine's Kill is a no-op against an already-dead pid.
func startAuxExitWatcher(ctx context.Context, pid int) {
	auxName := common.WaitContainerName
	if common.IsInitlessPod() {
		auxName = common.SupervisorContainerName
	}
	logger := logging.RequireLoggerFromContext(ctx)
	go func() {
		em, err := emissary.New()
		if err != nil {
			logger.WithError(err).Warn(ctx, "aux-exit watcher: failed to create emissary; skipping safety net")
			return
		}
		if err := em.Wait(ctx, []string{auxName}); err != nil {
			if ctx.Err() == nil {
				logger.WithError(err).Warn(ctx, "aux-exit watcher: wait returned error")
			}
			return
		}
		logger.WithField("auxContainer", auxName).Info(ctx, "aux container exited; SIGTERM the plugin process as safety net")
		_ = osspecific.Kill(pid, syscall.SIGTERM)
		// Give the plugin a moment to exit on SIGTERM before SIGKILL.
		select {
		case <-ctx.Done():
			return
		case <-time.After(executor.GetTerminationGracePeriodDuration()):
		}
		_ = osspecific.Kill(pid, syscall.SIGKILL)
	}()
}
