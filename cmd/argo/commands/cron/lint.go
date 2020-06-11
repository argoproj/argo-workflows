package cron

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

func NewLintCommand() *cobra.Command {
	var (
		strict  bool
		offline bool
	)
	var command = &cobra.Command{
		Use:   "lint FILE...",
		Short: "validate files or directories of CronWorkflow manifests",
		Run: func(cmd *cobra.Command, args []string) {
			resources, err := validate.ParseResourcesFromFiles(args, strict)
			if err != nil {
				log.Fatal(err)
			}

			invalid := false

			if offline {
				templateGetter := templateresolution.WrapWorkflowTemplateList(resources.WorkflowTemplates)
				clusterTemplateGetter := templateresolution.WrapClusterWorkflowTemplateList(resources.ClusterWorkflowTemplates)

				for _, cron := range resources.CronWorkflows {
					conditions, err := validate.ValidateCronWorkflow(templateGetter, clusterTemplateGetter, &cron, true)
					if err != nil {
						log.Errorf("Error in CronWorkflow %s: %s", cron.Name, err)
						invalid = true
					}
					for _, condition := range *conditions {
						log.Warnf("Warning in CronWorkflow %s: %s", cron.Name, condition.Message)
					}
				}
			} else {
				ctx, apiClient := client.NewAPIClient()
				serviceClient := apiClient.NewCronWorkflowServiceClient()
				namespace := client.Namespace()

				for _, cron := range resources.CronWorkflows {
					_, err := serviceClient.LintCronWorkflow(ctx, &cronworkflow.LintCronWorkflowRequest{Namespace: namespace, CronWorkflow: &cron})
					if err != nil {
						log.Errorf("Error in CronWorkflow %s: %s", cron.Name, err)
						invalid = true
					}
				}
			}

			if invalid {
				log.Fatalf("Errors encountered in validation")
			}
			fmt.Printf("CronWorkflow manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict validation")
	command.Flags().BoolVar(&offline, "offline", false,
		"lint template references against local files instead of remote server state")
	return command
}
