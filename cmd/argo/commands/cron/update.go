package cron

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func NewUpdateCommand() *cobra.Command {
	var (
		cliCreateOpts cliCreateOpts
		submitOpts    wfv1.SubmitOpts
	)
	command := &cobra.Command{
		Use:   "update FILE1 FILE2...",
		Short: "update a cron workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			updateCronWorkflows(args, &cliCreateOpts, &submitOpts)
		},
	}

	util.PopulateSubmitOpts(command, &submitOpts, false)
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateCronWorkflows(filePaths []string, cliOpts *cliCreateOpts, submitOpts *wfv1.SubmitOpts) {
	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewCronWorkflowServiceClient()

	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var cronWorkflows []wfv1.CronWorkflow
	for _, body := range fileContents {
		cronWfs := unmarshalCronWorkflows(body, cliOpts.strict)
		cronWorkflows = append(cronWorkflows, cronWfs...)
	}

	if len(cronWorkflows) == 0 {
		log.Println("No CronWorkflows found in given files")
		os.Exit(1)
	}

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
