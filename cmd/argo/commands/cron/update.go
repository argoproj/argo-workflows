package cron

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

type cliUpdateOpts struct {
	output string // --output
	strict bool   // --strict
}

func NewUpdateCommand() *cobra.Command {
	var (
		cliUpdateOpts  cliUpdateOpts
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
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(cmd, args, parametersFile, &submitOpts)

			updateCronWorkflows(cmd.Context(), args, &cliUpdateOpts, &submitOpts)
		},
	}

	util.PopulateSubmitOpts(command, &submitOpts, &parametersFile, false)
	command.Flags().StringVarP(&cliUpdateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliUpdateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateCronWorkflows(ctx context.Context, filePaths []string, cliOpts *cliUpdateOpts, submitOpts *wfv1.SubmitOpts) {
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewCronWorkflowServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	cronWorkflows := generateCronWorkflows(filePaths, cliOpts.strict)

	for _, cronWf := range cronWorkflows {
		newWf := wfv1.Workflow{Spec: cronWf.Spec.WorkflowSpec}
		err := util.ApplySubmitOpts(&newWf, submitOpts)
		if err != nil {
			log.Fatal(err)
		}
		if cronWf.Namespace == "" {
			cronWf.Namespace = client.Namespace()
		}
		current, err := serviceClient.GetCronWorkflow(ctx, &cronworkflowpkg.GetCronWorkflowRequest{
			Name:      cronWf.Name,
			Namespace: cronWf.Namespace,
		})
		if err != nil {
			log.Fatalf("Failed to get existing cron workflow %q to update: %v", cronWf.Name, err)
		}
		cronWf.ResourceVersion = current.ResourceVersion
		updated, err := serviceClient.UpdateCronWorkflow(ctx, &cronworkflowpkg.UpdateCronWorkflowRequest{
			Namespace:    cronWf.Namespace,
			CronWorkflow: &cronWf,
		})
		if err != nil {
			log.Fatalf("Failed to update workflow template: %v", err)
		}
		fmt.Print(getCronWorkflowGet(updated))
	}
}
