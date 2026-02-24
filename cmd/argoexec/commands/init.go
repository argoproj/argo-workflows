package commands

import (
	"context"
	"fmt"

	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
)

func NewInitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "init",
		Short: "Load artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := loadArtifacts(cmd.Context())
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		},
	}
	return &command
}

func loadArtifacts(ctx context.Context) error {
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)
	errHandler := wfExecutor.HandleError(ctx)
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
