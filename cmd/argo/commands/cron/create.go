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

var (
	createCronWFExample = `# Create a cron workflow from a file

  argo cron create FILE1

# Create a cron workflow and print it as YAML

  argo cron create FILE1 --output yaml

# Create a cron workflow with relaxed validation

  argo cron create FILE1 --strict false

# Create a cron workflow with a custom schedule(override the schedule in the cron workflow)

  argo cron create FILE1 --schedule "0 0 * * *"`
)

func NewCreateCommand() *cobra.Command {
	var (
		cliCreateOpts  cliCreateOpts
		submitOpts     wfv1.SubmitOpts
		parametersFile string
	)
	command := &cobra.Command{
		Use:     "create FILE1 FILE2...",
		Short:   "create a cron workflow",
		Example: createCronWFExample,
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
