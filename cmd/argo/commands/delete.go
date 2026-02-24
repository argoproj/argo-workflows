package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var (
		flags         listFlags
		all           bool
		allNamespaces bool
		dryRun        bool
		force         bool
		hasFilterFlag = func() bool {
			return all || allNamespaces || flags.completed || flags.resubmitted || flags.prefix != "" ||
				flags.labels != "" || flags.fields != "" || flags.finishedBefore != "" || len(flags.status) > 0
		}
	)
	command := &cobra.Command{
		Use:   "delete [--dry-run] [WORKFLOW...|[--all] [--older] [--completed] [--resubmitted] [--prefix PREFIX] [--selector SELECTOR] [--force] [--status STATUS] ]",
		Short: "delete workflows",
		Example: `# Delete a workflow:

  argo delete my-wf

# Delete the latest workflow:

  argo delete @latest
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasFilterFlag() {
				return errors.New("requires either a workflow or other argument")
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
			var workflows wfv1.Workflows
			if !allNamespaces {
				flags.namespace = client.Namespace(ctx)
			}
			for _, name := range args {
				workflows = append(workflows, wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: flags.namespace},
				})
			}
			if hasFilterFlag() {
				listed, err := listWorkflows(ctx, serviceClient, flags)
				if err != nil {
					return err
				}
				workflows = append(workflows, listed...)
			}

			if len(workflows) == 0 {
				fmt.Printf("No resources found\n")
				return nil
			}

			for _, wf := range workflows {
				if dryRun {
					fmt.Printf("Workflow '%s' deleted (dry-run)\n", wf.Name)
					continue
				}

				_, err := serviceClient.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Name: wf.Name, Namespace: wf.Namespace, Force: force})
				if err != nil {
					if status.Code(err) == codes.NotFound {
						fmt.Printf("Workflow '%s' not found\n", wf.Name)
						continue
					} else {
						return err
					}
				}
				fmt.Printf("Workflow '%s' deleted\n", wf.Name)
			}

			return nil
		},
	}

	command.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Delete workflows from all namespaces")
	command.Flags().BoolVar(&all, "all", false, "Delete all workflows")
	command.Flags().BoolVar(&flags.completed, "completed", false, "Delete completed workflows")
	command.Flags().BoolVar(&flags.resubmitted, "resubmitted", false, "Delete resubmitted workflows")
	command.Flags().StringVar(&flags.prefix, "prefix", "", "Delete workflows by prefix")
	command.Flags().StringVarP(&flags.labels, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&flags.fields, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().StringVar(&flags.finishedBefore, "older", "", "Delete completed workflows finished before the specified duration (e.g. 10m, 3h, 1d)")
	command.Flags().StringSliceVar(&flags.status, "status", []string{}, "Delete by status (comma separated)")
	command.Flags().Int64VarP(&flags.chunkSize, "query-chunk-size", "", 0, "Run the list query in chunks (deletes will still be executed individually)")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Do not delete the workflow, only print what would happen")
	command.Flags().BoolVar(&force, "force", false, "Force delete workflows by removing finalizers")
	return command
}
