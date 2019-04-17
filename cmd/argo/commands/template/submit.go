package template

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/validate"
)

type cliSubmitOpts struct {
	output string // --output
	strict bool   // --strict
}

func NewSubmitCommand() *cobra.Command {
	var (
		cliSubmitOpts cliSubmitOpts
	)
	var command = &cobra.Command{
		Use:   "submit FILE1 FILE2...",
		Short: "submit a workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			SubmitWorkflowTemplates(args, &cliSubmitOpts)
		},
	}
	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliSubmitOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func SubmitWorkflowTemplates(filePaths []string, cliOpts *cliSubmitOpts) {
	if cliOpts == nil {
		cliOpts = &cliSubmitOpts{}
	}
	wftmplClient := InitWorkflowTemplateClient()
	var workflowTemplates []wfv1.WorkflowTemplate
	if len(filePaths) == 1 && filePaths[0] == "-" {
		reader := bufio.NewReader(os.Stdin)
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Fatal(err)
		}
		workflowTemplates = unmarshalWorkflowTemplates(body, cliOpts.strict)
	} else {
		for _, filePath := range filePaths {
			var body []byte
			var err error
			if cmdutil.IsURL(filePath) {
				response, err := http.Get(filePath)
				if err != nil {
					log.Fatal(err)
				}
				body, err = ioutil.ReadAll(response.Body)
				_ = response.Body.Close()
				if err != nil {
					log.Fatal(err)
				}
			} else {
				body, err = ioutil.ReadFile(filePath)
				if err != nil {
					log.Fatal(err)
				}
			}
			wftmpls := unmarshalWorkflowTemplates(body, cliOpts.strict)
			workflowTemplates = append(workflowTemplates, wftmpls...)
		}
	}

	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		log.Fatal(err)
	}

	var workflowTemplateNames []string
	for _, wftmpl := range workflowTemplates {
		err := validate.ValidateWorkflowTemplate(wfClientset, namespace, &wftmpl)
		if err != nil {
			log.Fatalf("Failed to submit workflow template: %v", err)
		}
		created, err := wftmplClient.Create(&wftmpl)
		if err != nil {
			log.Fatalf("Failed to submit workflow template: %v", err)
		}
		printWorkflowTemplate(created, cliOpts.output)
		workflowTemplateNames = append(workflowTemplateNames, created.Name)
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
