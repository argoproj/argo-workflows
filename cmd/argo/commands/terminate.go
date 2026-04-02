package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
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

	command := &cobra.Command{
		Use:   "terminate WORKFLOW WORKFLOW2...",
		Short: "terminate zero or more workflows immediately",
		Long:  "Immediately stop a workflow and do not run any exit handlers.",
		Example: `# Terminate a workflow:

  argo terminate my-wf

# Terminate the latest workflow:

  argo terminate @latest

# Terminate multiple workflows by label selector

  argo terminate -l workflows.argoproj.io/test=true

# Terminate multiple workflows by field selector

  argo terminate --field-selector metadata.namespace=argo
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !t.isList() {
				return errors.New("requires either selector or workflow")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, apiClient, err := client.NewAPIClient(ctx)
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx)
			t.namespace = client.Namespace(ctx)

			var workflows wfv1.Workflows

			if t.isList() {
				listed, err := listWorkflows(ctx, serviceClient, listFlags{
					namespace: t.namespace,
					fields:    t.fields,
					labels:    t.labels,
				})
				if err != nil {
					return err
				}
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
				if err != nil {
					return err
				}
				fmt.Printf("workflow %s terminated\n", wf.Name)
			}
			return nil
		},
	}

	command.Flags().StringVarP(&t.labels, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&t.fields, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().BoolVar(&t.dryRun, "dry-run", false, "Do not terminate the workflow, only print what would happen")
	return command
}
