package commands

import (
	"context"
	"os"

	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/workflow/common"
)

func NewInitCommand() *cobra.Command {
	var command = cobra.Command{
		Use:   "init",
		Short: "Load artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			err := installBinary()
			if err != nil {
				return err
			}
			wfExecutor := initExecutor()
			defer wfExecutor.HandleError(ctx)
			defer stats.LogStats()

			// Download input artifacts
			err = wfExecutor.StageFiles()
			if err != nil {
				wfExecutor.AddError(err)
				return err
			}
			if os.Getenv(common.EnvVarContainerRuntimeExecutor) != common.ContainerRuntimeExecutorInline {
				err = wfExecutor.LoadArtifacts(ctx)
				if err != nil {
					wfExecutor.AddError(err)
					return err
				}
			}
			return nil
		},
	}
	return &command
}
