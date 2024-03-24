package clustertemplate

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
)

func NewGetCommand() *cobra.Command {
	var output string

	command := &cobra.Command{
		Use:   "get CLUSTER WORKFLOW_TEMPLATE...",
		Short: "display details about a cluster workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
			if err != nil {
				log.Fatal(err)
			}
			for _, name := range args {
				wftmpl, err := serviceClient.GetClusterWorkflowTemplate(ctx, &clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest{
					Name: name,
				})
				if err != nil {
					log.Fatal(err)
				}
				printClusterWorkflowTemplate(wftmpl, output)
			}
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: json|yaml|wide")
	return command
}
