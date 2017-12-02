package commands

import (
	"os"

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
	// Wait for main container to complete and kill sidecars
	err := wfExecutor.Wait()
	if err != nil {
		log.Errorf("Error waiting on main container to be ready, %+v", err)
	}
	err = wfExecutor.SaveArtifacts()
	if err != nil {
		log.Fatalf("Error saving output artifacts, %+v", err)
	}
	// Saving output parameters
	err = wfExecutor.SaveParameters()
	if err != nil {
		log.Fatalf("Error saving output parameters, %+v", err)
	}
	// Capture output script result
	err = wfExecutor.CaptureScriptResult()
	if err != nil {
		log.Fatalf("Error capturing script output, %+v", err)
	}
	err = wfExecutor.AnnotateOutputs()
	if err != nil {
		log.Fatalf("Error annotating outputs, %+v", err)
	}
	os.Exit(0)
}
