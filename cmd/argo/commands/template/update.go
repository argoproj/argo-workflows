package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
)

type cliUpdateOpts struct {
	output common.EnumFlagValue // --output
	strict bool                 // --strict
}

func NewUpdateCommand() *cobra.Command {
	var cliUpdateOpts = cliUpdateOpts{output: common.NewPrintWorkflowOutputValue("")}
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
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return updateWorkflowTemplates(ctx, args, &cliUpdateOpts)
		},
	}
	command.Flags().VarP(&cliUpdateOpts.output, "output", "o", "Output format. "+cliUpdateOpts.output.Usage())
	command.Flags().BoolVar(&cliUpdateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliUpdateOpts) error {
	if cliOpts == nil {
		cliOpts = &cliUpdateOpts{}
	}
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
	if err != nil {
		return err
	}

	workflowTemplates := generateWorkflowTemplates(ctx, filePaths, cliOpts.strict)

	for _, wftmpl := range workflowTemplates {
		if wftmpl.Namespace == "" {
			wftmpl.Namespace = client.Namespace(ctx)
		}
		current, err := serviceClient.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{
			Name:      wftmpl.Name,
			Namespace: wftmpl.Namespace,
		})
		if err != nil {
			return fmt.Errorf("failed to get existing workflow template %q to update: %w", wftmpl.Name, err)
		}
		wftmpl.ResourceVersion = current.ResourceVersion
		updated, err := serviceClient.UpdateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateUpdateRequest{
			Namespace: wftmpl.Namespace,
			Template:  &wftmpl,
		})
		if err != nil {
			return fmt.Errorf("failed to update workflow template: %w", err)
		}
		printWorkflowTemplate(updated, cliOpts.output.String())
	}
	return nil
}
