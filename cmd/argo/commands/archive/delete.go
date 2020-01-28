package archive

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/server/workflowarchive"
)

func NewDeleteCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "delete UID...",
		Run: func(cmd *cobra.Command, args []string) {
			for _, uid := range args {
				conn := client.GetClientConn()
				ctx := client.GetContext()
				client := workflowarchive.NewArchivedWorkflowServiceClient(conn)
				_, err := client.DeleteArchivedWorkflow(ctx, &workflowarchive.DeleteArchivedWorkflowRequest{
					Uid: uid,
				})
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("Archived workflow '%s' deleted\n", uid)
			}
		},
	}
	return command
}
