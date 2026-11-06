package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"

	"github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
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
func execData(ctx context.Context) (err error) {
	ctx = tracing.InjectTraceContext(ctx)
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)
	defer func() {
		if shutdownErr := wfExecutor.Tracing.Shutdown(context.WithoutCancel(ctx)); shutdownErr != nil {
			logging.RequireLoggerFromContext(ctx).WithError(shutdownErr).Error(ctx, "Failed to shutdown tracing")
		}
	}()
	span := trace.SpanFromContext(ctx)

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	//nolint:contextcheck
	bgCtx := trace.ContextWithSpan(logging.RequireLoggerFromContext(ctx).NewBackgroundContext(), span)
	errHandler := wfExecutor.HandleError(bgCtx)
	defer errHandler()
	// Writes the outputs with LabelKeyReportOutputsCompleted set to true.
	defer func() { err = wfExecutor.FinalizeOutputFor(bgCtx, err) }()

	err = wfExecutor.Data(ctx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}
	return nil
}
