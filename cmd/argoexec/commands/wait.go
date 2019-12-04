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

	defer func() {
		// Killing sidecar containers
		err := wfExecutor.KillSidecars()
		if err != nil {
			log.Errorf("Failed to kill sidecars: %s", err.Error())
		}
	}()

	// Wait for main container to complete
	waitErr := wfExecutor.Wait()
	if waitErr != nil {
		wfExecutor.AddError(waitErr)
		// do not return here so we can still try to kill sidecars & save outputs
	}

	// Saving logs
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

	// To prevent the workflow step from completing successfully, return the error occurred during wait.
	if waitErr != nil {
		return waitErr
	}

	return nil
}
