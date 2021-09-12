package clustertemplate

import (
	"context"
	"log"
	"os"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
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
		created, err := serviceClient.CreateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest{
			Template: &wftmpl,
		})
		if err != nil {
			log.Fatalf("Failed to create cluster workflow template: %s,  %v", wftmpl.Name, err)
		}
		printClusterWorkflowTemplate(created, cliOpts.output)
	}
}

// unmarshalClusterWorkflowTemplates unmarshals the input bytes as either json or yaml
func unmarshalClusterWorkflowTemplates(wfBytes []byte, strict bool) ([]wfv1.ClusterWorkflowTemplate, error) {
	var cwft wfv1.ClusterWorkflowTemplate
	var jsonOpts []json.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, json.DisallowUnknownFields)
	}
	err := json.Unmarshal(wfBytes, &cwft, jsonOpts...)
	if err == nil {
		return []wfv1.ClusterWorkflowTemplate{cwft}, nil
	}
	yamlWfs, err := common.SplitClusterWorkflowTemplateYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs, nil
	}
	return nil, err
}
