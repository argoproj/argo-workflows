package template

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/argoproj/pkg/json"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
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

var templateErrorRegex *regexp.Regexp = nil

// Compiling regex is expensive, only compile this once if we need it
func getTemplateErrorRegex() *regexp.Regexp {
	if templateErrorRegex != nil {
		return templateErrorRegex
	}
	templateErrorRegex = regexp.MustCompile(`template reference (.+?) not found`)
	return templateErrorRegex
}

func CreateWorkflowTemplates(filePaths []string, cliOpts *cliCreateOpts) {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewWorkflowTemplateServiceClient()

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
		log.Println("No workflow template found in given files")
		os.Exit(1)
	}

	var retryWorkflowTemplates []wfv1.WorkflowTemplate

main:
	for _, wftmpl := range workflowTemplates {
		if wftmpl.Namespace == "" {
			wftmpl.Namespace = client.Namespace()
		}
		created, err := serviceClient.CreateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateCreateRequest{
			Namespace: wftmpl.Namespace,
			Template:  &wftmpl,
		})

		if err != nil {

			// If we are trying to create a template that references another template that is about to be created with this
			// operation (but has not been created yet), that template will fail with a "reference not found" error. If
			// a template returns that error, we check to see if the reference is indeed about to be created in this operation.
			// If it is, we add this template to a list and then retry to create it once all templates have had a chance
			// to be created.
			re := getTemplateErrorRegex()
			// Check if the error received matches a "reference not found" error.
			if referenceNotFoundErrorMatch := re.FindStringSubmatch(err.Error()); len(referenceNotFoundErrorMatch) > 0 {
				reference := referenceNotFoundErrorMatch[1]
				// Check if the template referenced is indeed in this operation
				for _, w := range workflowTemplates {
					if strings.HasPrefix(reference, w.Name) || strings.HasSuffix(reference, w.GenerateName) {
						// Add this template to the list of references to retry
						retryWorkflowTemplates = append(retryWorkflowTemplates, w)
						continue main
					}
				}
			}

			log.Fatalf("Failed to create workflow template: %v", err)
		}
		printWorkflowTemplate(created, cliOpts.output)
	}

	// Retry the workflow templates that were not created due to "reference not found" error (see above). This will usually
	// be empty and not run.
	for _, wftmpl := range retryWorkflowTemplates {
		if wftmpl.Namespace == "" {
			wftmpl.Namespace = client.Namespace()
		}
		created, err := serviceClient.CreateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateCreateRequest{
			Namespace: wftmpl.Namespace,
			Template:  &wftmpl,
		})
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
