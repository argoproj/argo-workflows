package commands

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

func NewFailCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "fail WORKFLOW WORKFLOW2...",
		Short: "fail zero or more workflows",
		Run: func(cmd *cobra.Command, args []string) {

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			for _, name := range args {
				wf, err := serviceClient.FailWorkflow(ctx, &workflowpkg.WorkflowFailRequest{
					Name:      name,
					Namespace: namespace,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s failed\n", wf.Name)
			}
		},
	}
	return command
}
