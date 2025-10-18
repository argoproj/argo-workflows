package commands

import (
	"errors"
	"fmt"

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
		
  argo resume --node-field-selector inputs.parameters.myparam.value=abc		
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && resumeArgs.nodeFieldSelector == "" {
				return errors.New("requires either node field selector or workflow")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, apiClient, err := client.NewAPIClient(ctx)
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx)
			namespace := client.Namespace(ctx)

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
					return fmt.Errorf("failed to resume %s: %+v", wfName, err)
				}
				fmt.Printf("workflow %s resumed\n", wfName)
			}
			return nil
		},
	}
	command.Flags().StringVar(&resumeArgs.nodeFieldSelector, "node-field-selector", "", "selector of node to resume, eg: --node-field-selector inputs.parameters.myparam.value=abc")
	return command
}
