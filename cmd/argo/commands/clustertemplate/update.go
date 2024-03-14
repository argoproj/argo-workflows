package clustertemplate

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

type cliUpdateOpts struct {
	output string // --output
	strict bool   // --strict
}

func NewUpdateCommand() *cobra.Command {
	var cliUpdateOpts cliUpdateOpts
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
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			updateClusterWorkflowTemplates(cmd.Context(), args, &cliUpdateOpts)
		},
	}
	command.Flags().StringVarP(&cliUpdateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliUpdateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateClusterWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliUpdateOpts) {
	if cliOpts == nil {
		cliOpts = &cliUpdateOpts{}
	}
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var clusterWorkflowTemplates []wfv1.ClusterWorkflowTemplate
	for _, body := range fileContents {
		cwftmpls, err := unmarshalClusterWorkflowTemplates(body, cliOpts.strict)
		if err != nil {
			log.Fatalf("Failed to parse cluster workflow template: %v", err)
		}
		clusterWorkflowTemplates = append(clusterWorkflowTemplates, cwftmpls...)
	}

	if len(clusterWorkflowTemplates) == 0 {
		log.Println("No cluster workflow template found in given files")
		os.Exit(1)
	}

	for _, wftmpl := range clusterWorkflowTemplates {
		current, err := serviceClient.GetClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest{
			Name: wftmpl.Name,
		})
		if err != nil {
			log.Fatalf("Failed to get existing cluster workflow template %q to update: %v", wftmpl.Name, err)
		}
		wftmpl.ResourceVersion = current.ResourceVersion
		updated, err := serviceClient.UpdateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest{
			Template: &wftmpl,
		})
		if err != nil {
			log.Fatalf("Failed to update cluster workflow template: %s,  %v", wftmpl.Name, err)
		}
		printClusterWorkflowTemplate(updated, cliOpts.output)
	}
}
