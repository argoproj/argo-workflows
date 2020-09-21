package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
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
		Example: `# Resume a workflow that has been stopped or suspended:

  argo resume my-wf

# Resume the latest workflow:
  argo resume @latest
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(resumeArgs.nodeFieldSelector)
			if err != nil {
				return fmt.Errorf("unable to parse node field selector '%s': %s", resumeArgs.nodeFieldSelector, err)
			}

			for _, wfName := range args {
				_, err := serviceClient.ResumeWorkflow(ctx, &workflowpkg.WorkflowResumeRequest{
					Name:              wfName,
					Namespace:         namespace,
					NodeFieldSelector: selector.String(),
				})
				if err != nil {
					return fmt.Errorf("Failed to resume %s: %+v", wfName, err)
				}
				fmt.Printf("workflow %s resumed\n", wfName)
			}
			return nil
		},
	}
	command.Flags().StringVar(&resumeArgs.nodeFieldSelector, "node-field-selector", "", "selector of node to resume, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	return command
}
