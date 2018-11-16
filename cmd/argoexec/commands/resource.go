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
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		}
		err := execResource(args[0])
		if err != nil {
			log.Fatalf("%+v", err)
		}
	},
}

func execResource(action string) error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError()
	err := wfExecutor.StageFiles()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	resourceName, err := wfExecutor.ExecResource(action, common.ExecutorResourceManifestPath)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	err = wfExecutor.WaitResource(resourceName)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	err = wfExecutor.SaveResourceParameters(resourceName)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	return nil
}
