package cron

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

type cliUpdateOpts struct {
	output common.EnumFlagValue // --output
	strict bool                 // --strict
}

func NewUpdateCommand() *cobra.Command {
	var (
		cliUpdateOpts  = cliUpdateOpts{output: common.NewPrintWorkflowOutputValue("")}
		submitOpts     wfv1.SubmitOpts
		parametersFile string
	)
	command := &cobra.Command{
		Use:   "update FILE1 FILE2...",
		Short: "update a cron workflow",
		Example: `# Update a Cron Workflow Template:
  argo cron update FILE1
	
# Update a Cron Workflow Template and print it as YAML:
  argo cron update FILE1 --output yaml
  
# Update a Cron Workflow Template with relaxed validation:
  argo cron update FILE1 --strict false
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if parametersFile != "" {
				err := util.ReadParametersFile(cmd.Context(), parametersFile, &submitOpts)
				if err != nil {
					return err
				}
			}
			return updateCronWorkflows(cmd.Context(), args, &cliUpdateOpts, &submitOpts)
		},
	}

	util.PopulateSubmitOpts(command, &submitOpts, &parametersFile, false)
	command.Flags().VarP(&cliUpdateOpts.output, "output", "o", "Output format. "+cliUpdateOpts.output.Usage())
	command.Flags().BoolVar(&cliUpdateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateCronWorkflows(ctx context.Context, filePaths []string, cliOpts *cliUpdateOpts, submitOpts *wfv1.SubmitOpts) error {
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
		newWf := wfv1.Workflow{Spec: cronWf.Spec.WorkflowSpec}
		err := util.ApplySubmitOpts(&newWf, submitOpts)
		if err != nil {
			return err
		}
		if cronWf.Namespace == "" {
			cronWf.Namespace = client.Namespace(ctx)
		}
		current, err := serviceClient.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{
			Name:      cronWf.Name,
			Namespace: cronWf.Namespace,
		})
		if err != nil {
			return fmt.Errorf("failed to get existing cron workflow %q to update: %v", cronWf.Name, err)
		}
		cronWf.ResourceVersion = current.ResourceVersion
		updated, err := serviceClient.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{
			Namespace:    cronWf.Namespace,
			CronWorkflow: &cronWf,
		})
		if err != nil {
			return fmt.Errorf("failed to update workflow template: %v", err)
		}
		fmt.Print(getCronWorkflowGet(ctx, updated))
	}
	return nil
}
