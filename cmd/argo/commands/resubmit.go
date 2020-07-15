package commands

import (
	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

func NewResubmitCommand() *cobra.Command {
	var (
		memoized      bool
		priority      int32
		cliSubmitOpts cliSubmitOpts
	)
	var command = &cobra.Command{
		Use:   "resubmit [WORKFLOW...]",
		Short: "resubmit one or more workflows",
		Example: `# Resubmit a workflow:

  argo resubmit my-wf

# Resubmit and wait for completion:

  argo resubmit --wait my-wf.yaml

# Resubmit and watch until completion:

  argo resubmit --watch my-wf.yaml

# Resubmit and tail logs until completion:

  argo resubmit --log my-wf.yaml

# Resubmit the latest workflow:

  argo resubmit @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.priority = &priority
			}

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			for _, name := range args {
				created, err := serviceClient.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{
					Namespace: namespace,
					Name:      name,
					Memoized:  memoized,
				})
				errors.CheckError(err)
				printWorkflow(created, getFlags{output: cliSubmitOpts.output})
				waitWatchOrLog(ctx, serviceClient, namespace, []string{created.Name}, cliSubmitOpts)
			}
		},
	}

	command.Flags().Int32Var(&priority, "priority", 0, "workflow priority")
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&cliSubmitOpts.log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&memoized, "memoized", false, "re-use successful steps & outputs from the previous run (experimental)")
	return command
}
