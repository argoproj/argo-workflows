package commands

import (
	"os"
	"os/signal"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

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

			// Artifact sidecars ignore SIGTERM (ignoreTerm=true), and only honor
			// that signal via file-based termination from the aux container. We hang
			// around to assist the aux container even when kubernetes is SIGTERMing us.
			forwardSignals(ctx, signals, command.Process.Pid, true)
			// Use background context for signal handler so it responds to wait
			// even after the plugin server process exits
			signalCtx := logger.NewBackgroundContext()
			startFileSignalHandler(signalCtx, command.Process.Pid)

			cmdErr := osspecific.Wait(command.Process)
			exitCode = exitCodeFromErr(cmdErr, exitCode)

			logger.Info(ctx, "artifact plugin sidecar command exited")
			return nil
		},
	}
	return &command
}
