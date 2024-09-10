package archive

import (
	"context"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	client "github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
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
		Example: `# Resubmit a workflow:

  argo archive resubmit uid

# Resubmit multiple workflows:

  argo archive resubmit uid another-uid

# Resubmit multiple workflows by label selector:

  argo archive resubmit -l workflows.argoproj.io/test=true

# Resubmit multiple workflows by field selector:

  argo archive resubmit --field-selector metadata.namespace=argo

# Resubmit and wait for completion:

  argo archive resubmit --wait uid

# Resubmit and watch until completion:

  argo archive resubmit --watch uid

# Resubmit and tail logs until completion:

  argo archive resubmit --log uid
`,
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.Priority = &resubmitOpts.priority
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient() // needed for wait watch or log flags
			archiveServiceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			resubmitOpts.namespace = client.Namespace()
			err = resubmitArchivedWorkflows(ctx, archiveServiceClient, serviceClient, resubmitOpts, cliSubmitOpts, args)
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

// resubmitArchivedWorkflows resubmits workflows by given resubmitOpts or workflow names
func resubmitArchivedWorkflows(ctx context.Context, archiveServiceClient workflowarchivepkg.ArchivedWorkflowServiceClient, serviceClient workflowpkg.WorkflowServiceClient, resubmitOpts resubmitOps, cliSubmitOpts common.CliSubmitOpts, args []string) error {
	var (
		wfs wfv1.Workflows
		err error
	)

	if resubmitOpts.hasSelector() {
		wfs, err = listArchivedWorkflows(ctx, archiveServiceClient, resubmitOpts.fieldSelector, resubmitOpts.labelSelector, 0)
		if err != nil {
			return err
		}
	}

	for _, uid := range args {
		wfs = append(wfs, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(uid),
				Namespace: resubmitOpts.namespace,
			},
		})
	}

	var lastResubmitted *wfv1.Workflow
	resubmittedUids := make(map[string]bool)

	for _, wf := range wfs {
		if _, ok := resubmittedUids[string(wf.UID)]; ok {
			// de-duplication in case there is an overlap between the selector and given workflow names
			continue
		}
		resubmittedUids[string(wf.UID)] = true

		lastResubmitted, err = archiveServiceClient.ResubmitArchivedWorkflow(ctx, &workflowarchivepkg.ResubmitArchivedWorkflowRequest{
			Uid:        string(wf.UID),
			Namespace:  wf.Namespace,
			Name:       wf.Name,
			Memoized:   resubmitOpts.memoized,
			Parameters: cliSubmitOpts.Parameters,
		})
		if err != nil {
			return err
		}
		printWorkflow(lastResubmitted, cliSubmitOpts.Output)
	}

	if len(resubmittedUids) == 1 {
		// watch or wait when there is only one workflow retried
		common.WaitWatchOrLog(ctx, serviceClient, lastResubmitted.Namespace, []string{lastResubmitted.Name}, cliSubmitOpts)
	}
	return nil
}
