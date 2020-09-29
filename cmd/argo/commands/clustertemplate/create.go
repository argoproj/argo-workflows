package clustertemplate

import (
	"fmt"
	"log"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

type cliCreateOpts struct {
	output string // --output
	strict bool   // --strict
}

func NewCreateCommand() *cobra.Command {
	var (
		cliCreateOpts cliCreateOpts
	)
	var command = &cobra.Command{
		Use:          "create FILE1 FILE2...",
		Short:        "create a cluster workflow template",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				return cmdcommon.MissingArgumentsError
			}

			return createClusterWorkflowTemplates(args, &cliCreateOpts)
		},
	}
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func createClusterWorkflowTemplates(filePaths []string, cliOpts *cliCreateOpts) error {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
	serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()

	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		return err
	}

	var clusterWorkflowTemplates []wfv1.ClusterWorkflowTemplate
	for _, body := range fileContents {
		cwftmpls, err := unmarshalClusterWorkflowTemplates(body, cliOpts.strict)
		if err != nil {
			return err
		}
		clusterWorkflowTemplates = append(clusterWorkflowTemplates, cwftmpls...)
	}

	if len(clusterWorkflowTemplates) == 0 {
		log.Println("No cluster workflow template found in given files")
		return nil
	}

	for _, wftmpl := range clusterWorkflowTemplates {
		created, err := serviceClient.CreateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest{
			Template: &wftmpl,
		})
		if err != nil {
			return fmt.Errorf("Failed to create cluster workflow template: %s,  %v", wftmpl.Name, err)
		}
		err = printClusterWorkflowTemplate(created, cliOpts.output)
		if err != nil {
			return err
		}
	}
	return nil
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
