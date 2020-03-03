package commands

import (
	"fmt"
	"time"

	"github.com/argoproj/pkg/errors"
	argotime "github.com/argoproj/pkg/time"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/v2/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/v2/pkg/apiclient/workflow"
	"github.com/argoproj/argo/v2/workflow/common"
)

var (
	completedLabelSelector = fmt.Sprintf("%s=true", common.LabelKeyCompleted)
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var (
		selector  string
		all       bool
		completed bool
		older     string
	)

	var command = &cobra.Command{
		Use: "delete WORKFLOW...",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			var workflowsToDelete []metav1.ObjectMeta
			for _, name := range args {
				workflowsToDelete = append(workflowsToDelete, metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				})
			}
			if all || completed || older != "" {
				// all is effectively the default, completed takes precedence over all
				if completed {
					if selector != "" {
						selector = selector + "," + completedLabelSelector
					} else {
						selector = completedLabelSelector
					}
				}
				// you can mix older with either of these
				var olderTime *time.Time
				if older != "" {
					var err error
					olderTime, err = argotime.ParseSince(older)
					errors.CheckError(err)
				}
				list, err := serviceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
					Namespace:   namespace,
					ListOptions: &metav1.ListOptions{LabelSelector: selector},
				})
				errors.CheckError(err)
				for _, wf := range list.Items {
					if olderTime != nil && (wf.Status.FinishedAt.IsZero() || wf.Status.FinishedAt.After(*olderTime)) {
						continue
					}
					workflowsToDelete = append(workflowsToDelete, wf.ObjectMeta)
				}
			}
			for _, md := range workflowsToDelete {
				_, err := serviceClient.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{
					Name:      md.Name,
					Namespace: md.Namespace,
				})
				errors.CheckError(err)
				fmt.Printf("Workflow '%s' deleted\n", md.Name)
			}
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflows")
	command.Flags().BoolVar(&completed, "completed", false, "Delete completed workflows")
	command.Flags().StringVar(&older, "older", "", "Delete completed workflows older than the specified duration (e.g. 10m, 3h, 1d)")
	command.Flags().StringVarP(&selector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones")
	return command
}
