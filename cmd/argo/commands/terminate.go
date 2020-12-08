package commands

import (
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type terminateOption struct {
	namespace string
	labels    string
	fields    string
	dryRun    bool
}

func (t *terminateOption) isList() bool {
	if t.labels != "" || t.fields != "" {
		return true
	}

	return false
}

func (t *terminateOption) convertToWorkflows(names []string) wfv1.Workflows {
	var wfs wfv1.Workflows

	for _, n := range names {
		wfs = append(wfs, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: n, Namespace: t.namespace},
		})
	}

	return wfs
}

func NewTerminateCommand() *cobra.Command {
	t := &terminateOption{}

	var command = &cobra.Command{
		Use:   "terminate WORKFLOW WORKFLOW2...",
		Short: "terminate zero or more workflows immediately",
		Example: `# Terminate a workflow:

  argo terminate my-wf

# Terminate the latest workflow:
  argo terminate @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !t.isList() {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			t.namespace = client.Namespace()

			var workflows wfv1.Workflows

			if t.isList() {
				listed, err := listWorkflows(ctx, serviceClient, listFlags{
					namespace: t.namespace,
					fields:    t.fields,
					labels:    t.labels,
				})
				errors.CheckError(err)
				workflows = append(workflows, listed...)
			} else {
				workflows = t.convertToWorkflows(args)
			}

			for _, w := range workflows {
				if t.dryRun {
					fmt.Printf("workflow %s terminated (dry-run)\n", w.Name)
					continue
				}

				wf, err := serviceClient.TerminateWorkflow(ctx, &workflowpkg.WorkflowTerminateRequest{
					Name:      w.Name,
					Namespace: w.Namespace,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s terminated\n", wf.Name)
			}
		},
	}

	command.Flags().StringVarP(&t.labels, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones")
	command.Flags().StringVar(&t.fields, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selectorkey1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().BoolVar(&t.dryRun, "dry-run", false, "Do not terminate the workflow, only print what would happen")
	return command
}
