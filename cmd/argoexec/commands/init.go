package commands

import (
	"context"
	"fmt"

	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

func NewInitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "init",
		Short: "Load artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := tracing.InjectTraceContext(cmd.Context())
			ctx, closer, err := teeContainerLogs(ctx, varRunArgo, common.InitContainerName)
			if err != nil {
				// log tee failure is a log-archive failure; init's primary processing continues
				logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Failed to set up log tee")
			} else {
				defer closer()
			}

			if err := loadArtifacts(ctx); err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		},
	}
	return &command
}

func loadArtifacts(ctx context.Context) error {
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)
	defer func() {
		if err := wfExecutor.Tracing.Shutdown(context.WithoutCancel(ctx)); err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Failed to shutdown tracing")
		}
	}()
	errHandler := wfExecutor.HandleError(ctx)
	ctx, span := wfExecutor.Tracing.StartRunInitContainer(ctx, wfExecutor.WorkflowName(), wfExecutor.Namespace)
	defer span.End()
	defer errHandler()
	defer stats.LogStats()

	if err := wfExecutor.Init(); err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}
	err := wfExecutor.StageFiles(ctx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}
	// Download input artifacts
	err = wfExecutor.LoadArtifactsWithoutPlugins(ctx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}
	return nil
}
