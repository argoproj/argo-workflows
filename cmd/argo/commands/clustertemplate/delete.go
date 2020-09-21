package clustertemplate

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiServerDeleteClusterWorkflowTemplates(all, args)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all cluster workflow templates")
	return command
}

func apiServerDeleteClusterWorkflowTemplates(allWFs bool, wfTmplNames []string) error {
	ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
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
		if err != nil {
			return err
		}
		fmt.Printf("ClusterWorkflowTemplate '%s' deleted\n", cwfTmplName)
	}
	return nil
}
