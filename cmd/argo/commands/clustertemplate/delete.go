package clustertemplate

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var all bool

	command := &cobra.Command{
		Use:   "delete WORKFLOW_TEMPLATE",
		Short: "delete a cluster workflow template",
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiServerDeleteClusterWorkflowTemplates(cmd.Context(), all, args)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all cluster workflow templates")
	return command
}

func apiServerDeleteClusterWorkflowTemplates(ctx context.Context, allWFs bool, wfTmplNames []string) error {
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
	if err != nil {
		return err
	}

	var delWFTmplNames []string
	if allWFs {
		cwftmplList, err := serviceClient.ListClusterWorkflowTemplates(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateListRequest{})
		if err != nil {
			return err
		}
		for _, cwfTmpl := range cwftmplList.Items {
			delWFTmplNames = append(delWFTmplNames, cwfTmpl.Name)
		}

	} else {
		delWFTmplNames = wfTmplNames
	}
	for _, cwfTmplName := range delWFTmplNames {
		_, err := serviceClient.DeleteClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest{
			Name: cwfTmplName,
		})
		if err != nil {
			return err
		}
		fmt.Printf("ClusterWorkflowTemplate '%s' deleted\n", cwfTmplName)
	}
	return nil
}
