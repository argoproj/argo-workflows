package clustertemplate

import (
	"context"
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
)

var (
	deleteExample = `# Delete one or more cluster workflow templates

  argo cluster-template delete cwf-template-name1 cwf-template-name2

# Delete all cluster workflow templates

  argo cluster-template delete --all`
)

// NewDeleteCommand returns a new instance of an `argo delete` command
func NewDeleteCommand() *cobra.Command {
	var all bool

	command := &cobra.Command{
		Use:     "delete WORKFLOW_TEMPLATE",
		Short:   "delete a cluster workflow template",
		Example: deleteExample,
		Run: func(cmd *cobra.Command, args []string) {
			apiServerDeleteClusterWorkflowTemplates(cmd.Context(), all, args)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Delete all cluster workflow templates")
	return command
}

func apiServerDeleteClusterWorkflowTemplates(ctx context.Context, allWFs bool, wfTmplNames []string) {
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
	errors.CheckError(err)

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
