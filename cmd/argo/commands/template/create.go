package template

import (
	"log"
	"os"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
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
		Use:   "create FILE1 FILE2...",
		Short: "create a workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			CreateWorkflowTemplates(args, &cliCreateOpts)
		},
	}
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func CreateWorkflowTemplates(filePaths []string, cliOpts *cliCreateOpts) {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	defaultWFTmplClient := InitWorkflowTemplateClient()

	fileContents, err := util.ReadManifest(filePaths...)
	if err != nil {
		log.Fatal(err)
	}

	var workflowTemplates []wfv1.WorkflowTemplate
	for _, body := range fileContents {
		wftmpls := unmarshalWorkflowTemplates(body, cliOpts.strict)
		workflowTemplates = append(workflowTemplates, wftmpls...)
	}

	if len(workflowTemplates) == 0 {
		log.Println("No WorkflowTemplate found in given files")
		os.Exit(1)
	}

	for _, wftmpl := range workflowTemplates {
		err := validate.ValidateWorkflowTemplate(wfClientset, namespace, &wftmpl)
		if err != nil {
			log.Fatalf("Failed to create workflow template: %v", err)
		}
		wftmplClient := defaultWFTmplClient
		if wftmpl.Namespace != "" {
			wftmplClient = InitWorkflowTemplateClient(wftmpl.Namespace)
		}
		created, err := wftmplClient.Create(&wftmpl)
		if err != nil {
			log.Fatalf("Failed to create workflow template: %v", err)
		}
		printWorkflowTemplate(created, cliOpts.output)
	}
}

// unmarshalWorkflowTemplates unmarshals the input bytes as either json or yaml
func unmarshalWorkflowTemplates(wfBytes []byte, strict bool) []wfv1.WorkflowTemplate {
	var wf wfv1.WorkflowTemplate
	var jsonOpts []json.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, json.DisallowUnknownFields)
	}
	err := json.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.WorkflowTemplate{wf}
	}
	yamlWfs, err := common.SplitWorkflowTemplateYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs
	}
	log.Fatalf("Failed to parse workflow template: %v", err)
	return nil
}
