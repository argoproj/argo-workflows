package archive

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
)

func NewListCommand() *cobra.Command {
	var (
		output string
	)
	var command = &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			namespace := client.Namespace()
			resp, err := serviceClient.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{
				ListOptions: &metav1.ListOptions{
					FieldSelector: "metadata.namespace=" + namespace,
				},
			})
			errors.CheckError(err)
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
