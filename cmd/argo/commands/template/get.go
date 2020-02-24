package template

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/pkg/humanize"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NewGetCommand() *cobra.Command {
	var (
		output string
	)

	var command = &cobra.Command{
		Use:   "get WORKFLOW_TEMPLATE...",
		Short: "display details about a workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowTemplateServiceClient()
			namespace := client.Namespace()
			for _, name := range args {
				wftmpl, err := serviceClient.GetWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateGetRequest{
					Name:      name,
					Namespace: namespace,
				})
				if err != nil {
					log.Fatal(err)
				}
				printWorkflowTemplate(wftmpl, output)
			}
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: json|yaml|wide")
	return command
}

func printWorkflowTemplate(wf *wfv1.WorkflowTemplate, outFmt string) {
	switch outFmt {
	case "name":
		fmt.Println(wf.ObjectMeta.Name)
	case "json":
		outBytes, _ := json.MarshalIndent(wf, "", "    ")
		fmt.Println(string(outBytes))
	case "yaml":
		outBytes, _ := yaml.Marshal(wf)
		fmt.Print(string(outBytes))
	case "wide", "":
		printWorkflowTemplateHelper(wf)
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func printWorkflowTemplateHelper(wf *wfv1.WorkflowTemplate) {
	const fmtStr = "%-20s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Namespace:", wf.ObjectMeta.Namespace)
	fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
}
