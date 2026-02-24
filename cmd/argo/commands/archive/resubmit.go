package archive

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	client "github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type resubmitOps struct {
	priority      int32  // --priority
	memoized      bool   // --memoized
	namespace     string // --namespace
	labelSelector string // --selector
	fieldSelector string // --field-selector
	forceName     bool   // --name
	forceUID      bool   // --uid
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
		cliSubmitOpts = common.NewCliSubmitOpts()
	)
	command := &cobra.Command{
		Use:   "resubmit [WORKFLOW...]",
		Short: "resubmit one or more workflows",
		Example: `# Resubmit a workflow by name:

  argo archive resubmit my-workflow

# Resubmit a workflow by UID (auto-detected):

  argo archive resubmit a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11

# Resubmit multiple workflows:

  argo archive resubmit my-workflow another-workflow

# Resubmit multiple workflows by label selector:

  argo archive resubmit -l workflows.argoproj.io/test=true

# Resubmit multiple workflows by field selector:

  argo archive resubmit --field-selector metadata.namespace=argo

# Resubmit and wait for completion:

  argo archive resubmit --wait my-workflow

# Resubmit and watch until completion:

  argo archive resubmit --watch my-workflow

# Resubmit and tail logs until completion:

  argo archive resubmit --log my-workflow

# Resubmit a workflow by name (forced):

  argo archive resubmit my-workflow --name

# Resubmit a workflow by UID (forced):

  argo archive resubmit a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11 --uid
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flag("priority").Changed {
				cliSubmitOpts.Priority = &resubmitOpts.priority
			}

			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx) // needed for wait watch or log flags
			archiveServiceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}
			resubmitOpts.namespace = client.Namespace(ctx)
			return resubmitArchivedWorkflows(ctx, archiveServiceClient, serviceClient, resubmitOpts, cliSubmitOpts, args)
		},
	}

	command.Flags().StringArrayVarP(&cliSubmitOpts.Parameters, "parameter", "p", []string{}, "input parameter to override on the original workflow spec")
	command.Flags().Int32Var(&resubmitOpts.priority, "priority", 0, "workflow priority")
	command.Flags().VarP(&cliSubmitOpts.Output, "output", "o", "Output format. "+cliSubmitOpts.Output.Usage())
	command.Flags().BoolVarP(&cliSubmitOpts.Wait, "wait", "w", false, "wait for the workflow to complete, only works when a single workflow is resubmitted")
	command.Flags().BoolVar(&cliSubmitOpts.Watch, "watch", false, "watch the workflow until it completes, only works when a single workflow is resubmitted")
	command.Flags().BoolVar(&cliSubmitOpts.Log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&resubmitOpts.memoized, "memoized", false, "re-use successful steps & outputs from the previous run")
	command.Flags().StringVarP(&resubmitOpts.labelSelector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&resubmitOpts.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().BoolVar(&resubmitOpts.forceName, "name", false, "force the argument to be treated as a name")
	command.Flags().BoolVar(&resubmitOpts.forceUID, "uid", false, "force the argument to be treated as a UID")
	command.MarkFlagsMutuallyExclusive("name", "uid")
	return command
}

// resubmitArchivedWorkflows resubmits workflows by given resubmitOpts or workflow names/UIDs
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

	// Add workflows from args - auto-detect UID vs NAME
	for _, identifier := range args {
		uid, err := resolveUID(ctx, archiveServiceClient, identifier, resubmitOpts.namespace, resubmitOpts.forceUID, resubmitOpts.forceName)
		if err != nil {
			return fmt.Errorf("resolve UID: %w", err)
		}
		wf := wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: resubmitOpts.namespace,
				UID:       types.UID(uid),
			},
		}
		wfs = append(wfs, wf)
	}

	var lastResubmitted *wfv1.Workflow
	resubmittedIdentifiers := make(map[string]bool)

	for _, wf := range wfs {
		// Use UID if available, otherwise use namespace/name for deduplication
		var identifier string
		if wf.UID != "" {
			identifier = "uid:" + string(wf.UID)
		} else {
			identifier = "name:" + wf.Namespace + "/" + wf.Name
		}

		if _, ok := resubmittedIdentifiers[identifier]; ok {
			// de-duplication in case there is an overlap between the selector and given workflow names
			continue
		}
		resubmittedIdentifiers[identifier] = true

		req := &workflowarchivepkg.ResubmitArchivedWorkflowRequest{
			Namespace:  wf.Namespace,
			Memoized:   resubmitOpts.memoized,
			Parameters: cliSubmitOpts.Parameters,
		}
		if wf.UID != "" {
			req.Uid = string(wf.UID)
		} else {
			req.Name = wf.Name
		}

		lastResubmitted, err = archiveServiceClient.ResubmitArchivedWorkflow(ctx, req)
		if err != nil {
			return err
		}
		printWorkflow(lastResubmitted, cliSubmitOpts.Output.String())
	}

	if len(resubmittedIdentifiers) == 1 {
		// watch or wait when there is only one workflow retried
		return common.WaitWatchOrLog(ctx, serviceClient, lastResubmitted.Namespace, []string{lastResubmitted.Name}, cliSubmitOpts)
	}
	return nil
}
