package executor

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// PostMain drives the phase after the user's main container has started:
// wait for main to exit, then capture script result, output parameters,
// output artifacts and logs, and report outputs. It is shared between
// the `argoexec wait` command (legacy pod layout) and the new
// `argoexec supervisor` command (init-less pod layout).
//
// If preMainFailed is true, the supervisor's pre-main phase failed (or the
// ready marker could not be written). In that case the user's main never
// produced outputs and we must NOT try to save script result / output
// parameters / output artifacts — those calls would fail with confusing
// "file not found" errors on required artifacts. We still wait for main
// to exit and capture logs so the failure is surfaced cleanly.
//
// The caller owns tracing init/shutdown and the defer stack
// (errHandler, FinalizeOutput, KillArtifactSidecars, stats). bgCtx is a
// span-scoped background context used for calls that must not be
// cancelled with the parent ctx so that outputs are still captured
// during termination.
func (we *WorkflowExecutor) PostMain(ctx, bgCtx context.Context, preMainFailed bool) error {
	we.InitializeOutput(bgCtx)

	if err := we.Wait(ctx); err != nil {
		we.AddError(ctx, err)
	}

	if we.Template.Resource != nil {
		if err := we.ReportOutputsLogs(bgCtx); err != nil {
			we.AddError(ctx, err)
		}
		return we.HasError()
	}

	var artifacts []wfv1.Artifact
	if !preMainFailed {
		if err := we.CaptureScriptResult(bgCtx); err != nil {
			we.AddError(ctx, err)
		}

		if err := we.SaveParameters(bgCtx); err != nil {
			we.AddError(ctx, err)
		}

		var err error
		artifacts, err = we.SaveArtifacts(bgCtx)
		if err != nil {
			we.AddError(ctx, err)
		}
	}

	// Save log artifacts (still useful even when pre-main failed — main's
	// stdout/stderr contains the emissary's "supervisor pre-main failed"
	// message).
	logArtifacts := we.SaveLogs(bgCtx)
	artifacts = append(artifacts, logArtifacts...)

	if err := we.ReportOutputs(bgCtx, artifacts); err != nil {
		we.AddError(ctx, err)
	}

	return we.HasError()
}
