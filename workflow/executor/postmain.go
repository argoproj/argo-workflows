package executor

import (
	"context"
)

// PostMain drives the task from the end of Prepare: the plan's Run phase
// (create the placeholder task result, then wait for main's container in
// a pod, or run main in a worker) on ctx, then its Collect phase (capture
// outputs and logs, report them) on bgCtx. The wait, supervisor and worker
// roles all use it.
//
// If preMainFailed is true, the supervisor's Prepare phase failed (or the
// ready marker could not be written). The user's main never produced
// outputs, so Collect stages marked NeedsMainOutputs are skipped — they
// would fail with confusing "file not found" errors on required
// artifacts. Main is still waited for and its logs captured so the
// failure is surfaced cleanly.
//
// The Harness running the task's Role owns tracing, failure reporting
// and the role's Finally stages (FinalizeOutput, KillArtifactSidecars).
// bgCtx is a span-scoped background context used for calls that must not
// be cancelled with the parent ctx so that outputs are still captured
// during termination.
func (we *WorkflowExecutor) PostMain(ctx, bgCtx context.Context, preMainFailed bool) error {
	plan := we.selectedPlan()
	we.runRecording(ctx, plan.Run, !preMainFailed)
	we.runRecording(bgCtx, plan.Collect, !preMainFailed)
	return we.HasError()
}
