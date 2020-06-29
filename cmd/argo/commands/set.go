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
		Use:   "set FIELD WORKFLOW SET_TO",
		Short: "set values to zero or more workflows",
		Example: `# Set outputs to a node within a workflow:

  argo set outputs my-wf '{"parameter-name": "Hello, world!"}' --node-field-selector displayName=approve

# Set the message of a node within a workflow:

  argo set message my-wf 'We did it!' --node-field-selector displayName=approve
`,
		Run: func(cmd *cobra.Command, args []string) {

			if len(args) != 3 {
				cmd.HelpFunc()(cmd, args)
			}

			switch args[0] {
			case "outputs", "output":
				setArgs.outputParameters = args[2]
			case "message":
				setArgs.message = args[2]
			default:
				log.Fatalf("cannot set '%s'", args[0])
			}

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			selector, err := fields.ParseSelector(setArgs.nodeFieldSelector)
			if err != nil {
				log.Fatalf("Unable to parse node field selector '%s': %s", setArgs.nodeFieldSelector, err)
			}

			outputParams := make(map[string]string)
			if setArgs.outputParameters != "" {
				err = json.Unmarshal([]byte(setArgs.outputParameters), &outputParams)
				if err != nil {
					log.Fatalf("unable to parse output parameter set request: %s", err)
				}
			}

			_, err = serviceClient.SetWorkflow(ctx, &workflowpkg.WorkflowSetRequest{
				Name:              args[1],
				Namespace:         namespace,
				NodeFieldSelector: selector.String(),
				Message:           setArgs.message,
				Phase:             setArgs.phase,
				OutputParameters:  setArgs.outputParameters,
			})
			errors.CheckError(err)
			fmt.Printf("workflow values set\n")
		},
	}
	command.Flags().StringVar(&setArgs.nodeFieldSelector, "node-field-selector", "", "Selector of node to set, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	return command
}
