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

type cliCreateOpts struct {
	output   string // --output
	schedule string // --schedule
	strict   bool   // --strict
}

func NewCreateCommand() *cobra.Command {
	var (
		cliCreateOpts  cliCreateOpts
		submitOpts     wfv1.SubmitOpts
		parametersFile string
	)
	command := &cobra.Command{
		Use:   "create FILE1 FILE2...",
		Short: "create a cron workflow",
		Run: func(cmd *cobra.Command, args []string) {
			checkArgs(cmd, args, parametersFile, &submitOpts)

			CreateCronWorkflows(cmd.Context(), args, &cliCreateOpts, &submitOpts)
		},
	}

	util.PopulateSubmitOpts(command, &submitOpts, &parametersFile, false)
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	command.Flags().StringVar(&cliCreateOpts.schedule, "schedule", "", "override cron workflow schedule")
	return command
}

func CreateCronWorkflows(ctx context.Context, filePaths []string, cliOpts *cliCreateOpts, submitOpts *wfv1.SubmitOpts) {
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewCronWorkflowServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	cronWorkflows := generateCronWorkflows(filePaths, cliOpts.strict)

	for _, cronWf := range cronWorkflows {
		if cliOpts.schedule != "" {
			cronWf.Spec.Schedule = cliOpts.schedule
		}

		newWf := wfv1.Workflow{Spec: cronWf.Spec.WorkflowSpec}
		err := util.ApplySubmitOpts(&newWf, submitOpts)
		if err != nil {
			log.Fatal(err)
		}
		cronWf.Spec.WorkflowSpec = newWf.Spec
		// We have only copied the workflow spec to the cron workflow but not the metadata
		// that includes name and generateName. Here we copy the metadata to the cron
		// workflow's metadata and remove the unnecessary and mutually exclusive part.
		if generateName := newWf.ObjectMeta.GenerateName; generateName != "" {
			cronWf.ObjectMeta.GenerateName = generateName
			cronWf.ObjectMeta.Name = ""
		}
		if name := newWf.ObjectMeta.Name; name != "" {
			cronWf.ObjectMeta.Name = name
			cronWf.ObjectMeta.GenerateName = ""
		}
		if cronWf.Namespace == "" {
			cronWf.Namespace = client.Namespace()
		}
		created, err := serviceClient.CreateCronWorkflow(ctx, &cronworkflowpkg.CreateCronWorkflowRequest{
			Namespace:    cronWf.Namespace,
			CronWorkflow: &cronWf,
		})
		if err != nil {
			log.Fatalf("Failed to create cron workflow: %v", err)
		}
		fmt.Print(getCronWorkflowGet(created))
	}
}
