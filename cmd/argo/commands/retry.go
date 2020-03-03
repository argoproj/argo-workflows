package commands

import (
	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/v2/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/v2/pkg/apiclient/workflow"
)

func NewRetryCommand() *cobra.Command {
	var (
		cliSubmitOpts cliSubmitOpts
	)
	var command = &cobra.Command{
		Use:   "retry [WORKFLOW...]",
		Short: "retry zero or more workflows",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			for _, name := range args {
				wf, err := serviceClient.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{
					Name:      name,
					Namespace: namespace,
				})
				if err != nil {
					errors.CheckError(err)
					return
				}
				printWorkflow(wf, cliSubmitOpts.output, DefaultStatus)
				waitOrWatch([]string{name}, cliSubmitOpts)
			}
		},
	}
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	return command
}
