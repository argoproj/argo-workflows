package clustertemplate

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/argoproj/pkg/humanize"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func NewGetCommand() *cobra.Command {
	var output string

	command := &cobra.Command{
		Use:   "get CLUSTER WORKFLOW_TEMPLATE...",
		Short: "display details about a cluster workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewClusterWorkflowTemplateServiceClient()
			if err != nil {
				log.Fatal(err)
			}
			for _, name := range args {
				wftmpl, err := serviceClient.GetClusterWorkflowTemplate(ctx, &clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest{
					Name: name,
				})
				if err != nil {
					log.Fatal(err)
				}
				printClusterWorkflowTemplate(wftmpl, output)
			}
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: json|yaml|wide")
	return command
}

func printClusterWorkflowTemplate(wf *wfv1.ClusterWorkflowTemplate, outFmt string) {
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
		printClusterWorkflowTemplateHelper(wf)
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func printClusterWorkflowTemplateHelper(wf *wfv1.ClusterWorkflowTemplate) {
	const fmtStr = "%-20s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
}
