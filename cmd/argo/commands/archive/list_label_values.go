package archive

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v4/util/errors"
)

func NewListLabelValueCommand() *cobra.Command {
	var (
		selector string
	)
	command := &cobra.Command{
		Use:   "list-label-values",
		Short: "get workflow label values in the archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			listOpts := &metav1.ListOptions{
				LabelSelector: selector,
			}

			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}
			labels, err := serviceClient.ListArchivedWorkflowLabelValues(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest{ListOptions: listOpts})
			if err != nil {
				return err
			}

			for _, str := range labels.Items {
				fmt.Printf("%s\n", str)
			}

			return nil
		},
	}
	ctx := command.Context()
	command.Flags().StringVarP(&selector, "selector", "l", "", "Selector (label query) to query on, allows 1 value (e.g. -l key1)")
	err := command.MarkFlagRequired("selector")
	errors.CheckError(ctx, err)
	return command
}
