package archive

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	client "github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
)

func NewDeleteCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "delete UID...",
		Short: "delete a workflow in the archive",
		Example: `# Delete a workflow in the archive by its UID:
  argo archive delete abc123-def456-ghi789-jkl012
`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			for _, uid := range args {
				_, err = serviceClient.DeleteArchivedWorkflow(ctx, &workflowarchivepkg.DeleteArchivedWorkflowRequest{Uid: uid})
				errors.CheckError(err)
				fmt.Printf("Archived workflow '%s' deleted\n", uid)
			}
		},
	}
	return command
}
