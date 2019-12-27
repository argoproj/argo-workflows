package history

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/argoproj/pkg/humanize"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflowhistory"
	"github.com/argoproj/argo/workflow/packer"
)

func NewGetCommand() *cobra.Command {
	var (
		output string
	)
	var command = &cobra.Command{
		Use: "get NAMESPACE UID",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			namespace := args[0]
			uid := args[1]
			conn := client.GetClientConn()
			ctx := client.ContextWithAuthorization()
			wfHistoryClient := workflowhistory.NewWorkflowHistoryServiceClient(conn)
			wf, err := wfHistoryClient.GetWorkflowHistory(ctx, &workflowhistory.WorkflowHistoryGetRequest{
				Namespace:  namespace,
				Uid:        uid,
				GetOptions: &metav1.GetOptions{},
			})
			if err != nil {
				log.Fatal(err)
			}
			err = packer.DecompressWorkflow(wf)
			if err != nil {
				log.Fatal(err)
			}
			switch output {
			case "json":
				output, err := json.Marshal(wf)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(output))
			case "yaml":
				output, err := yaml.Marshal(wf)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(output))
			default:
				const fmtStr = "%-20s %v\n"
				fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
				fmt.Printf(fmtStr, "Namespace:", wf.ObjectMeta.Namespace)
				serviceAccount := wf.Spec.ServiceAccountName
				if serviceAccount == "" {
					serviceAccount = "default"
				}
				fmt.Printf(fmtStr, "ServiceAccount:", serviceAccount)
				fmt.Printf(fmtStr, "Status:", wf.Status.Phase)
				if wf.Status.Message != "" {
					fmt.Printf(fmtStr, "Message:", wf.Status.Message)
				}
				fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
				if !wf.Status.StartedAt.IsZero() {
					fmt.Printf(fmtStr, "Started:", humanize.Timestamp(wf.Status.StartedAt.Time))
				}
				if !wf.Status.FinishedAt.IsZero() {
					fmt.Printf(fmtStr, "Finished:", humanize.Timestamp(wf.Status.FinishedAt.Time))
				}
				if !wf.Status.StartedAt.IsZero() {
					fmt.Printf(fmtStr, "Duration:", humanize.RelativeDuration(wf.Status.StartedAt.Time, wf.Status.FinishedAt.Time))
				}
			}
		},
	}
	command.Flags().StringVarP(&output, "output", "o", "wide", "Output format. One of: json|yaml|wide")
	return command
}
