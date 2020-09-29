package template

import (
	"fmt"
	"log"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
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
		Short:        "create a workflow template",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				return cmdcommon.MissingArgumentsError
			}

			return CreateWorkflowTemplates(args, &cliCreateOpts)
		},
	}
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func CreateWorkflowTemplates(filePaths []string, cliOpts *cliCreateOpts) error {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
	serviceClient := apiClient.NewWorkflowTemplateServiceClient()

	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		return err
	}

	var workflowTemplates []wfv1.WorkflowTemplate
	for _, body := range fileContents {
		wftmpls, err := unmarshalWorkflowTemplates(body, cliOpts.strict)
		if err != nil {
			return err
		}
		workflowTemplates = append(workflowTemplates, wftmpls...)
	}

	if len(workflowTemplates) == 0 {
		log.Println("No workflow template found in given files")
		return nil
	}

	for _, wftmpl := range workflowTemplates {
		if wftmpl.Namespace == "" {
			wftmpl.Namespace = client.Namespace()
		}
		created, err := serviceClient.CreateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateCreateRequest{
			Namespace: wftmpl.Namespace,
			Template:  &wftmpl,
		})
		if err != nil {
			return fmt.Errorf("Failed to create workflow template: %v", err)
		}
		err = printWorkflowTemplate(created, cliOpts.output)
		if err != nil {
			return err
		}
	}
	return nil
}

// unmarshalWorkflowTemplates unmarshals the input bytes as either json or yaml
func unmarshalWorkflowTemplates(wfBytes []byte, strict bool) ([]wfv1.WorkflowTemplate, error) {
	var wf wfv1.WorkflowTemplate
	var jsonOpts []json.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, json.DisallowUnknownFields)
	}
	err := json.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.WorkflowTemplate{wf}, nil
	}
	yamlWfs, err := common.SplitWorkflowTemplateYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs, nil
	}

	return nil, fmt.Errorf("Failed to parse workflow template: %v", err)
}
