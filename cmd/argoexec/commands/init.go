package commands

import (
	"github.com/argoproj/argo/util/stats"
	"github.com/argoproj/argo/workflow/common"
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
	defer wfExecutor.AnnotatePanic()
	defer stats.LogStats()

	// Download input artifacts
	err := wfExecutor.StageFiles()
	if err != nil {
		_ = wfExecutor.AddAnnotation(common.AnnotationKeyNodeMessage, err.Error())
		log.Fatalf("Error loading staging files: %+v", err)
	}
	err = wfExecutor.LoadArtifacts()
	if err != nil {
		_ = wfExecutor.AddAnnotation(common.AnnotationKeyNodeMessage, err.Error())
		log.Fatalf("Error downloading input artifacts: %+v", err)
	}
}
