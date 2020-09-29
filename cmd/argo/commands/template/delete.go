package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var (
		all bool
	)

	var command = &cobra.Command{
		Use:          "delete WORKFLOW_TEMPLATE",
		Short:        "delete a workflow template",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiServerDeleteWorkflowTemplates(all, args)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all workflow templates")
	return command
}

func apiServerDeleteWorkflowTemplates(allWFs bool, wfTmplNames []string) error {
	ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
	serviceClient := apiClient.NewWorkflowTemplateServiceClient()
	namespace := client.Namespace()
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
		err := apiServerDeleteWorkflowTemplate(serviceClient, ctx, namespace, wfTmplNames)
		if err != nil {
			return err
		}
	}
	return nil
}

func apiServerDeleteWorkflowTemplate(client workflowtemplatepkg.WorkflowTemplateServiceClient, ctx context.Context, namespace, wftmplName string) error {
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
