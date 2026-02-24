package commands

import (
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
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

			go func() {
				for s := range signals {
					// Artifact sidecars ignore SIGTERM, and only honor that signal via
					// file based termination from the wait container. We hang around
					// to assist wait container even when kubernetes is SIGTERMing us.
					if osspecific.CanIgnoreSignal(s) || s == syscall.SIGTERM {
						logger.WithField("signal", s).Debug(ctx, "ignore signal")
						continue
					}

					logger.WithField("signal", s).Debug(ctx, "forwarding signal")
					_ = osspecific.Kill(command.Process.Pid, s.(syscall.Signal))
				}
			}()
			// Use background context for signal handler so it responds to wait
			// even after the plugin server process exits
			signalCtx := logger.NewBackgroundContext()
			startFileSignalHandler(signalCtx, command.Process.Pid)

			cmdErr := osspecific.Wait(command.Process)
			if cmdErr == nil {
				exitCode = 0
			} else if exitError, ok := cmdErr.(errors.Exited); ok {
				if exitError.ExitCode() >= 0 {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 137 // SIGTERM
				}
			}

			logger.Info(ctx, "artifact plugin sidecar command exited")
			return nil
		},
	}
	return &command
}
