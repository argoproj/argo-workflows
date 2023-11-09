package commands

import (
	"context"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type resubmitOps struct {
	priority      int32  // --priority
	memoized      bool   // --memoized
	namespace     string // --namespace
	labelSelector string // --selector
	fieldSelector string // --field-selector
}

// hasSelector returns true if the CLI arguments selects multiple workflows
func (o *resubmitOps) hasSelector() bool {
	if o.labelSelector != "" || o.fieldSelector != "" {
		return true
	}
	return false
}

func NewResubmitCommand() *cobra.Command {
	var (
		resubmitOpts  resubmitOps
		cliSubmitOpts common.CliSubmitOpts
	)
	command := &cobra.Command{
		Use:   "resubmit [WORKFLOW...]",
		Short: "resubmit one or more workflows",
		Long:  "Submit a completed workflow again. Optionally override parameters and memoize. Similar to running `argo submit` again with the same parameters.",
		Example: `# Resubmit a workflow:

  argo resubmit my-wf

# Resubmit multiple workflows:

  argo resubmit my-wf my-other-wf my-third-wf

# Resubmit multiple workflows by label selector:

  argo resubmit -l workflows.argoproj.io/test=true

# Resubmit multiple workflows by field selector:

  argo resubmit --field-selector metadata.namespace=argo

# Resubmit and wait for completion:

  argo resubmit --wait my-wf.yaml

# Resubmit and watch until completion:

  argo resubmit --watch my-wf.yaml

# Resubmit and tail logs until completion:

  argo resubmit --log my-wf.yaml

# Resubmit the latest workflow:

  argo resubmit @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.Priority = &resubmitOpts.priority
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			resubmitOpts.namespace = client.Namespace()
			err := resubmitWorkflows(ctx, serviceClient, resubmitOpts, cliSubmitOpts, args)
			errors.CheckError(err)
		},
	}

	command.Flags().StringArrayVarP(&cliSubmitOpts.Parameters, "parameter", "p", []string{}, "input parameter to override on the original workflow spec")
	command.Flags().Int32Var(&resubmitOpts.priority, "priority", 0, "workflow priority")
	command.Flags().StringVarP(&cliSubmitOpts.Output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.Wait, "wait", "w", false, "wait for the workflow to complete, only works when a single workflow is resubmitted")
	command.Flags().BoolVar(&cliSubmitOpts.Watch, "watch", false, "watch the workflow until it completes, only works when a single workflow is resubmitted")
	command.Flags().BoolVar(&cliSubmitOpts.Log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&resubmitOpts.memoized, "memoized", false, "re-use successful steps & outputs from the previous run")
	command.Flags().StringVarP(&resubmitOpts.labelSelector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&resubmitOpts.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	return command
}

// resubmitWorkflows resubmits workflows by given resubmitOpts or workflow names
func resubmitWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, resubmitOpts resubmitOps, cliSubmitOpts common.CliSubmitOpts, args []string) error {
	var (
		wfs wfv1.Workflows
		err error
	)
	if resubmitOpts.hasSelector() {
		wfs, err = listWorkflows(ctx, serviceClient, listFlags{
			namespace: resubmitOpts.namespace,
			fields:    resubmitOpts.fieldSelector,
			labels:    resubmitOpts.labelSelector,
		})
		if err != nil {
			return err
		}
	}

	for _, n := range args {
		wfs = append(wfs, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      n,
				Namespace: resubmitOpts.namespace,
			},
		})
	}

	var lastResubmitted *wfv1.Workflow
	resubmittedNames := make(map[string]bool)

	for _, wf := range wfs {
		if _, ok := resubmittedNames[wf.Name]; ok {
			// de-duplication in case there is an overlap between the selector and given workflow names
			continue
		}
		resubmittedNames[wf.Name] = true

		lastResubmitted, err = serviceClient.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{
			Namespace:  wf.Namespace,
			Name:       wf.Name,
			Memoized:   resubmitOpts.memoized,
			Parameters: cliSubmitOpts.Parameters,
		})
		if err != nil {
			return err
		}
		printWorkflow(lastResubmitted, common.GetFlags{Output: cliSubmitOpts.Output})
	}
	if len(resubmittedNames) == 1 {
		// watch or wait when there is only one workflow retried
		common.WaitWatchOrLog(ctx, serviceClient, lastResubmitted.Namespace, []string{lastResubmitted.Name}, cliSubmitOpts)
	}
	return nil
}
