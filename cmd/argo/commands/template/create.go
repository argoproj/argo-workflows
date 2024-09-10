package template

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
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

			CreateWorkflowTemplates(cmd.Context(), args, &cliCreateOpts)
		},
	}
	command.Flags().StringVarP(&cliCreateOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVar(&cliCreateOpts.strict, "strict", true, "perform strict workflow validation")
	return command
}

func CreateWorkflowTemplates(ctx context.Context, filePaths []string, cliOpts *cliCreateOpts) {
	if cliOpts == nil {
		cliOpts = &cliCreateOpts{}
	}
	ctx, apiClient := client.NewAPIClient(ctx)
	serviceClient, err := apiClient.NewWorkflowTemplateServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	workflowTemplates := generateWorkflowTemplates(filePaths, cliOpts.strict)

	for _, wftmpl := range workflowTemplates {
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
