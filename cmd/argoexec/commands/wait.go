package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argoexec/executor"
	"github.com/argoproj/argo-workflows/v3/util/logging"
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

// nolint: contextcheck
func waitContainer(ctx context.Context) error {
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	//nolint:contextcheck
	bgCtx := logging.RequireLoggerFromContext(ctx).NewBackgroundContext()

	defer wfExecutor.HandleError(bgCtx)    // Must be placed at the bottom of defers stack.
	defer wfExecutor.FinalizeOutput(bgCtx) // Ensures the LabelKeyReportOutputsCompleted is set to true.
	defer func() {
		err := wfExecutor.KillArtifactSidecars(bgCtx)
		if err != nil {
			wfExecutor.AddError(bgCtx, err)
		}
	}()
	defer stats.LogStats()
	stats.StartStatsTicker(5 * time.Minute)

	// Create a new empty (placeholder) task result with LabelKeyReportOutputsCompleted set to false.
	wfExecutor.InitializeOutput(bgCtx)

	// Wait for main container to complete
	err := wfExecutor.Wait(ctx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
	}

	if wfExecutor.Template.Resource != nil {
		// Save log artifacts for resource template
		err = wfExecutor.ReportOutputsLogs(bgCtx)
		if err != nil {
			wfExecutor.AddError(ctx, err)
		}
		return wfExecutor.HasError()
	}

	// Capture output script result
	err = wfExecutor.CaptureScriptResult(bgCtx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
	}

	// Saving output parameters
	err = wfExecutor.SaveParameters(bgCtx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
	}

	// Saving output artifacts
	artifacts, err := wfExecutor.SaveArtifacts(bgCtx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
	}

	// Save log artifacts
	logArtifacts := wfExecutor.SaveLogs(bgCtx)
	artifacts = append(artifacts, logArtifacts...)

	err = wfExecutor.ReportOutputs(bgCtx, artifacts)
	if err != nil {
		wfExecutor.AddError(ctx, err)
	}

	return wfExecutor.HasError()
}
