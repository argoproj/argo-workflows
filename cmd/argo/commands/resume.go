package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type resumeOps struct {
	nodeFieldSelector string // --node-field-selector
}

func NewResumeCommand() *cobra.Command {
	var (
		resumeArgs resumeOps
	)

	var command = &cobra.Command{
		Use:   "resume WORKFLOW1 WORKFLOW2...",
		Short: "resume zero or more workflows",
		Run: func(cmd *cobra.Command, args []string) {
			apiClient := CLIOpt.client
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(resumeArgs.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", resumeArgs.nodeFieldSelector, err)
			}

			for _, wfName := range args {
				_, err := serviceClient.ResumeWorkflow(CLIOpt.ctx, &workflowpkg.WorkflowResumeRequest{
					Name:              wfName,
					Namespace:         namespace,
					NodeFieldSelector: selector.String(),
				})
				if err != nil {
					log.Fatalf("Failed to resume %s: %+v", wfName, err)
				}
				fmt.Printf("workflow %s resumed\n", wfName)
			}

		},
	}
	command.Flags().StringVar(&resumeArgs.nodeFieldSelector, "node-field-selector", "", "selector of node to resume, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	return command
}
