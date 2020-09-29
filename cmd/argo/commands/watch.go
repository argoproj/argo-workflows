package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
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
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			return watchWorkflow(ctx, serviceClient, namespace, args[0], getArgs)
		},
	}
	command.Flags().StringVar(&getArgs.status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error)")
	command.Flags().StringVar(&getArgs.nodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	return command
}

func watchWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflow string, getArgs getFlags) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector: util.GenerateFieldSelectorFromWorkflowName(workflow),
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	if err != nil {
		return err
	}

	wfChan := make(chan *wfv1.Workflow)
	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				log.Debug("Re-establishing workflow watch")
				stream, err = serviceClient.WatchWorkflows(ctx, req)
				if err != nil {
					log.Errorf("failed to watch workflow. %v", err)
					return
				}
				continue
			}
			if err != nil {
				log.Errorf("failed to watch workflow. %v", err)
				return
			}
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
				return nil
			}
			wf = newWf
		case <-ticker.C:
			// If we don't, refresh the workflow screen every second
		}

		err := printWorkflowStatus(wf, getArgs)
		if err != nil {
			return err
		}
		if wf != nil && !wf.Status.FinishedAt.IsZero() {
			return nil
		}
	}
}

func printWorkflowStatus(wf *wfv1.Workflow, getArgs getFlags) error {
	if wf == nil {
		return nil
	}
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		return err
	}
	print("\033[H\033[2J")
	print("\033[0;0H")
	fmt.Print(printWorkflowHelper(wf, getArgs))
	return nil
}
