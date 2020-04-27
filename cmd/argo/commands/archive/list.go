package archive

import (
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo/util/printer"
)

func NewListCommand() *cobra.Command {
	var (
		selector string
		output   string
	)
	var command = &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			namespace := client.Namespace()
			resp, err := serviceClient.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{
				ListOptions: &metav1.ListOptions{
					FieldSelector: "metadata.namespace=" + namespace,
					LabelSelector: selector,
				},
			})
			errors.CheckError(err)
			workflows := resp.Items
			err = printer.PrintWorkflows(workflows, os.Stdout, printer.PrintOpts{
				Output: output,
				AllNamespaces: true,
			})
			errors.CheckError(err)
		},
	}
	command.Flags().StringVarP(&output, "output", "o", "wide", "Output format. One of: json|yaml|wide")
	command.Flags().StringVarP(&selector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones")
	return command
}
