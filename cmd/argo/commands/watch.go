package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
)

func NewWatchCommand() *cobra.Command {
	var getArgs common.GetFlags

	command := &cobra.Command{
		Use:   "watch WORKFLOW",
		Short: "watch a workflow until it completes",
		Example: `# Watch a workflow:

  argo watch my-wf

# Watch the latest workflow:

  argo watch @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			common.WatchWorkflow(ctx, serviceClient, namespace, args[0], getArgs)
		},
	}
	command.Flags().StringVar(&getArgs.Status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error)")
	command.Flags().StringVar(&getArgs.NodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	return command
}
