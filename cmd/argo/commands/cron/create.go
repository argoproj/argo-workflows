package cron

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

type cliCreateOpts struct {
	output   common.EnumFlagValue // --output
	schedule string               // --schedule
	strict   bool                 // --strict
}

func NewCreateCommand() *cobra.Command {
	var (
		cliCreateOpts  = cliCreateOpts{output: common.NewPrintWorkflowOutputValue("")}
		submitOpts     wfv1.SubmitOpts
		parametersFile string
	)
	command := &cobra.Command{
		Use:   "create FILE1 FILE2...",
		Short: "create a cron workflow",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if parametersFile != "" {
				err := util.ReadParametersFile(cmd.Context(), parametersFile, &submitOpts)
				if err != nil {
					return err
				}
			}
			return CreateCronWorkflows(cmd.Context(), args, &cliCreateOpts, &submitOpts)
		},
	}

	util.PopulateSubmitOpts(command, &submitOpts, &parametersFile, false)
	command.Flags().VarP(&cliCreateOpts.output, "output", "o", "Output format. "+cliCreateOpts.output.Usage())
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	command.Flags().StringVar(&cliCreateOpts.schedule, "schedule", "", "override cron workflow schedule")
	return command
}

func CreateCronWorkflows(ctx context.Context, filePaths []string, cliOpts *cliCreateOpts, submitOpts *wfv1.SubmitOpts) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewCronWorkflowServiceClient()
	if err != nil {
		return err
	}

	cronWorkflows := generateCronWorkflows(ctx, filePaths, cliOpts.strict)

	for _, cronWf := range cronWorkflows {
		if cliOpts.schedule != "" {
			cronWf.Spec.Schedule = cliOpts.schedule
		}

		newWf := wfv1.Workflow{Spec: cronWf.Spec.WorkflowSpec}
		err := util.ApplySubmitOpts(&newWf, submitOpts)
		if err != nil {
			return err
		}
		cronWf.Spec.WorkflowSpec = newWf.Spec
		// We have only copied the workflow spec to the cron workflow but not the metadata
		// that includes name and generateName. Here we copy the metadata to the cron
		// workflow's metadata and remove the unnecessary and mutually exclusive part.
		if generateName := newWf.GenerateName; generateName != "" {
			cronWf.GenerateName = generateName
			cronWf.Name = ""
		}
		if name := newWf.Name; name != "" {
			cronWf.Name = name
			cronWf.GenerateName = ""
		}
		if cronWf.Namespace == "" {
			cronWf.Namespace = client.Namespace(ctx)
		}
		created, err := serviceClient.CreateCronWorkflow(ctx, &cronworkflowpkg.CreateCronWorkflowRequest{
			Namespace:    cronWf.Namespace,
			CronWorkflow: &cronWf,
		})
		if err != nil {
			return fmt.Errorf("failed to create cron workflow: %v", err)
		}
		fmt.Print(getCronWorkflowGet(ctx, created))
	}
	return nil
}
