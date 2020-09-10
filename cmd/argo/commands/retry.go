package commands

import (
	"fmt"
	"log"
	
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
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
		Example: `# Retry a workflow:

  argo retry my-wf

# Retry several workflows: 

  argo retry my-wf my-other-wf my-third-wf

# Retry and wait for completion:

  argo retry --wait my-wf.yaml

# Retry and watch until completion:

  argo retry --watch my-wf.yaml

# Retry and tail logs until completion:

  argo retry --log my-wf.yaml

# Retry the latest workflow:

  argo retry @latest
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(retryOps.nodeFieldSelector)
			if err != nil {
				return fmt.Errorf("unable to parse node field selector '%s': %s", retryOps.nodeFieldSelector, err)
			}

			for _, name := range args {
				wf, err := serviceClient.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{
					Name:              name,
					Namespace:         namespace,
					RestartSuccessful: retryOps.restartSuccessful,
					NodeFieldSelector: selector.String(),
				})
				if err != nil {
					return err
				}
				printWorkflow(wf, getFlags{output: cliSubmitOpts.output})
				waitWatchOrLog(ctx, serviceClient, namespace, []string{name}, cliSubmitOpts)
			}
			return nil
		},
	}
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&retryOps.restartSuccessful, "restart-successful", false, "indicates to restart successful nodes matching the --node-field-selector")
	command.Flags().StringVar(&retryOps.nodeFieldSelector, "node-field-selector", "", "selector of nodes to reset, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	return command
}
