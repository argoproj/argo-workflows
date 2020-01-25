package artifact

import (
	"os"

	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {

	var command = &cobra.Command{
		Use:   "list WORKFLOW",
		Short: "List a workfow's artifact",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			workflowName := args[0]
			
			conn := client.GetClientConn()
            ctx := client.GetContext()
            client := workflowarchive.NewArchivedWorkflowServiceClient(conn)
            resp, err := client.ListArchivedWorkflows(ctx, &workflowarchive.ListArchivedWorkflowsRequest{
                ListOptions: &metav1.ListOptions{FieldSelector: "metadata.namespace=" + namespace},
            })
            if err != nil {
                log.Fatal(err)
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
		},
	}

	return command
}
