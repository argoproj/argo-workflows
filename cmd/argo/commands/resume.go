package commands

import (
	"fmt"
	"k8s.io/apimachinery/pkg/fields"
	"log"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type resumeOps struct {
	nodeSelector       string   // --node-selector
}

func NewResumeCommand() *cobra.Command {
	var (
		resumeArgs resumeOps
	)

	var command = &cobra.Command{
		Use:   "resume WORKFLOW1 WORKFLOW2...",
		Short: "resume zero or more workflows",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(resumeArgs.nodeSelector)
			if err != nil {
				log.Fatalf("Unable to parse node selector '%s': %s", resumeArgs.nodeSelector, err)
			}

			for _, wfName := range args {
				_, err := serviceClient.ResumeWorkflow(ctx, &workflowpkg.WorkflowResumeRequest{
					Name:      wfName,
					Namespace: namespace,
					NodeSelector: selector.String(),
				})
				if err != nil {
					log.Fatalf("Failed to resume %s: %+v", wfName, err)
				}
				fmt.Printf("workflow %s resumed\n", wfName)
			}

		},
	}
	command.Flags().StringVar(&resumeArgs.nodeSelector, "node-selector", "", "selector of node to resume, eg: --node-selector inputs.paramaters.myparam.value=abc")
	return command
}
