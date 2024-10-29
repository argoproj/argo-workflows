package clustertemplate

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
)

type cliUpdateOpts struct {
	output common.EnumFlagValue // --output
	strict bool                 // --strict
}

func NewUpdateCommand() *cobra.Command {
	var cliUpdateOpts = cliUpdateOpts{output: common.NewPrintWorkflowOutputValue("")}
	command := &cobra.Command{
		Use:   "update FILE1 FILE2...",
		Short: "update a cluster workflow template",
		Example: `# Update a Cluster Workflow Template:
  argo cluster-template update FILE1
	
# Update a Cluster Workflow Template and print it as YAML:
  argo cluster-template update FILE1 --output yaml
  
# Update a Cluster Workflow Template with relaxed validation:
  argo cluster-template update FILE1 --strict false
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateClusterWorkflowTemplates(cmd.Context(), args, &cliUpdateOpts)
		},
	}
	command.Flags().VarP(&cliUpdateOpts.output, "output", "o", "Output format. "+cliUpdateOpts.output.Usage())
	command.Flags().BoolVar(&cliUpdateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateClusterWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliUpdateOpts) error {
	if cliOpts == nil {
		cliOpts = &cliUpdateOpts{}
	}
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
	if err != nil {
		return err
	}

	clusterWorkflowTemplates := generateClusterWorkflowTemplates(filePaths, cliOpts.strict)

	for _, wftmpl := range clusterWorkflowTemplates {
		current, err := serviceClient.GetClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest{
			Name: wftmpl.Name,
		})
		if err != nil {
			return fmt.Errorf("Failed to get existing cluster workflow template %q to update: %v", wftmpl.Name, err)
		}
		wftmpl.ResourceVersion = current.ResourceVersion
		updated, err := serviceClient.UpdateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest{
			Template: &wftmpl,
		})
		if err != nil {
			return fmt.Errorf("Failed to update cluster workflow template: %s,  %v", wftmpl.Name, err)
		}
		printClusterWorkflowTemplate(updated, cliOpts.output.String())
	}
	return nil
}
