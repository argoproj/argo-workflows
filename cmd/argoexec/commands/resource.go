package commands

import (
	"fmt"
	"os"

	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewResourceCommand() *cobra.Command {
	var command = cobra.Command{
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
	return &command
}

func execResource(action string) error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError()
	err := wfExecutor.StageFiles()
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	isDelete := action == "delete"
	if isDelete && (wfExecutor.Template.Resource.SuccessCondition != "" || wfExecutor.Template.Resource.FailureCondition != "" || len(wfExecutor.Template.Outputs.Parameters) > 0) {
		err = fmt.Errorf("successCondition, failureCondition and outputs are not supported for delete action")
		wfExecutor.AddError(err)
		return err
	}
	resourceNamespace, resourceName, err := wfExecutor.ExecResource(action, common.ExecutorResourceManifestPath, isDelete)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	if !isDelete {
		err = wfExecutor.WaitResource(resourceNamespace, resourceName)
		if err != nil {
			wfExecutor.AddError(err)
			return err
		}
		err = wfExecutor.SaveResourceParameters(resourceNamespace, resourceName)
		if err != nil {
			wfExecutor.AddError(err)
			return err
		}
	}
	return nil
}
