package template

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
)

type cliUpdateOpts struct {
	output string // --output
	strict bool   // --strict
}

func NewUpdateCommand() *cobra.Command {
	var cliUpdateOpts cliUpdateOpts
	command := &cobra.Command{
		Use:   "update FILE1 FILE2...",
		Short: "update a workflow template",
		Example: `# Update a Workflow Template:
  argo template update FILE1
	
# Update a Workflow Template and print it as YAML:
  argo template update FILE1 --output yaml
  
# Update a Workflow Template with relaxed validation:
  argo template update FILE1 --strict false
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			updateWorkflowTemplates(cmd.Context(), args, &cliUpdateOpts)
		},
	}
	command.Flags().StringVarP(&cliUpdateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliUpdateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliUpdateOpts) {
	if cliOpts == nil {
		cliOpts = &cliUpdateOpts{}
	}
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	workflowTemplates := generateWorkflowTemplates(filePaths, cliOpts.strict)

	for _, wftmpl := range workflowTemplates {
		if wftmpl.Namespace == "" {
			wftmpl.Namespace = client.Namespace()
		}
		current, err := serviceClient.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{
			Name:      wftmpl.Name,
			Namespace: wftmpl.Namespace,
		})
		if err != nil {
			log.Fatalf("Failed to get existing workflow template %q to update: %v", wftmpl.Name, err)
		}
		wftmpl.ResourceVersion = current.ResourceVersion
		updated, err := serviceClient.UpdateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateUpdateRequest{
			Namespace: wftmpl.Namespace,
			Template:  &wftmpl,
		})
		if err != nil {
			log.Fatalf("Failed to update workflow template: %v", err)
		}
		printWorkflowTemplate(updated, cliOpts.output)
	}
}
