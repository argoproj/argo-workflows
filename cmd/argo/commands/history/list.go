package history

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowhistory"
	"github.com/argoproj/argo/workflow/packer"
)

func NewListCommand() *cobra.Command {
	var (
		output string
	)
	var command = &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			conn := client.GetClientConn()
			ctx := client.ContextWithAuthorization()
			wfHistoryClient := workflowhistory.NewWorkflowHistoryServiceClient(conn)
			resp, err := wfHistoryClient.ListWorkflowHistory(ctx, &workflowhistory.WorkflowHistoryListRequest{
				ListOptions: &metav1.ListOptions{},
			})
			if err != nil {
				log.Fatal(err)
			}
			for _, wf := range resp.Items {
				err = packer.DecompressWorkflow(&wf)
				if err != nil {
					log.Fatal(err)
				}
			}
			switch output {
			case "json":
				output, err := json.Marshal(resp.Items)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(output))
			case "yaml":
				output, err := yaml.Marshal(resp.Items)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(output))
			default:
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				_, _ = fmt.Fprintln(w, "NAMESPACE", "NAME", "UID")
				for _, item := range resp.Items {
					_, _ = fmt.Fprintln(w, item.Namespace, item.Name, item.UID)
				}
				_ = w.Flush()
			}
		},
	}
	command.Flags().StringVarP(&output, "output", "o", "wide", "Output format. One of: json|yaml|wide")
	return command
}
