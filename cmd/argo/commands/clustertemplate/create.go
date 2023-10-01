package main

import (
	"context"
	"log"
	"os"

	"github.com/argoproj/pkg/json"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/clustertemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/spf13/cobra"
)

type cliCreateOpts struct {
	output string // --output
	strict bool   // --strict
}

func main() {
	ctx := context.Background()

	// List of file paths containing cluster workflow template definitions
	filePaths := []string{"template1.yaml", "template2.json"}

	// Create options for the cluster workflow template creation
	cliCreateOpts := cliCreateOpts{
		output: "json", // Output format (json|yaml|name|wide)
		strict: true,   // Perform strict workflow validation
	}

	// Create the cluster workflow templates
	createClusterWorkflowTemplates(ctx, filePaths, &cliCreateOpts)
}

func createClusterWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliCreateOpts) {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}

	// Create an API client and service client
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Read file contents from the provided file paths
	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var clusterWorkflowTemplates []v1alpha1.ClusterWorkflowTemplate
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
		// Create the cluster workflow template
		created, err := serviceClient.CreateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest{
			Template: &wftmpl,
		})
		if err != nil {
			log.Fatalf("Failed to create cluster workflow template: %s, %v", wftmpl.Name, err)
		}
		printClusterWorkflowTemplate(created, cliOpts.output)
	}
}

// unmarshalClusterWorkflowTemplates unmarshals the input bytes as either JSON or YAML
func unmarshalClusterWorkflowTemplates(wfBytes []byte, strict bool) ([]v1alpha1.ClusterWorkflowTemplate, error) {
	var cwft v1alpha1.ClusterWorkflowTemplate
	var jsonOpts []json.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, json.DisallowUnknownFields)
	}
	err := json.Unmarshal(wfBytes, &cwft, jsonOpts...)
	if err == nil {
		return []v1alpha1.ClusterWorkflowTemplate{cwft}, nil
	}
	yamlWfs, err := common.SplitClusterWorkflowTemplateYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs, nil
	}
	return nil, err
}

// printClusterWorkflowTemplate is a helper function to print the created cluster workflow template
func printClusterWorkflowTemplate(template *v1alpha1.ClusterWorkflowTemplate, outputFormat string) {
	// Your printing logic here...
}
