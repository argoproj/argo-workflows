package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func NewDataCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "data",
		Short: "Process data",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := execData(cmd.Context())
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		},
	}
	return &command
}

//nolint:contextcheck
func execData(ctx context.Context) error {
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	//nolint:contextcheck
	bgCtx := logging.RequireLoggerFromContext(ctx).NewBackgroundContext()
	// Create a new empty (placeholder) task result with LabelKeyReportOutputsCompleted set to false.
	errHandler := wfExecutor.HandleError(bgCtx)
	defer errHandler()
	defer wfExecutor.FinalizeOutput(bgCtx) // Ensures the LabelKeyReportOutputsCompleted is set to true.

	err := wfExecutor.Data(ctx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}
	return nil
}
