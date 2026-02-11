package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"

	"github.com/argoproj/argo-workflows/v3/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/tracing"
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
	ctx = tracing.InjectTraceContext(ctx)
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)
	defer func() {
		if err := wfExecutor.Tracing.Shutdown(context.WithoutCancel(ctx)); err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Failed to shutdown tracing")
		}
	}()
	span := trace.SpanFromContext(ctx)

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	//nolint:contextcheck
	bgCtx := trace.ContextWithSpan(logging.RequireLoggerFromContext(ctx).NewBackgroundContext(), span)
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
