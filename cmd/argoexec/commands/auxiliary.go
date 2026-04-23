package commands

import (
	"context"
	"time"

	"github.com/argoproj/pkg/stats"
	"go.opentelemetry.io/otel/trace"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// runAuxiliaryContainer runs the executor with the shared lifecycle scaffolding
// used by both the legacy `wait` command and the init-less `supervisor` command:
// tracing init/shutdown, span start/end, background context for cleanup work
// that must outlive ctx cancellation, and the standard defer stack
// (errHandler at the bottom, then FinalizeOutput, KillArtifactSidecars,
// LogStats, in LIFO defer order).
//
// startSpan is supplied by the caller to choose the appropriate top-level span
// (`StartRunWaitContainer` for wait, `StartRunSupervisorContainer` for
// supervisor). body runs inside the span with both the cancellable ctx (for
// waiting on main) and bgCtx (for capturing outputs even during termination).
//
// Keeping this in one place prevents the two callers from drifting when a new
// cross-cutting concern (e.g. additional defer cleanup, additional ticker)
// needs to land in both.
//
//nolint:contextcheck
func runAuxiliaryContainer(
	ctx context.Context,
	startSpan func(we *wfexecutor.WorkflowExecutor, ctx context.Context) (context.Context, trace.Span),
	body func(ctx, bgCtx context.Context, we *wfexecutor.WorkflowExecutor) error,
) error {
	ctx = tracing.InjectTraceContext(ctx)
	wfExecutor := argoexecexecutor.Init(ctx, clientConfig, varRunArgo)
	defer func() {
		if err := wfExecutor.Tracing.Shutdown(context.WithoutCancel(ctx)); err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Failed to shutdown tracing")
		}
	}()

	ctx, span := startSpan(wfExecutor, ctx)
	defer span.End()

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	//nolint:contextcheck
	bgCtx := trace.ContextWithSpan(logging.RequireLoggerFromContext(ctx).NewBackgroundContext(), span)

	errHandler := wfExecutor.HandleError(bgCtx)
	defer errHandler()                     // Must be placed at the bottom of defers stack.
	defer wfExecutor.FinalizeOutput(bgCtx) // Ensures the LabelKeyReportOutputsCompleted is set to true.
	defer func() {
		if err := wfExecutor.KillArtifactSidecars(bgCtx); err != nil {
			wfExecutor.AddError(bgCtx, err)
		}
	}()
	defer stats.LogStats()

	stats.StartStatsTicker(5 * time.Minute)

	return body(ctx, bgCtx, wfExecutor)
}
