package executor

import (
	"context"
)

// PostMain drives the task from the point the user's main container has
// started: the plan's Run phase (create the placeholder task result, wait
// for main to exit) on ctx, then its Collect phase (capture outputs and
// logs, report them) on bgCtx. It is shared between the `argoexec wait`
// command (legacy pod layout) and the `argoexec supervisor` command
// (init-less pod layout).
//
// If preMainFailed is true, the supervisor's Prepare phase failed (or the
// ready marker could not be written). The user's main never produced
// outputs, so Collect stages marked NeedsMainOutputs are skipped — they
// would fail with confusing "file not found" errors on required
// artifacts. Main is still waited for and its logs captured so the
// failure is surfaced cleanly.
//
// The caller owns tracing init/shutdown and the defer stack
// (errHandler, FinalizeOutput, KillArtifactSidecars, stats). bgCtx is a
// span-scoped background context used for calls that must not be
// cancelled with the parent ctx so that outputs are still captured
// during termination.
func (we *WorkflowExecutor) PostMain(ctx, bgCtx context.Context, preMainFailed bool) error {
	plan := we.selectedPlan()
	we.runRecording(ctx, plan.Run, !preMainFailed)
	we.runRecording(bgCtx, plan.Collect, !preMainFailed)
	return we.HasError()
}
