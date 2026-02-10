package clustertemplate

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
)

type cliCreateOpts struct {
	output common.EnumFlagValue // --output
	strict bool                 // --strict
}

func NewCreateCommand() *cobra.Command {
	var cliCreateOpts = cliCreateOpts{output: common.NewPrintWorkflowOutputValue("")}
	command := &cobra.Command{
		Use:   "create FILE1 FILE2...",
		Short: "create a cluster workflow template",
		Example: `# Create a Cluster Workflow Template:
  argo cluster-template create FILE1
	
# Create a Cluster Workflow Template and print it as YAML:
  argo cluster-template create FILE1 --output yaml
	
# Create a Cluster Workflow Template with relaxed validation:
  argo cluster-template create FILE1 --strict false
`,

		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createClusterWorkflowTemplates(cmd.Context(), args, &cliCreateOpts)
		},
	}
	command.Flags().VarP(&cliCreateOpts.output, "output", "o", "Output format. "+cliCreateOpts.output.Usage())
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func createClusterWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliCreateOpts) error {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	ctx, apiClient, err := client.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
	if err != nil {
		return err
	}

	clusterWorkflowTemplates := generateClusterWorkflowTemplates(ctx, filePaths, cliOpts.strict)

	for _, wftmpl := range clusterWorkflowTemplates {
		created, err := serviceClient.CreateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest{
			Template: &wftmpl,
		})
		if err != nil {
			return fmt.Errorf("failed to create cluster workflow template: %s,  %w", wftmpl.Name, err)
		}
		printClusterWorkflowTemplate(created, cliOpts.output.String())
	}
	return nil
}
