package commands

import (
	"context"
	"time"

	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewWaitCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "wait",
		Short: "wait for main container to finish and save artifacts",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			err := waitContainer(ctx)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func waitContainer(ctx context.Context) error {
	wfExecutor := initExecutor()

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	bgCtx := context.Background()

	defer wfExecutor.HandleError(bgCtx)    // Must be placed at the bottom of defers stack.
	defer wfExecutor.FinalizeOutput(bgCtx) // Ensures the LabelKeyReportOutputsCompleted is set to true.
	defer stats.LogStats()
	stats.StartStatsTicker(5 * time.Minute)

	// Create a new empty (placeholder) task result with LabelKeyReportOutputsCompleted set to false.
	wfExecutor.InitializeOutput(bgCtx)

	// Wait for main container to complete
	err := wfExecutor.Wait(ctx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Capture output script result
	err = wfExecutor.CaptureScriptResult(bgCtx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Saving output parameters
	err = wfExecutor.SaveParameters(bgCtx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Saving output artifacts
	artifacts, err := wfExecutor.SaveArtifacts(bgCtx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Save log artifacts
	logArtifacts := wfExecutor.SaveLogs(bgCtx)
	artifacts = append(artifacts, logArtifacts...)

	// Try to upsert TaskResult. If it fails, we will try to update the Pod's Annotations
	err = wfExecutor.ReportOutputs(bgCtx, artifacts)
	if err != nil {
		wfExecutor.AddError(err)
	}

	return wfExecutor.HasError()
}
