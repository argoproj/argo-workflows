package commands

import (
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
			err := waitContainer()
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func waitContainer() error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError()
	defer stats.LogStats()
	stats.StartStatsTicker(5 * time.Minute)

	// Wait for main container to complete
	err := wfExecutor.Wait()
	if err != nil {
		wfExecutor.AddError(err)
		// do not return here so we can still try to kill sidecars & save outputs
	}
	err = wfExecutor.KillSidecars()
	if err != nil {
		wfExecutor.AddError(err)
		// do not return here so we can still try save outputs
	}
	logArt, err := wfExecutor.SaveLogs()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Saving output parameters
	err = wfExecutor.SaveParameters()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Saving output artifacts
	err = wfExecutor.SaveArtifacts()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	// Capture output script result
	err = wfExecutor.CaptureScriptResult()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	err = wfExecutor.AnnotateOutputs(logArt)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	return nil
}
