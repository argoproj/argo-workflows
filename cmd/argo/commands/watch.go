package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/v2/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/v2/pkg/apiclient/workflow"

	wfv1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/v2/workflow/packer"
)

func NewWatchCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "watch WORKFLOW",
		Short: "watch a workflow until it completes",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			watchWorkflow(args[0])

		},
	}
	return command
}

func watchWorkflow(wfName string) {

	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewWorkflowServiceClient()
	namespace := client.Namespace()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := serviceClient.WatchWorkflows(ctx, &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector: fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", wfName)).String(),
		},
	})
	errors.CheckError(err)
	for {
		event, err := stream.Recv()
		errors.CheckError(err)
		wf := event.Object
		if wf == nil {
			break
		}
		printWorkflowStatus(wf)
		if !wf.Status.FinishedAt.IsZero() {
			break
		}
	}
}

func printWorkflowStatus(wf *wfv1.Workflow) {
	err := packer.DecompressWorkflow(wf)
	errors.CheckError(err)
	print("\033[H\033[2J")
	print("\033[0;0H")
	printWorkflowHelper(wf, getFlags{})
}
