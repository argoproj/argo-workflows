package template

import (
	"context"
	"fmt"
	"log"

	"github.com/argoproj/pkg/errors"
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
		Example: `# Delete a workflow template

  argo template delete my-wftmpl

# Delete all workflow templates

  argo template delete --all`,
		Run: func(cmd *cobra.Command, args []string) {
			apiServerDeleteWorkflowTemplates(cmd.Context(), all, args)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflow templates")
	return command
}

func apiServerDeleteWorkflowTemplates(ctx context.Context, allWFs bool, wfTmplNames []string) {
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
	if err != nil {
		log.Fatal(err)
	}
	namespace := client.Namespace()
	var delWFTmplNames []string
	if allWFs {
		wftmplList, err := serviceClient.ListWorkflowTemplates(ctx, &workflowtemplatepkg.WorkflowTemplateListRequest{
			Namespace: namespace,
		})
		if err != nil {
			log.Fatal(err)
		}
		for _, wfTmpl := range wftmplList.Items {
			delWFTmplNames = append(delWFTmplNames, wfTmpl.Name)
		}

	} else {
		delWFTmplNames = wfTmplNames
	}
	for _, wfTmplNames := range delWFTmplNames {
		apiServerDeleteWorkflowTemplate(serviceClient, ctx, namespace, wfTmplNames)
	}
}

func apiServerDeleteWorkflowTemplate(client workflowtemplatepkg.WorkflowTemplateServiceClient, ctx context.Context, namespace, wftmplName string) {
	_, err := client.DeleteWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateDeleteRequest{
		Name:      wftmplName,
		Namespace: namespace,
	})
	if err != nil {
		errors.CheckError(err)
	}
	fmt.Printf("WorkflowTemplate '%s' deleted\n", wftmplName)
}
