package archive

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
)

func NewListLabelValueCommand() *cobra.Command {
	var (
		selector string
	)
	command := &cobra.Command{
		Use:   "list-label-values",
		Short: "get workflow label values in the archive",
		Example: `# Get workflow label values in the archive:
  argo archive list-label-values -l key1
`,
		Run: func(cmd *cobra.Command, args []string) {
			listOpts := &metav1.ListOptions{
				LabelSelector: selector,
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			labels, err := serviceClient.ListArchivedWorkflowLabelValues(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest{ListOptions: listOpts})
			errors.CheckError(err)

			for _, str := range labels.Items {
				fmt.Printf("%s\n", str)
			}
		},
	}
	command.Flags().StringVarP(&selector, "selector", "l", "", "Selector (label query) to query on, allows 1 value (e.g. -l key1)")
	err := command.MarkFlagRequired("selector")
	errors.CheckError(err)
	return command
}
