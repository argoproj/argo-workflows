package template

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func NewUpdateCommand() *cobra.Command {
	var cliUpdateOpts cliCreateOpts
	command := &cobra.Command{
		Use:   "update FILE1 FILE2...",
		Short: "update a workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			updateWorkflowTemplates(args, &cliUpdateOpts)
		},
	}
	command.Flags().StringVarP(&cliUpdateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliUpdateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func updateWorkflowTemplates(filePaths []string, cliOpts *cliCreateOpts) {
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

	for _, wftmpl := range workflowTemplates {
		if wftmpl.Namespace == "" {
			wftmpl.Namespace = client.Namespace()
		}
		current, err := serviceClient.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{
			Name:      wftmpl.Name,
			Namespace: wftmpl.Namespace,
		})
		if err != nil {
			log.Fatalf("Failed to get existing workflow template %q to update: %v", wftmpl.Name, err)
		}
		wftmpl.ResourceVersion = current.ResourceVersion
		updated, err := serviceClient.UpdateWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateUpdateRequest{
			Namespace: wftmpl.Namespace,
			Template:  &wftmpl,
		})
		if err != nil {
			log.Fatalf("Failed to update workflow template: %v", err)
		}
		printWorkflowTemplate(updated, cliOpts.output)
	}
}
