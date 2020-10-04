package commands

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type terminateOps struct {
	labelSelector string // --selector
	fieldSelector string // --field-selector
}

func NewTerminateCommand() *cobra.Command {
	var (
		terminateArgs terminateOps
	)

	var command = &cobra.Command{
		Use:   "terminate WORKFLOW WORKFLOW2...",
		Short: "terminate zero or more workflows",
		Example: `# Terminate a workflow:

  argo terminate my-wf

# Terminate the latest workflow:
  argo terminate @latest
`,
		Run: func(cmd *cobra.Command, args []string) {

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			var names []string

			if terminateArgs.labelSelector != "" || terminateArgs.fieldSelector != "" {
				listed, err := listWorkflows(ctx, serviceClient, listFlags{
					namespace: namespace,
					labels:    terminateArgs.labelSelector,
					fields:    terminateArgs.fieldSelector,
				})
				errors.CheckError(err)

				for _, w := range listed {
					names = append(names, w.GetName())
				}
			}

			names = append(names, args...)
			for _, name := range names {
				wf, err := serviceClient.TerminateWorkflow(ctx, &workflowpkg.WorkflowTerminateRequest{
					Name:      name,
					Namespace: namespace,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s terminated\n", wf.Name)
			}
		},
	}

	command.Flags().StringVarP(&terminateArgs.labelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&terminateArgs.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selectorkey1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	return command
}
