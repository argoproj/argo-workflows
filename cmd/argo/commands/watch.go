package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/packer"
)

func NewWatchCommand() *cobra.Command {
	var (
		getArgs getFlags
	)

	var command = &cobra.Command{
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(workflow),
			ResourceVersion: "0",
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	errors.CheckError(err)

	wfChan := make(chan *wfv1.Workflow)
	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				log.Debug("Re-establishing workflow watch")
				stream, err = serviceClient.WatchWorkflows(ctx, req)
				errors.CheckError(err)
				continue
			}
			errors.CheckError(err)
			if event == nil {
				continue
			}
			wfChan <- event.Object
		}
	}()

	var wf *wfv1.Workflow
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case newWf := <-wfChan:
			// If we get a new event, update our workflow
			if newWf == nil {
				return
			}
			wf = newWf
		case <-ticker.C:
			// If we don't, refresh the workflow screen every second
		}

		printWorkflowStatus(wf, getArgs)
		if wf != nil && !wf.Status.FinishedAt.IsZero() {
			return
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
