package archive

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowarchive"
)

func NewDeleteCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "delete NAMESPACE UID",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			namespace := args[0]
			uid := args[1]
			conn := client.GetClientConn()
			ctx := client.ContextWithAuthorization()
			client := workflowarchive.NewArchivedWorkflowServiceClient(conn)
			_, err := client.DeleteArchivedWorkflow(ctx, &workflowarchive.DeleteArchivedWorkflowRequest{
				Namespace: namespace,
				Uid:       uid,
			})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Archived workflow '%s' deleted\n", uid)
		},
	}
	return command
}
