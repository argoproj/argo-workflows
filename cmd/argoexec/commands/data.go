package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewDataCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "data",
		Short: "Process data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			err := execData(ctx)
			if err != nil {
				return fmt.Errorf("%+v", err)
			}
			return nil
		},
	}
	return &command
}

func execData(ctx context.Context) error {
	wfExecutor := initExecutor()

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	bgCtx := context.Background()
	// Create a new empty (placeholder) task result with LabelKeyReportOutputsCompleted set to false.
	wfExecutor.InitializeOutput(bgCtx)
	defer wfExecutor.HandleError(bgCtx)
	defer wfExecutor.FinalizeOutput(bgCtx) //Ensures the LabelKeyReportOutputsCompleted is set to true.

	err := wfExecutor.Data(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	return nil
}
