package clustertemplate

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
)

type cliCreateOpts struct {
	output string // --output
	strict bool   // --strict
}

func NewCreateCommand() *cobra.Command {
	var cliCreateOpts cliCreateOpts
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

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			createClusterWorkflowTemplates(cmd.Context(), args, &cliCreateOpts)
		},
	}
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func createClusterWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliCreateOpts) {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	clusterWorkflowTemplates := generateClusterWorkflowTemplates(filePaths, cliOpts.strict)

	for _, wftmpl := range clusterWorkflowTemplates {
		created, err := serviceClient.CreateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest{
			Template: &wftmpl,
		})
		if err != nil {
			log.Fatalf("Failed to create cluster workflow template: %s,  %v", wftmpl.Name, err)
		}
		printClusterWorkflowTemplate(created, cliOpts.output)
	}
}
