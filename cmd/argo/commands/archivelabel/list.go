package archivelabel

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivelabelpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchivelabel"
)

func NewListCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "list",
		Short: "list workflows label keys in the archive",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewArchivedWorkflowLabelServiceClient()
			errors.CheckError(err)
			keys, err := serviceClient.ListArchivedWorkflowLabel(ctx, &workflowarchivelabelpkg.ListArchivedWorkflowLabelRequest{})
			errors.CheckError(err)
			for _, str := range keys.Items {
				fmt.Printf("%s\n", str)
			}
		},
	}
	return command
}
