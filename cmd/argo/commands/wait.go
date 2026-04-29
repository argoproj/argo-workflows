package commands

import (
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, apiClient, err := client.NewAPIClient(ctx)
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx)
			namespace := client.Namespace(ctx)
			common.WaitWorkflows(ctx, serviceClient, namespace, args, ignoreNotFound, false)
			return nil
		},
	}
	command.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", false, "Ignore the wait if the workflow is not found")
	return command
}
