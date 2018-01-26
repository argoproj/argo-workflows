package commands

import (
	"github.com/argoproj/argo/util/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Load artifacts",
	Run:   loadArtifacts,
}

func loadArtifacts(cmd *cobra.Command, args []string) {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError()
	defer stats.LogStats()

	// Download input artifacts
	err := wfExecutor.StageFiles()
	if err != nil {
		wfExecutor.AddError(err)
		log.Fatalf("Error loading staging files: %+v", err)
	}
	err = wfExecutor.LoadArtifacts()
	if err != nil {
		wfExecutor.AddError(err)
		log.Fatalf("Error downloading input artifacts: %+v", err)
	}
}
