package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
)

type resumeOps struct {
	nodeFieldSelector string // --node-field-selector
}

func NewResumeCommand() *cobra.Command {
	var resumeArgs resumeOps

	command := &cobra.Command{
		Use:   "resume WORKFLOW1 WORKFLOW2...",
		Short: "resume zero or more workflows (opposite of suspend)",
		Example: `# Resume a workflow that has been suspended:

  argo resume my-wf

# Resume multiple workflows:
		
  argo resume my-wf my-other-wf my-third-wf		
		
# Resume the latest workflow:
		
  argo resume @latest
		
# Resume multiple workflows by node field selector:
		
  argo resume --node-field-selector inputs.paramaters.myparam.value=abc		
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && resumeArgs.nodeFieldSelector == "" {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(resumeArgs.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", resumeArgs.nodeFieldSelector, err)
			}

			for _, wfName := range args {
				_, err := serviceClient.ResumeWorkflow(ctx, &workflowpkg.WorkflowResumeRequest{
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
