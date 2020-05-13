package commands

import (
	"log"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type retryOps struct {
	nodeFieldSelector string // --node-field-selector
	restartSuccessful bool   // --restart-successful
}

func NewRetryCommand() *cobra.Command {
	var (
		cliSubmitOpts cliSubmitOpts
		retryOps      retryOps
	)
	var command = &cobra.Command{
		Use:   "retry [WORKFLOW...]",
		Short: "retry zero or more workflows",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(retryOps.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", retryOps.nodeFieldSelector, err)
			}

			for _, name := range args {
				wf, err := serviceClient.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{
					Name:              name,
					Namespace:         namespace,
					RestartSuccessful: retryOps.restartSuccessful,
					NodeFieldSelector: selector.String(),
				})
				if err != nil {
					errors.CheckError(err)
					return
				}
				printWorkflow(wf, getFlags{output: cliSubmitOpts.output})
				waitOrWatch([]string{name}, cliSubmitOpts)
			}
		},
	}
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&retryOps.restartSuccessful, "restart-successful", false, "indicates to restart successful nodes matching the --node-field-selector")
	command.Flags().StringVar(&retryOps.nodeFieldSelector, "node-field-selector", "", "selector of nodes to reset, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	return command
}
