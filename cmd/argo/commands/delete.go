package commands

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var (
		listArgs      listFlags
		all           bool
		allNamespaces bool
		dryRun        bool
	)
	var command = &cobra.Command{
		Use:   "delete [--dry-run] [WORKFLOW...|[--all] [--older] [--completed] [--prefix PREFIX] [--selector SELECTOR]]",
		Short: "delete workflows",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			if !allNamespaces {
				listArgs.namespace = client.Namespace()
			}
			var workflows wfv1.Workflows
			for _, name := range args {
				workflows = append(workflows, wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: listArgs.namespace},
				})
			}
			if all || !listArgs.IsZero() {
				listed, err := listWorkflows(ctx, serviceClient, listArgs)
				errors.CheckError(err)
				workflows = append(workflows, listed...)
			}
			for _, md := range workflows {
				if !dryRun {
					_, err := serviceClient.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Name: md.Name, Namespace: md.Namespace})
					if err != nil && apierr.IsNotFound(err) {
						fmt.Printf("Workflow '%s' not found\n", md.Name)
						continue
					}
					errors.CheckError(err)
				}
				fmt.Printf("Workflow '%s' deleted\n", md.Name)
			}
		},
	}

	command.Flags().BoolVar(&allNamespaces, "all-namespaces", false, "Delete workflows from all namespaces")
	command.Flags().BoolVar(&all, "all", false, "Delete all workflows")
	command.Flags().BoolVar(&listArgs.completed, "completed", false, "Delete completed workflows")
	command.Flags().StringVar(&listArgs.prefix, "prefix", "", "Delete workflows by prefix")
	command.Flags().StringVar(&listArgs.finisheAfter, "older", "", "Delete completed workflows finished before the specified duration (e.g. 10m, 3h, 1d)")
	command.Flags().StringVarP(&listArgs.labels, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Do not delete the workflow, only print what would happen")
	return command
}
