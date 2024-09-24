package archive

import (
	"fmt"

	"github.com/spf13/cobra"

	client "github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
)

func NewDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "delete UID...",
		Short: "delete a workflow in the archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}
			for _, uid := range args {
				if _, err = serviceClient.DeleteArchivedWorkflow(ctx, &workflowarchivepkg.DeleteArchivedWorkflowRequest{Uid: uid}); err != nil {
					return err
				}
				fmt.Printf("Archived workflow '%s' deleted\n", uid)
			}
			return nil
		},
	}
	return command
}
