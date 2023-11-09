package archive

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/argoproj/pkg/humanize"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func NewGetCommand() *cobra.Command {
	var output string
	command := &cobra.Command{
		Use:   "get UID",
		Short: "get a workflow in the archive",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			uid := args[0]

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			wf, err := serviceClient.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: uid})
			errors.CheckError(err)
			printWorkflow(wf, output)
		},
	}
	command.Flags().StringVarP(&output, "output", "o", "wide", "Output format. One of: json|yaml|wide")
	return command
}

func printWorkflow(wf *wfv1.Workflow, output string) {

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
		serviceAccount := wf.GetExecSpec().ServiceAccountName
		if serviceAccount == "" {
			// if serviceAccountName was not specified in a submitted Workflow, we will
			// use the serviceAccountName provided in Workflow Defaults (if any). If that
			// also isn't set, we will use the 'default' ServiceAccount in the namespace
			// the workflow will run in.
			serviceAccount = "unset (will run with the default ServiceAccount)"
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

}
