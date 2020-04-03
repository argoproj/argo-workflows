package clustertemplate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/pkg/errors"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var (
		all bool
	)

	var command = &cobra.Command{
		Use:   "delete WORKFLOW_TEMPLATE",
		Short: "delete a cluster workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			apiServerDeleteClusterWorkflowTemplates(all, args)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all cluster workflow templates")
	return command
}

func apiServerDeleteClusterWorkflowTemplates(allWFs bool, wfTmplNames []string) {
	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()
	var delWFTmplNames []string
	if allWFs {
		cwftmplList, err := serviceClient.ListClusterWorkflowTemplates(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateListRequest{})
		errors.CheckError(err)
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
		errors.CheckError(err)
		fmt.Printf("ClusterWorkflowTemplate '%s' deleted\n", cwfTmplName)
	}
}
