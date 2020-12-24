package commands

import (
	"context"
	"time"

	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewWaitCommand() *cobra.Command {
	var command = cobra.Command{
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
		// Killing sidecar containers
		err := wfExecutor.KillSidecars(ctx)
		if err != nil {
			wfExecutor.AddError(err)
		}
	}()

	// Wait for main container to complete
	waitErr := wfExecutor.Wait(ctx)
	if waitErr != nil {
		wfExecutor.AddError(waitErr)
		// do not return here so we can still try to kill sidecars & save outputs
	}

	// Capture output script result
	err := wfExecutor.CaptureScriptResult(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Capture output script exit code
	err = wfExecutor.CaptureScriptExitCode(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Saving logs
	logArt, err := wfExecutor.SaveLogs(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Saving output parameters
	err = wfExecutor.SaveParameters(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Saving output artifacts
	err = wfExecutor.SaveArtifacts(ctx)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	err = wfExecutor.AnnotateOutputs(ctx, logArt)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}

	// To prevent the workflow step from completing successfully, return the error occurred during wait.
	if waitErr != nil {
		return waitErr
	}

	return nil
}
