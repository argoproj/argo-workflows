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
	defer wfExecutor.HandleError(ctx)    // Must be placed at the bottom of defers stack.
	defer wfExecutor.FinalizeOutput(ctx) // Ensures the LabelKeyReportOutputsCompleted is set to true.
	defer stats.LogStats()
	stats.StartStatsTicker(5 * time.Minute)

	// Wait for main container to complete
	err := wfExecutor.Wait(ctx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Don't allow cancellation to impact capture of results, parameters, or artifacts
	ctx = context.Background()

	// Capture output script result
	err = wfExecutor.CaptureScriptResult(ctx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Saving output parameters
	err = wfExecutor.SaveParameters(ctx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Saving output artifacts
	err = wfExecutor.SaveArtifacts(ctx)
	if err != nil {
		wfExecutor.AddError(err)
	}

	// Save log artifacts
	logArtifacts := wfExecutor.SaveLogs(ctx)

	// Try to upsert TaskResult. If it fails, we will try to update the Pod's Annotations
	wfExecutor.ReportOutputs(ctx, logArtifacts)

	return wfExecutor.HasError()
}
