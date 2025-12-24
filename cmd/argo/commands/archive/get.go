package archive

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/humanize"
)

func NewGetCommand() *cobra.Command {
	var (
		output = common.EnumFlagValue{
			AllowedValues: []string{"json", "yaml", "wide"},
			Value:         "wide",
		}
		forceName bool
		forceUID  bool
	)
	command := &cobra.Command{
		Use:   "get WORKFLOW",
		Short: "get a workflow in the archive",
		Args:  cobra.ExactArgs(1),
		Example: `# Get information about an archived workflow by name:
  argo archive get my-workflow

# Get information about an archived workflow by UID (auto-detected):
  argo archive get a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11

# Get information about an archived workflow in YAML format:
  argo archive get my-workflow -o yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]

			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}

			var wf *wfv1.Workflow
			isUID := isUID(identifier)
			if forceUID {
				isUID = true
			} else if forceName {
				isUID = false
			}
			if isUID {
				// Lookup by UID
				wf, err = serviceClient.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: identifier})
			} else {
				// Lookup by Name
				namespace := client.Namespace(ctx)
				wf, err = serviceClient.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{
					Name:      identifier,
					Namespace: namespace,
				})
			}
			if err != nil {
				return err
			}
			printWorkflow(wf, output.String())
			return nil
		},
	}
	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
	command.Flags().BoolVar(&forceName, "name", false, "force the argument to be treated as a name")
	command.Flags().BoolVar(&forceUID, "uid", false, "force the argument to be treated as a UID")
	command.MarkFlagsMutuallyExclusive("name", "uid")
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
		fmt.Printf(fmtStr, "Name:", wf.Name)
		fmt.Printf(fmtStr, "Namespace:", wf.Namespace)
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
		fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.CreationTimestamp.Time))
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
