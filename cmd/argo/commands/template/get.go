package template

import (
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
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
			if err != nil {
				return err
			}
			namespace := client.Namespace()
			for _, name := range args {
				wftmpl, err := serviceClient.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{
					Name:      name,
					Namespace: namespace,
				})
				if err != nil {
					return err
				}
				printWorkflowTemplate(wftmpl, output.String())
			}
			return nil
		},
	}

	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
	return command
}
