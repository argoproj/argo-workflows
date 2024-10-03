package template

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
)

func NewGetCommand() *cobra.Command {
	var output = common.NewPrintWorkflowOutputValue("")

	command := &cobra.Command{
		Use:   "get WORKFLOW_TEMPLATE...",
		Short: "display details about a workflow template",
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
				printWorkflowTemplate(wftmpl, output.String())
			}
		},
	}

	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
	return command
}
