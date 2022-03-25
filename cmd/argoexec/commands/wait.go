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
			ctx := context.Background()
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
	defer wfExecutor.HandleError(ctx) // Must be placed at the bottom of defers stack.
	defer stats.LogStats()
	stats.StartStatsTicker(5 * time.Minute)

	defer func() {
		if err := wfExecutor.KillSidecars(ctx); err != nil {
			wfExecutor.AddError(err)
		}
	}()

	// Wait for main container to complete
	err := wfExecutor.Wait(ctx)
	if err != nil {
		wfExecutor.AddError(err)
	}
	// Capture output script result
	err = wfExecutor.CaptureScriptResult(ctx)
	if err != nil {
		wfExecutor.AddError(err)
	}
	// Saving logs
	logArt, err := wfExecutor.SaveLogs(ctx)
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
	// Annotating pod with output
	err = wfExecutor.ReportOutputs(ctx, logArt)
	if err != nil {
		wfExecutor.AddError(err)
	}

	return wfExecutor.HasError()
}
