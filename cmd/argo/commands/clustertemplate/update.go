package clustertemplate

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func NewUpdateCommand() *cobra.Command {
	var cliCreateOpts cliCreateOpts
	command := &cobra.Command{
		Use:   "update FILE1 FILE2...",
		Short: "update a cluster workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			updateClusterWorkflowTemplates(args, &cliCreateOpts)
		},
	}
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateClusterWorkflowTemplates(filePaths []string, cliOpts *cliCreateOpts) {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()

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
