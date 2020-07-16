package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

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
		Use:   "set WORKFLOW FIELD SET_TO",
		Short: "set values to zero or more workflows",
		Example: `# Set outputs to a node within a workflow:

  argo set my-wf outputs parameters parameter-name="Hello, world!" --node-field-selector displayName=approve

# Set the message of a node within a workflow:

  argo set my-wf message 'We did it!' --node-field-selector displayName=approve
`,
		Run: func(cmd *cobra.Command, args []string) {

			if len(args) < 3 {
				cmd.HelpFunc()(cmd, args)
			}

			switch args[1] {
			case "outputs", "output":
				switch args[2] {
				case "parameters", "parameter":
					outputParams := make(map[string]string)
					for _, param := range args[3:] {
						parts := strings.SplitN(param, "=", 2)
						if len(parts) != 2 {
							log.Fatalf("expected parameter of the form: NAME=VALUE. Received: %s", param)
						}
						unquoted, err := strconv.Unquote(parts[1])
						if err != nil {
							log.Fatalf("error unqoting value: %s", err)
						}
						outputParams[parts[0]] = unquoted
					}
					res, err := json.Marshal(outputParams)
					if err != nil {
						log.Fatalf("unable to parse output parameter set request: %s", err)
					}
					setArgs.outputParameters = string(res)
				default:
					log.Fatalf("must specify which outputs to set: 'argo set outputs parameters ...'")
				}
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

			_, err = serviceClient.SetWorkflow(ctx, &workflowpkg.WorkflowSetRequest{
				Name:              args[0],
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
