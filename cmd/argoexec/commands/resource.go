package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"

	"github.com/argoproj/argo-workflows/v3/cmd/argoexec/executor"
)

func NewResourceCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "resource (get|create|apply|delete) MANIFEST",
		Short: "update a resource and wait for resource conditions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := execResource(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("%+v", err)
			}
			return nil
		},
	}
	return &command
}

// nolint: contextcheck
func execResource(ctx context.Context, action string) error {
	wfExecutor := executor.Init(ctx, clientConfig, varRunArgo)

	// Don't allow cancellation to impact capture of results, parameters, artifacts, or defers.
	// nolint:contextcheck
	bgCtx := logging.RequireLoggerFromContext(ctx).NewBackgroundContext()

	wfExecutor.InitializeOutput(bgCtx)
	defer wfExecutor.HandleError(bgCtx)
	if !wfExecutor.Template.SaveLogsAsArtifact() {
		defer wfExecutor.FinalizeOutput(bgCtx) //Ensures the LabelKeyReportOutputsCompleted is set to true.
	}
	err := wfExecutor.StageFiles(ctx)
	if err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}

	isDelete := action == "delete"
	if isDelete && (wfExecutor.Template.Resource.SuccessCondition != "" || wfExecutor.Template.Resource.FailureCondition != "" || len(wfExecutor.Template.Outputs.Parameters) > 0) {
		err = fmt.Errorf("successCondition, failureCondition and outputs are not supported for delete action")
		wfExecutor.AddError(ctx, err)
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
	resourceNamespace, resourceName, selfLink, err := wfExecutor.ExecResource(ctx,
		action, manifestPath, wfExecutor.Template.Resource.Flags,
	)
	if err != nil {
		wfExecutor.AddError(ctx, err)
		return err
	}
	if !isDelete {
		err = wfExecutor.WaitResource(ctx, resourceNamespace, resourceName, selfLink)
		if err != nil {
			wfExecutor.AddError(ctx, err)
			return err
		}
		err = wfExecutor.SaveResourceParameters(ctx, resourceNamespace, resourceName)
		if err != nil {
			wfExecutor.AddError(ctx, err)
			return err
		}
	}
	return nil
}
