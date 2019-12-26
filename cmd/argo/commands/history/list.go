package history

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowhistory"
)

func NewListCommand() *cobra.Command {
	var server string
	var command = &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			conn := client.GetClientConn(server)
			ctx := client.ContextWithAuthorization()
			wfHistoryClient := workflowhistory.NewWorkflowHistoryServiceClient(conn)
			for c := "0"; c != ""; {
				resp, err := wfHistoryClient.ListWorkflowHistory(ctx, &workflowhistory.WorkflowHistoryListRequest{
					ListOptions: &metav1.ListOptions{Continue: c, Limit: 100},
				})
				if err != nil {
					log.Fatal(err)
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				_, _ = fmt.Fprintln(w, "NAMESPACE", "NAME", "UID")
				for _, item := range resp.Items {
					_, _ = fmt.Fprintln(w, item.Namespace, item.Name, item.UID)
				}
				_ = w.Flush()
				fmt.Printf("%v results", len(resp.Items))
				c = resp.Continue
			}
		},
	}
	command.Flags().StringVar(&server, "server", "localhost:2746", "Server")
	return command
}
