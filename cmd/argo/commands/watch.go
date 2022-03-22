package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
)

func NewWatchCommand() *cobra.Command {
	var getArgs common.GetFlags

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
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			common.WatchWorkflow(ctx, serviceClient, namespace, args[0], getArgs)
		},
	}
	command.Flags().StringVar(&getArgs.Status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error)")
	command.Flags().StringVar(&getArgs.NodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	return command
}

// func watchWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflow string, getArgs common.GetFlags) {
// 	req := &workflowpkg.WatchWorkflowsRequest{
// 		Namespace: namespace,
// 		ListOptions: &metav1.ListOptions{
// 			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(workflow),
// 			ResourceVersion: "0",
// 		},
// 	}
// 	stream, err := serviceClient.WatchWorkflows(ctx, req)
// 	errors.CheckError(err)

// 	wfChan := make(chan *wfv1.Workflow)
// 	go func() {
// 		for {
// 			event, err := stream.Recv()
// 			if err == io.EOF {
// 				log.Debug("Re-establishing workflow watch")
// 				stream, err = serviceClient.WatchWorkflows(ctx, req)
// 				errors.CheckError(err)
// 				continue
// 			}
// 			errors.CheckError(err)
// 			if event == nil {
// 				continue
// 			}
// 			wfChan <- event.Object
// 		}
// 	}()

// 	var wf *wfv1.Workflow
// 	ticker := time.NewTicker(time.Second)
// 	for {
// 		select {
// 		case newWf := <-wfChan:
// 			// If we get a new event, update our workflow
// 			if newWf == nil {
// 				return
// 			}
// 			wf = newWf
// 		case <-ticker.C:
// 			// If we don't, refresh the workflow screen every second
// 		case <-ctx.Done():
// 			// When the context gets canceled
// 			return
// 		}

// 		printWorkflowStatus(wf, getArgs)
// 		if wf != nil && !wf.Status.FinishedAt.IsZero() {
// 			return
// 		}
// 	}
// }

// func printWorkflowStatus(wf *wfv1.Workflow, getArgs common.GetFlags) {
// 	if wf == nil {
// 		return
// 	}
// 	err := packer.DecompressWorkflow(wf)
// 	errors.CheckError(err)
// 	print("\033[H\033[2J")
// 	print("\033[0;0H")
// 	fmt.Print(printWorkflowHelper(wf, getArgs))
// }
