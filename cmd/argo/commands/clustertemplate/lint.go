package clustertemplate

import (
	"fmt"

	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
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
		Short: "validate files or directories of ClusterWorkflowTemplate manifests",
		Run: func(cmd *cobra.Command, args []string) {
			resources, err := validate.ParseResourcesFromFiles(args, strict)
			if err != nil {
				log.Fatal(err)
			}

			invalid := false

			if offline {
				templateGetter := templateresolution.WrapWorkflowTemplateList(resources.WorkflowTemplates)
				clusterTemplateGetter := templateresolution.WrapClusterWorkflowTemplateList(resources.ClusterWorkflowTemplates)

				for _, wftmpl := range resources.ClusterWorkflowTemplates {
					conditions, err := validate.ValidateClusterWorkflowTemplate(templateGetter, clusterTemplateGetter, &wftmpl, true)
					if err != nil {
						log.Errorf("Error in ClusterWorkflowTemplate %s: %s", wftmpl.Name, err)
						invalid = true
					}
					for _, condition := range *conditions {
						log.Warnf("Warning in ClusterWorkflowTemplate %s: %s", wftmpl.Name, condition.Message)
					}
				}
			} else {
				ctx, apiClient := client.NewAPIClient()
				serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()

				for _, wftmpl := range resources.ClusterWorkflowTemplates {
					_, err := serviceClient.LintClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest{Template: &wftmpl})
					if err != nil {
						log.Errorf("Error in ClusterWorkflowTemplate %s: %s", wftmpl.Name, err)
						invalid = true
					}
				}
			}

			if invalid {
				log.Fatalf("Errors encountered in validation")
			}
			fmt.Printf("ClusterWorkflowTemplate manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict validation")
	command.Flags().BoolVar(&offline, "offline", false,
		"lint template references against local files instead of remote server state")
	return command
}
