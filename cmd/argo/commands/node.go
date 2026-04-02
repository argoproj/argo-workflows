package commands

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

type setOps struct {
	message           string   // --message
	phase             string   // --phase
	outputParameters  []string // --output-parameters
	nodeFieldSelector string   // --node-field-selector
}

func NewNodeCommand() *cobra.Command {
	var setArgs setOps

	command := &cobra.Command{
		Use:   "node ACTION WORKFLOW FLAGS",
		Short: "perform action on a node in a workflow",
		Example: `# Set outputs to a node within a workflow:

  argo node set my-wf --output-parameter parameter-name="Hello, world!" --node-field-selector displayName=approve

# Set the message of a node within a workflow:

  argo node set my-wf --message "We did it!"" --node-field-selector displayName=approve
`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "set" {
				return fmt.Errorf("unknown action '%s'", args[0])
			}

			outputParameters := ""
			if len(setArgs.outputParameters) > 0 {
				outputParams := make(map[string]string)
				for _, param := range setArgs.outputParameters {
					parts := strings.SplitN(param, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("expected parameter of the form: NAME=VALUE. Received: %s", param)
					}
					unquoted, err := strconv.Unquote(parts[1])
					if err != nil {
						unquoted = parts[1]
					}
					outputParams[parts[0]] = unquoted
				}
				res, err := json.Marshal(outputParams)
				if err != nil {
					return fmt.Errorf("unable to parse output parameter set request: %w", err)
				}
				outputParameters = string(res)
			}

			ctx := cmd.Context()
			ctx, apiClient, err := client.NewAPIClient(ctx)
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx)
			namespace := client.Namespace(ctx)

			selector, err := fields.ParseSelector(setArgs.nodeFieldSelector)
			if err != nil {
				return fmt.Errorf("unable to parse node field selector '%s': %w", setArgs.nodeFieldSelector, err)
			}

			_, err = serviceClient.SetWorkflow(ctx, &workflowpkg.WorkflowSetRequest{
				Name:              args[1],
				Namespace:         namespace,
				NodeFieldSelector: selector.String(),
				Message:           setArgs.message,
				Phase:             setArgs.phase,
				OutputParameters:  outputParameters,
			})
			if err != nil {
				return err
			}
			fmt.Printf("workflow values set\n")
			return nil
		},
	}
	command.Flags().StringVar(&setArgs.nodeFieldSelector, "node-field-selector", "", "Selector of node to set, eg: --node-field-selector inputs.parameters.myparam.value=abc")
	command.Flags().StringVar(&setArgs.phase, "phase", "", "Phase to set the node to, eg: --phase Succeeded")
	command.Flags().StringArrayVarP(&setArgs.outputParameters, "output-parameter", "p", []string{}, "Set a \"supplied\" output parameter of node, eg: --output-parameter parameter-name=\"Hello, world!\"")
	command.Flags().StringVarP(&setArgs.message, "message", "m", "", "Set the message of a node, eg: --message \"Hello, world!\"")
	return command
}
