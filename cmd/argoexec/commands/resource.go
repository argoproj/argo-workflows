package commands

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func NewResourceCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "resource (get|create|apply|delete) MANIFEST",
		Short: "update a resource and wait for resource conditions",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			ctx := context.Background()
			err := execResource(ctx, args[0])
			if err != nil {
				log.Fatalf("%+v", err)
			}
		},
	}
	return &command
}

func execResource(ctx context.Context, action string) error {
	wfExecutor := initExecutor()
	defer wfExecutor.HandleError(ctx)
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
	manifestPath := common.ExecutorResourceManifestPath
	if wfExecutor.Template.Resource.ManifestFrom != nil {
		targetArtName := wfExecutor.Template.Resource.ManifestFrom.Artifact.Name
		for _, art := range wfExecutor.Template.Inputs.Artifacts {
			if art.Name == targetArtName {
				manifestPath = art.Path
				break
			}
		}
	}
	resourceNamespace, resourceName, selfLink, err := wfExecutor.ExecResource(
		action, manifestPath, wfExecutor.Template.Resource.Flags,
	)
	if err != nil {
		wfExecutor.AddError(err)
		return err
	}
	if !isDelete {
		err = wfExecutor.WaitResource(ctx, resourceNamespace, resourceName, selfLink)
		if err != nil {
			wfExecutor.AddError(err)
			return err
		}
		err = wfExecutor.SaveResourceParameters(ctx, resourceNamespace, resourceName)
		if err != nil {
			wfExecutor.AddError(err)
			return err
		}
	}
	return nil
}
