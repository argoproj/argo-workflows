package commands

import (
	"time"

	"github.com/argoproj/argo/util/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(waitCmd)
}

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "wait for main container to finish and save artifacts",
	Run:   waitContainer,
}

func waitContainer(cmd *cobra.Command, args []string) {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError()
	defer stats.LogStats()
	stats.StartStatsTicker(5 * time.Minute)

	// Wait for main container to complete and kill sidecars
	err := wfExecutor.Wait()
	if err != nil {
		wfExecutor.AddError(err)
		log.Errorf("Error on wait, %+v", err)
	}
	err = wfExecutor.SaveArtifacts()
	if err != nil {
		wfExecutor.AddError(err)
		log.Fatalf("Error saving output artifacts, %+v", err)
	}
	// Saving output parameters
	err = wfExecutor.SaveParameters()
	if err != nil {
		wfExecutor.AddError(err)
		log.Fatalf("Error saving output parameters, %+v", err)
	}
	// Capture output script result
	err = wfExecutor.CaptureScriptResult()
	if err != nil {
		wfExecutor.AddError(err)
		log.Fatalf("Error capturing script output, %+v", err)
	}
	err = wfExecutor.AnnotateOutputs()
	if err != nil {
		wfExecutor.AddError(err)
		log.Fatalf("Error annotating outputs, %+v", err)
	}
}
