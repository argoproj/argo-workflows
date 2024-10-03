package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func NewGetCommand() *cobra.Command {
	var getArgs = common.GetFlags{
		Output: common.EnumFlagValue{
			AllowedValues: []string{"name", "json", "yaml", "short", "wide"},
		},
	}

	command := &cobra.Command{
		Use:   "get WORKFLOW...",
		Short: "display details about a workflow",
		Example: `# Get information about a workflow:

  argo get my-wf

# Get the latest workflow:
  argo get @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			for _, name := range args {
				wf, err := serviceClient.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{
					Name:      name,
					Namespace: namespace,
				})
				errors.CheckError(err)
				printWorkflow(wf, getArgs)
			}
		},
	}

	command.Flags().VarP(&getArgs.Output, "output", "o", "Output format. "+getArgs.Output.Usage())
	command.Flags().BoolVar(&common.NoColor, "no-color", false, "Disable colorized output")
	command.Flags().BoolVar(&common.NoUtf8, "no-utf8", false, "Use plain 7-bits ascii characters")
	command.Flags().StringVar(&getArgs.Status, "status", "", "Filter by status (Pending, Running, Succeeded, Skipped, Failed, Error)")
	command.Flags().StringVar(&getArgs.NodeFieldSelectorString, "node-field-selector", "", "selector of node to display, eg: --node-field-selector phase=abc")
	return command
}

func printWorkflow(wf *wfv1.Workflow, getArgs common.GetFlags) {
	switch getArgs.Output.String() {
	case "name":
		fmt.Println(wf.ObjectMeta.Name)
	case "json":
		outBytes, _ := json.MarshalIndent(wf, "", "    ")
		fmt.Println(string(outBytes))
	case "yaml":
		outBytes, _ := yaml.Marshal(wf)
		fmt.Print(string(outBytes))
	case "short", "wide", "":
		fmt.Print(common.PrintWorkflowHelper(wf, getArgs))
	default:
		log.Fatalf("Unknown output format: %s", getArgs.Output)
	}
}
