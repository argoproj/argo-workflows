package commands

import (
	"fmt"
	"log"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type resumeOps struct {
	nodeFieldSelector string // --node-field-selector
	labelSelector     string // --selector
	fieldSelector     string // --field-selector
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
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(resumeArgs.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", resumeArgs.nodeFieldSelector, err)
			}

			var names []string

			if resumeArgs.labelSelector != "" || resumeArgs.fieldSelector != "" {
				listed, err := listWorkflows(ctx, serviceClient, listFlags{
					namespace: namespace,
					labels:    resumeArgs.labelSelector,
					fields:    resumeArgs.fieldSelector,
				})
				errors.CheckError(err)

				for _, w := range listed {
					names = append(names, w.GetName())
				}
			}

			names = append(names, args...)
			for _, wfName := range names {
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
	command.Flags().StringVarP(&resumeArgs.labelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&resumeArgs.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selectorkey1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	return command
}
