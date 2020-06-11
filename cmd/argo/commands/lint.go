package commands

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
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
		Short: "validate files or directories of workflow manifests",
		Run: func(cmd *cobra.Command, args []string) {
			resources, err := validate.ParseResourcesFromFiles(args, strict)
			if err != nil {
				log.Fatal(err)
			}

			invalid := false
			opts := validate.ValidateOpts{Lint: true}

			if offline {
				templateGetter := templateresolution.WrapWorkflowTemplateList(resources.WorkflowTemplates)
				clusterTemplateGetter := templateresolution.WrapClusterWorkflowTemplateList(resources.ClusterWorkflowTemplates)

				for _, wf := range resources.Workflows {
					conditions, err := validate.ValidateWorkflow(templateGetter, clusterTemplateGetter, &wf, opts)
					if err != nil {
						log.Errorf("Error in workflow %s: %s", wf.Name, err)
						invalid = true
					}
					for _, condition := range *conditions {
						log.Warnf("Warning in workflow %s: %s", wf.Name, condition.Message)
					}
				}
			} else {
				ctx, apiClient := client.NewAPIClient()
				serviceClient := apiClient.NewWorkflowServiceClient()
				namespace := client.Namespace()

				for _, wf := range resources.Workflows {
					_, err := serviceClient.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Namespace: namespace, Workflow: &wf})
					if err != nil {
						log.Errorf("Error in workflow %s: %s", wf.Name, err)
						invalid = true
					}
				}
			}

			if invalid {
				log.Fatalf("Errors encountered in validation")
			}
			fmt.Printf("Workflow manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	command.Flags().BoolVar(&offline, "offline", false,
		"lint template references against local files instead of remote server state")
	return command
}
