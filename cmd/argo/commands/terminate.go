package commands

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

func NewTerminateCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "terminate WORKFLOW WORKFLOW2...",
		Short: "terminate zero or more workflows",
		Example: `# Terminate a workflow:

  argo terminate my-wf

# Terminate the latest workflow:
  argo terminate @latest
`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {

			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			for _, name := range args {
				wf, err := serviceClient.TerminateWorkflow(ctx, &workflowpkg.WorkflowTerminateRequest{
					Name:      name,
					Namespace: namespace,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s terminated\n", wf.Name)
			}
		},
	}
	return command
}
