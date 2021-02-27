package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/packer"
)

func NewWatchCommand() *cobra.Command {
	var getArgs getFlags

	command := &cobra.Command{
		Use:   "watch WORKFLOW",
		Short: "watch a workflow until it completes",
		Example: `# Watch a workflow:

  argo watch my-wf

# Watch the latest workflow:

  argo watch @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			watchWorkflow(ctx, serviceClient, namespace, args[0], getArgs)
		},
	}
	command.Flags().StringVar(&getArgs.status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error)")
	command.Flags().StringVar(&getArgs.nodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	return command
}

func watchWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflow string, getArgs getFlags) {
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector: util.GenerateFieldSelectorFromWorkflowName(workflow),
		},
	}
	var wf *wfv1.Workflow
	go func() {
		for range time.Tick(time.Second) {
			if wf != nil {
				printWorkflowStatus(wf, getArgs)
			}
		}
	}()
	for {
		stream, err := serviceClient.WatchWorkflows(ctx, req)
		errors.CheckError(err)
		for {
			e, err := stream.Recv()
			errors.CheckError(err)
			wf = e.Object
			if e.Type == string(watch.Deleted) || !wf.Status.FinishedAt.IsZero() {
				return
			}
		}
	}
}

func printWorkflowStatus(wf *wfv1.Workflow, getArgs getFlags) {
	if wf == nil {
		return
	}
	err := packer.DecompressWorkflow(wf)
	errors.CheckError(err)
	print("\033[H\033[2J")
	print("\033[0;0H")
	fmt.Print(printWorkflowHelper(wf, getArgs))
}
