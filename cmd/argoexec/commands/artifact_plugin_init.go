package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func NewArtifactPluginInitCommand() *cobra.Command {
	var artifactPlugin string
	command := cobra.Command{
		Use:   "artifact-plugin-init",
		Short: "Load artifacts from an artifact plugin only, as an init container",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := logging.RequireLoggerFromContext(ctx)
			containerName := os.Getenv(common.EnvVarContainerName)
			includeScriptOutput := os.Getenv(common.EnvVarIncludeScriptOutput) == "true"

			// The plugin server binds its socket inside this directory, so it has to exist
			// before the server starts. If the server wins that race, bind() fails with
			// ENOENT, the server exits, and the load below waits out its full 120s timeout
			// for a socket that can never appear.
			pluginName := wfv1.ArtifactPluginName(artifactPlugin)
			if err := os.MkdirAll(pluginName.SocketDir(), 0o755); err != nil {
				return fmt.Errorf("failed to create artifact plugin socket directory: %w", err)
			}

			name, args := args[0], args[1:]
			logger.WithFields(logging.Fields{"name": name, "args": args}).Debug(ctx, "starting command")

			go func() {
				command, closer, err := startCommand(ctx, name, args, &wfv1.Template{}, containerName, includeScriptOutput)
				if err != nil {
					logger.WithError(err).Error(ctx, "failed to start command")
					return
				}
				defer closer()
				// setup signal handlers
				signals := make(chan os.Signal, 1)
				defer close(signals)
				signal.Notify(signals)
				defer signal.Reset()

				forwardSignals(ctx, signals, command.Process.Pid, false)
			}()
			err := loadArtifactPlugin(ctx, pluginName)
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		},
	}
	command.Flags().StringVar(&artifactPlugin, "plugin-name", "", "Artifact plugin name")
	return &command
}

func loadArtifactPlugin(ctx context.Context, pluginName wfv1.ArtifactPluginName) error {
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)
	errHandler := wfExecutor.HandleError(ctx)
	defer errHandler()
	defer stats.LogStats()

	err := wfExecutor.LoadArtifactsFromPlugin(ctx, pluginName)
	if err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}
	return nil
}
