package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

func NewSuspendCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "suspend WORKFLOW1 WORKFLOW2...",
		Short: "suspend zero or more workflow",
		Run: func(cmd *cobra.Command, args []string) {
			apiClient := CLIOpt.client
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			for _, wfName := range args {
				_, err := serviceClient.SuspendWorkflow(CLIOpt.ctx, &workflowpkg.WorkflowSuspendRequest{
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
