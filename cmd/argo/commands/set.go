package commands

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type setOps struct {
	message           string // --message
	phase             string // --phase
	outputParameters  string // --output-parameters
	nodeFieldSelector string // --node-field-selector
}

func NewSetCommand() *cobra.Command {
	var (
		setArgs setOps
	)

	var command = &cobra.Command{
		Use:   "set WORKFLOW WORKFLOW2...",
		Short: "set values to zero or more workflows",
		Example: `# Set about a workflow:

  argo set my-wf

# Set the latest workflow:
  argo set @latest
`,
		Run: func(cmd *cobra.Command, args []string) {

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(setArgs.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", setArgs.nodeFieldSelector, err)
			}

			outputParams := make(map[string]string)
			err = json.Unmarshal([]byte(setArgs.outputParameters), &outputParams)
			if err != nil {
				log.Fatalf("unable to parse output parameter set request: %s", err)
			}

			for _, name := range args {
				wf, err := serviceClient.SetWorkflow(ctx, &workflowpkg.WorkflowSetRequest{
					Name:              name,
					Namespace:         namespace,
					NodeFieldSelector: selector.String(),
					Message:           setArgs.message,
					Phase:             setArgs.phase,
					OutputParameters:  setArgs.outputParameters,
				})
				errors.CheckError(err)
				fmt.Printf("workflow %s setped\n", wf.Name)
			}
		},
	}
	command.Flags().StringVar(&setArgs.message, "message", "", "Message to add to previously running nodes")
	command.Flags().StringVar(&setArgs.phase, "phase", "", "Phase to set node")
	command.Flags().StringVar(&setArgs.outputParameters, "output-parameters", "", "Output parameters to set in a JSON dict, eg: --output-parameters {\"hello\": \"world\"}")
	command.Flags().StringVar(&setArgs.nodeFieldSelector, "node-field-selector", "", "Selector of node to set, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	return command
}
