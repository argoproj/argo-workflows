package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
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
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			for _, wfName := range args {
				_, err := serviceClient.SuspendWorkflow(ctx, &workflowpkg.WorkflowSuspendRequest{
					Name:      wfName,
					Namespace: namespace,
				})
				if err != nil {
					return fmt.Errorf("Failed to suspended %s: %+v", wfName, err)
				}
				fmt.Printf("workflow %s suspended\n", wfName)
			}
			return nil
		},
	}
	return command
}
