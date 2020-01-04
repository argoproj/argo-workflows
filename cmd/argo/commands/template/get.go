package template

import (
	"encoding/json"
	"fmt"
	"log"
	"os"


	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/pkg/humanize"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowtemplate"
)

func NewGetCommand() *cobra.Command {
	var (
		output string
	)

	var command = &cobra.Command{
		Use:   "get WORKFLOW_TEMPLATE",
		Short: "display details about a workflow template",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			var wftmpl *wfv1.WorkflowTemplate
			var err error
			if client.ArgoServer != "" {
				conn := client.GetClientConn()
				defer conn.Close()
				ns, _, _ := client.Config.Namespace()
				wftmplApiClient, ctx := GetWFtmplApiServerGRPCClient(conn)
				for _, arg := range args {
					wfTempReq := workflowtemplate.WorkflowTemplateGetRequest{
						TemplateName: arg,
						Namespace:    ns,
					}
					wftmpl, err = wftmplApiClient.GetWorkflowTemplate(ctx, &wfTempReq)
					if err != nil {
						log.Fatal(err)
					}
					printWorkflowTemplate(wftmpl, output)
				}
			} else {
				wftmplClient := InitWorkflowTemplateClient()
				for _, arg := range args {
					wftmpl, err = wftmplClient.Get(arg, metav1.GetOptions{})
					if err != nil {
						log.Fatal(err)
					}
					printWorkflowTemplate(wftmpl, output)
				}
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
		printWorkflowTemplateHelper(wf, outFmt)
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func printWorkflowTemplateHelper(wf *wfv1.WorkflowTemplate, outFmt string) {
	const fmtStr = "%-20s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Namespace:", wf.ObjectMeta.Namespace)
	fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
}
