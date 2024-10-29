package archive

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
)

func NewListLabelKeyCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "list-label-keys",
		Short: "list workflows label keys in the archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}
			keys, err := serviceClient.ListArchivedWorkflowLabelKeys(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest{})
			if err != nil {
				return err
			}
			for _, str := range keys.Items {
				fmt.Printf("%s\n", str)
			}
			return nil
		},
	}
	return command
}
