package template

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
)

func NewGetCommand() *cobra.Command {
	var output string

	command := &cobra.Command{
		Use:   "get WORKFLOW_TEMPLATE...",
		Short: "display details about a workflow template",
		Example: `# Display details about a workflow template
  argo template get my-wftmpl

# Display details about multiple workflow templates printed as YAML

  argo template get my-wftmpl1 my-wftmpl2 -o yaml
`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
			if err != nil {
				log.Fatal(err)
			}
			namespace := client.Namespace()
			for _, name := range args {
				wftmpl, err := serviceClient.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{
					Name:      name,
					Namespace: namespace,
				})
				if err != nil {
					log.Fatal(err)
				}
				printWorkflowTemplate(wftmpl, output)
			}
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: json|yaml|wide")
	return command
}
