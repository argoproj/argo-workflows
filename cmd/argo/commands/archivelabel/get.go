package archivelabel

import (
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivelabelpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchivelabel"
)

func NewGetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "get labelkey",
		Short: "get workflow label key=value in the archive",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			key := args[0]

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewArchivedWorkflowLabelServiceClient()
			errors.CheckError(err)
			labels, err := serviceClient.GetArchivedWorkflowLabel(ctx, &workflowarchivelabelpkg.GetArchivedWorkflowLabelRequest{Key: key})
			errors.CheckError(err)

			for _, str := range labels.Items {
				fmt.Printf("%s=%s\n", key, str)
			}
		},
	}
	return command
}
