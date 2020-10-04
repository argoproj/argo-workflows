package commands

import (
	"fmt"
	"log"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type stopOps struct {
	message           string // --message
	nodeFieldSelector string // --node-field-selector
	labelSelector     string // --selector
	fieldSelector     string // --field-selector
	dryRun            bool   // --dry-run
}

func NewStopCommand() *cobra.Command {
	var (
		stopArgs stopOps
	)

	var command = &cobra.Command{
		Use:   "stop WORKFLOW WORKFLOW2...",
		Short: "stop zero or more workflows",
		Example: `# Stop a workflow:

  argo stop my-wf

# Stop the latest workflow:
  argo stop @latest
`,
		Run: func(cmd *cobra.Command, args []string) {

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(stopArgs.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", stopArgs.nodeFieldSelector, err)
			}

			var names []string

			if stopArgs.labelSelector != "" || stopArgs.fieldSelector != "" {
				listed, err := listWorkflows(ctx, serviceClient, listFlags{
					namespace: namespace,
					labels:    stopArgs.labelSelector,
					fields:    stopArgs.fieldSelector,
				})
				errors.CheckError(err)

				for _, w := range listed {
					names = append(names, w.GetName())
				}
			}

			names = append(names, args...)
			for _, name := range names {
				if stopArgs.dryRun {
					fmt.Printf("workflow %s stopped (dry-run)\n", name)
					continue
				}

				wf, err := serviceClient.StopWorkflow(ctx, &workflowpkg.WorkflowStopRequest{
					Name:              name,
					Namespace:         namespace,
					NodeFieldSelector: selector.String(),
					Message:           stopArgs.message,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s stopped\n", wf.Name)
			}
		},
	}
	command.Flags().StringVar(&stopArgs.message, "message", "", "Message to add to previously running nodes")
	command.Flags().StringVar(&stopArgs.nodeFieldSelector, "node-field-selector", "", "selector of node to stop, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	command.Flags().StringVarP(&stopArgs.labelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&stopArgs.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selectorkey1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().BoolVar(&stopArgs.dryRun, "dry-run", false, "Do not delete the workflow, only print what would happen")
	return command
}
