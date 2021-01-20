package commands

import (
	"context"

	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
)

func NewRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "run",
		SilenceUsage: true, // prevent usage being printe on error
		RunE: func(cmd *cobra.Command, args []string) error {
			name, args := args[0], args[1:]
			ctx := context.Background()
			wfExecutor := initExecutor()
			defer wfExecutor.HandleError(ctx)
			defer stats.LogStats()
			err := wfExecutor.LoadArtifacts(ctx)
			if err != nil {
				return err
			}
			err = wfExecutor.Run(ctx, name, args)
			if err := wfExecutor.Close(ctx); err != nil {
				return err
			}
			return err
		},
	}
}
