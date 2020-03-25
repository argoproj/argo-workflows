package commands

import (
	"fmt"
	"k8s.io/apimachinery/pkg/fields"
	"log"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type stopOps struct {
	message      string // --message
	nodeSelector string // --node-selector
}

func NewStopCommand() *cobra.Command {
	var (
		stopArgs stopOps
	)

	var command = &cobra.Command{
		Use:   "stop WORKFLOW WORKFLOW2...",
		Short: "stop zero or more workflows",
		Run: func(cmd *cobra.Command, args []string) {

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(stopArgs.nodeSelector)
			if err != nil {
				log.Fatalf("Unable to parse node selector '%s': %s", stopArgs.nodeSelector, err)
			}

			for _, name := range args {
				wf, err := serviceClient.StopWorkflow(ctx, &workflowpkg.WorkflowStopRequest{
					Name:      name,
					Namespace: namespace,
					NodeSelector: selector.String(),
					Message: stopArgs.message,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s stopped\n", wf.Name)
			}
		},
	}
	command.Flags().StringVar(&stopArgs.message, "message", "", "Message to add to previously running nodes")
	command.Flags().StringVar(&stopArgs.nodeSelector, "node-selector", "", "selector of node to stop, eg: --node-selector inputs.paramaters.myparam.value=abc")
	return command
}
