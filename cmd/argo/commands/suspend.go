package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

func NewSuspendCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "suspend WORKFLOW1 WORKFLOW2...",
		Short: "suspend zero or more workflows (opposite of resume)",
		Example: `# Suspend a workflow:

  argo suspend my-wf

# Suspend the latest workflow:
  argo suspend @latest
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, apiClient, err := client.NewAPIClient(ctx)
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx)
			namespace := client.Namespace(ctx)
			for _, wfName := range args {
				_, err := serviceClient.SuspendWorkflow(ctx, &workflowpkg.WorkflowSuspendRequest{
					Name:      wfName,
					Namespace: namespace,
				})
				if err != nil {
					return fmt.Errorf("failed to suspend %s: %w", wfName, err)
				}
				fmt.Printf("workflow %s suspended\n", wfName)
			}
			return nil
		},
	}
	return command
}
