package history

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowhistory"
)

func NewResubmitCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "resubmit NAMESPACE UID",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			namespace := args[0]
			uid := args[1]
			conn := client.GetClientConn()
			ctx := client.ContextWithAuthorization()
			wfHistoryClient := workflowhistory.NewWorkflowHistoryServiceClient(conn)
			wf, err := wfHistoryClient.ResubmitWorkflowHistory(ctx, &workflowhistory.WorkflowHistoryUpdateRequest{
				Namespace: namespace,
				Uid:       uid,
			})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Workflow history '%s' resubmitted\n", wf.Name)
		},
	}
	return command
}
