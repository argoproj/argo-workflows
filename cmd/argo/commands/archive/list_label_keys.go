package archive

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
)

func NewListLabelKeyCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "list-label-keys",
		Short: "list workflows label keys in the archive",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			keys, err := serviceClient.ListArchivedWorkflowLabelKeys(ctx, &workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest{})
			errors.CheckError(err)
			for _, str := range keys.Items {
				fmt.Printf("%s\n", str)
			}
		},
	}
	return command
}
