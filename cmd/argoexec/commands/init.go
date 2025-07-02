package commands

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "init",
		Short: "Load artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
				cmd.SetContext(ctx)
			}
			log := logging.GetLoggerFromContext(ctx)
			if log == nil {
				log = logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
				ctx = logging.WithLogger(ctx, log)
				cmd.SetContext(ctx)
			}
			err := loadArtifacts(ctx)
			if err != nil {
				return fmt.Errorf("%+v", err)
			}
			return nil
		},
	}
	return &command
}

func loadArtifacts(ctx context.Context) error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError(ctx)
	defer stats.LogStats()

	if err := wfExecutor.Init(); err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Download input artifacts
	err := wfExecutor.StageFiles()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	err = wfExecutor.LoadArtifacts(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	return nil
}
