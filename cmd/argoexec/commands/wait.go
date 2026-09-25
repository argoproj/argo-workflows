package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"

	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
)

func NewWaitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "wait",
		Short: "wait for main container to finish and save artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := waitContainer(cmd.Context())
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		},
	}
	return &command
}

func waitContainer(ctx context.Context) error {
	return runAuxiliaryContainer(ctx,
		func(we *wfexecutor.WorkflowExecutor, ctx context.Context) (context.Context, trace.Span) {
			return we.Tracing.StartRunWaitContainer(ctx, we.WorkflowName(), we.Namespace)
		},
		func(ctx, bgCtx context.Context, we *wfexecutor.WorkflowExecutor) error {
			// Legacy wait: pre-main ran in the init container; if it failed
			// the pod never got here. Always pass preMainFailed=false.
			return we.PostMain(ctx, bgCtx, false)
		},
	)
}
