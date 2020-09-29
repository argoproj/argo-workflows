package clustertemplate

import (
	"encoding/json"
	"fmt"

	"github.com/argoproj/pkg/humanize"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	cmdcommon "github.com/argoproj/argo/cmd/argo/commands/common"
	clusterworkflowtmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NewGetCommand() *cobra.Command {
	var (
		output string
	)

	var command = &cobra.Command{
		Use:          "get CLUSTER WORKFLOW_TEMPLATE...",
		Short:        "display details about a cluster workflow template",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := cmdcommon.CreateNewAPIClientFunc()
			serviceClient := apiClient.NewClusterWorkflowTemplateServiceClient()
			for _, name := range args {
				wftmpl, err := serviceClient.GetClusterWorkflowTemplate(ctx, &clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest{
					Name: name,
				})
				if err != nil {
					return err
				}
				err = printClusterWorkflowTemplate(wftmpl, output)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: json|yaml|wide")
	return command
}

func printClusterWorkflowTemplate(wf *wfv1.ClusterWorkflowTemplate, outFmt string) error {
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
		return fmt.Errorf("Unknown output format: %s", outFmt)
	}
	return nil
}

func printClusterWorkflowTemplateHelper(wf *wfv1.ClusterWorkflowTemplate) {
	const fmtStr = "%-20s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
}
