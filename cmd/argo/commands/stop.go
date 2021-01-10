package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type stopOps struct {
	namespace         string
	labels            string
	fields            string
	message           string // --message
	nodeFieldSelector string // --node-field-selector
	dryRun            bool
}

func (o *stopOps) isList() bool {
	if o.labels != "" || o.fields != "" {
		return true
	}

	return false
}

func (o *stopOps) convertToWorkflows(names []string) wfv1.Workflows {
	var wfs wfv1.Workflows

	for _, n := range names {
		wfs = append(wfs, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: n, Namespace: o.namespace},
		})
	}

	return wfs
}

func NewStopCommand() *cobra.Command {
	o := &stopOps{}

	var command = &cobra.Command{
		Use:   "stop WORKFLOW WORKFLOW2...",
		Short: "stop zero or more workflows allowing all exit handlers to run",
		Example: `# Stop a workflow:

  argo stop my-wf

# Stop the latest workflow:
  argo stop @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !o.isList() {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			o.namespace = client.Namespace()

			selector, err := fields.ParseSelector(o.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", o.nodeFieldSelector, err)
			}

			var workflows wfv1.Workflows

			if o.isList() {
				listed, err := listWorkflows(ctx, serviceClient, listFlags{
					namespace: o.namespace,
					fields:    o.fields,
					labels:    o.labels,
				})
				errors.CheckError(err)
				workflows = append(workflows, listed...)
			} else {
				workflows = o.convertToWorkflows(args)
			}

			for _, w := range workflows {
				if o.dryRun {
					fmt.Printf("workflow %s stopped (dry-run)\n", w.Name)
					continue
				}

				wf, err := serviceClient.StopWorkflow(ctx, &workflowpkg.WorkflowStopRequest{
					Name:              w.Name,
					Namespace:         w.Namespace,
					NodeFieldSelector: selector.String(),
					Message:           o.message,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s stopped\n", wf.Name)
			}
		},
	}
	command.Flags().StringVarP(&o.labels, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones")
	command.Flags().StringVar(&o.fields, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selectorkey1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().StringVar(&o.message, "message", "", "Message to add to previously running nodes")
	command.Flags().StringVar(&o.nodeFieldSelector, "node-field-selector", "", "selector of node to stop, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	command.Flags().BoolVar(&o.dryRun, "dry-run", false, "Do not stop the workflow, only print what would happen")
	return command
}
