package archive

import (
	"fmt"

	"github.com/spf13/cobra"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
)

func NewDeleteCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:          "delete UID...",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}
			for _, uid := range args {
				_, err = serviceClient.DeleteArchivedWorkflow(ctx, &workflowarchivepkg.DeleteArchivedWorkflowRequest{Uid: uid})
				if err != nil {
					return err
				}
				fmt.Printf("Archived workflow '%s' deleted\n", uid)
			}
			return nil
		},
	}
	return command
}
