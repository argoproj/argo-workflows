package commands

import (
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
)

func NewWaitCommand() *cobra.Command {
	var ignoreNotFound bool
	command := &cobra.Command{
		Use:   "wait [WORKFLOW...]",
		Short: "waits for workflows to complete",
		Example: `# Wait on a workflow:

  argo wait my-wf

# Wait on the latest workflow:

  argo wait @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			common.WaitWorkflows(ctx, serviceClient, namespace, args, ignoreNotFound, false)
		},
	}
	command.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", false, "Ignore the wait if the workflow is not found")
	return command
}
