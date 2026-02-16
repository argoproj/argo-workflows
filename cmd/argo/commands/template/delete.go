package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var all bool

	command := &cobra.Command{
		Use:   "delete WORKFLOW_TEMPLATE",
		Short: "delete a workflow template",
		Example: `
# Delete a workflow template by its name:
    argo template delete <my-template>

# Delete all workflow templates:
    argo template delete --all
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiServerDeleteWorkflowTemplates(cmd.Context(), all, args)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflow templates")
	return command
}

func apiServerDeleteWorkflowTemplates(ctx context.Context, allWFs bool, wfTmplNames []string) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
	if err != nil {
		return err
	}
	namespace := client.Namespace(ctx)
	var delWFTmplNames []string
	if allWFs {
		wftmplList, err := serviceClient.ListWorkflowTemplates(ctx, &workflowtemplatepkg.WorkflowTemplateListRequest{
			Namespace: namespace,
		})
		if err != nil {
			return err
		}
		for _, wfTmpl := range wftmplList.Items {
			delWFTmplNames = append(delWFTmplNames, wfTmpl.Name)
		}
	} else {
		delWFTmplNames = wfTmplNames
	}
	for _, wfTmplNames := range delWFTmplNames {
		if err := apiServerDeleteWorkflowTemplate(ctx, serviceClient, namespace, wfTmplNames); err != nil {
			return err
		}
	}
	return nil
}

func apiServerDeleteWorkflowTemplate(ctx context.Context, client workflowtemplatepkg.WorkflowTemplateServiceClient, namespace, wftmplName string) error {
	_, err := client.DeleteWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateDeleteRequest{
		Name:      wftmplName,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}
	fmt.Printf("WorkflowTemplate '%s' deleted\n", wftmplName)
	return nil
}
