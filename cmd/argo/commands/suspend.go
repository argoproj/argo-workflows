package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

func NewSuspendCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "suspend WORKFLOW1 WORKFLOW2...",
		Short: "suspend a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			for _, wfName := range args {
				_, err := serviceClient.SuspendWorkflow(ctx, &workflowpkg.WorkflowSuspendRequest{
					Name:      wfName,
					Namespace: namespace,
				})
				if err != nil {
					log.Fatalf("Failed to suspended %s: %+v", wfName, err)
				}
				fmt.Printf("workflow %s suspended\n", wfName)
			}
		},
	}
	return command
}
