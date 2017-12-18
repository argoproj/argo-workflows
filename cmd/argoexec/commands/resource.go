package commands

import (
	"os"

	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(resourceCmd)
}

var resourceCmd = &cobra.Command{
	Use:   "resource (get|create|apply|delete) MANIFEST",
	Short: "update a resource and wait for resource conditions",
	Run:   execResource,
}

func execResource(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}

	wfExecutor := initExecutor()
	err := wfExecutor.StageFiles()
	if err != nil {
		_ = wfExecutor.AddAnnotation(common.AnnotationKeyNodeMessage, err.Error())
		log.Fatalf("Error staing resource: %+v", err)
	}
	resourceName, err := wfExecutor.ExecResource(args[0], common.ExecutorResourceManifestPath)
	if err != nil {
		_ = wfExecutor.AddAnnotation(common.AnnotationKeyNodeMessage, err.Error())
		log.Fatalf("Error running %s resource: %+v", args[0], err)
	}
	err = wfExecutor.WaitResource(resourceName)
	if err != nil {
		_ = wfExecutor.AddAnnotation(common.AnnotationKeyNodeMessage, err.Error())
		log.Fatalf("Error waiting for resource %s: %+v", resourceName, err)
	}
}
