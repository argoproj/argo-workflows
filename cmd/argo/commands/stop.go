package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type stopOps struct {
	message           string // --message
	nodeFieldSelector string // --node-field-selector
	namespace         string // --namespace
	labelSelector     string // --selector
	fieldSelector     string // --field-selector
	dryRun            bool   // --dry-run
}

// hasSelector returns true if the CLI arguments selects multiple workflows
func (o *stopOps) hasSelector() bool {
	if o.labelSelector != "" || o.fieldSelector != "" {
		return true
	}
	return false
}

func NewStopCommand() *cobra.Command {
	var stopArgs stopOps

	command := &cobra.Command{
		Use:   "stop WORKFLOW WORKFLOW2...",
		Short: "stop zero or more workflows allowing all exit handlers to run",
		Long:  "Stop a workflow but still run exit handlers.",
		Example: `# Stop a workflow:

  argo stop my-wf

# Stop the latest workflow:

  argo stop @latest

# Stop multiple workflows by label selector

  argo stop -l workflows.argoproj.io/test=true

# Stop multiple workflows by field selector

  argo stop --field-selector metadata.namespace=argo
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !stopArgs.hasSelector() {
				return errors.New("requires either selector or workflow")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient()
			stopArgs.namespace = client.Namespace()

			return stopWorkflows(ctx, serviceClient, stopArgs, args)
		},
	}
	command.Flags().StringVar(&stopArgs.message, "message", "", "Message to add to previously running nodes")
	command.Flags().StringVar(&stopArgs.nodeFieldSelector, "node-field-selector", "", "selector of node to stop, eg: --node-field-selector inputs.parameters.myparam.value=abc")
	command.Flags().StringVarP(&stopArgs.labelSelector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&stopArgs.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().BoolVar(&stopArgs.dryRun, "dry-run", false, "If true, only print the workflows that would be stopped, without stopping them.")
	return command
}

// stopWorkflows stops workflows by given stopArgs or workflow names
func stopWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, stopArgs stopOps, args []string) error {
	selector, err := fields.ParseSelector(stopArgs.nodeFieldSelector)
	if err != nil {
		return fmt.Errorf("unable to parse node field selector '%s': %s", stopArgs.nodeFieldSelector, err)
	}
	var wfs wfv1.Workflows
	if stopArgs.hasSelector() {
		wfs, err = listWorkflows(ctx, serviceClient, listFlags{
			namespace: stopArgs.namespace,
			fields:    stopArgs.fieldSelector,
			labels:    stopArgs.labelSelector,
		})
		if err != nil {
			return err
		}
	}

	for _, n := range args {
		wfs = append(wfs, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      n,
				Namespace: stopArgs.namespace,
			},
		})
	}

	stoppedNames := make(map[string]bool)
	for _, wf := range wfs {
		if _, ok := stoppedNames[wf.Name]; ok {
			// de-duplication in case there is a overlap between the selector and given workflow names
			continue
		}
		stoppedNames[wf.Name] = true

		if stopArgs.dryRun {
			fmt.Printf("workflow %s stopped (dry-run)\n", wf.Name)
			continue
		}
		wf, err := serviceClient.StopWorkflow(ctx, &workflowpkg.WorkflowStopRequest{
			Name:              wf.Name,
			Namespace:         wf.Namespace,
			NodeFieldSelector: selector.String(),
			Message:           stopArgs.message,
		})
		if err != nil {
			return err
		}
		fmt.Printf("workflow %s stopped\n", wf.Name)
	}
	return nil
}
