package archive

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"

	client "github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type retryOps struct {
	nodeFieldSelector string // --node-field-selector
	restartSuccessful bool   // --restart-successful
	namespace         string // --namespace
	labelSelector     string // --selector
	fieldSelector     string // --field-selector
	forceName         bool   // --name
	forceUID          bool   // --uid
}

// hasSelector returns true if the CLI arguments selects multiple workflows
func (o *retryOps) hasSelector() bool {
	if o.labelSelector != "" || o.fieldSelector != "" {
		return true
	}
	return false
}

func NewRetryCommand() *cobra.Command {
	var (
		cliSubmitOpts = common.NewCliSubmitOpts()
		retryOpts     retryOps
	)
	command := &cobra.Command{
		Use:   "retry [WORKFLOW...]",
		Short: "retry zero or more workflows",
		Example: `# Retry a workflow by name:

  argo archive retry my-workflow

# Retry a workflow by UID (auto-detected):

  argo archive retry a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11

# Retry multiple workflows:

  argo archive retry my-workflow another-workflow

# Retry multiple workflows by label selector:

  argo archive retry -l workflows.argoproj.io/test=true

# Retry multiple workflows by field selector:

  argo archive retry --field-selector metadata.namespace=argo

# Retry and wait for completion:

  argo archive retry --wait my-workflow

# Retry and watch until completion:

  argo archive retry --watch my-workflow
		
# Retry and tail logs until completion:

  argo archive retry --log my-workflow
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !retryOpts.hasSelector() {
				return errors.New("requires either selector or workflow")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx)
			archiveServiceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}
			retryOpts.namespace = client.Namespace(ctx)

			return retryArchivedWorkflows(ctx, archiveServiceClient, serviceClient, retryOpts, cliSubmitOpts, args)
		},
	}

	command.Flags().StringArrayVarP(&cliSubmitOpts.Parameters, "parameter", "p", []string{}, "input parameter to override on the original workflow spec")
	command.Flags().VarP(&cliSubmitOpts.Output, "output", "o", "Output format. "+cliSubmitOpts.Output.Usage())
	command.Flags().BoolVarP(&cliSubmitOpts.Wait, "wait", "w", false, "wait for the workflow to complete, only works when a single workflow is retried")
	command.Flags().BoolVar(&cliSubmitOpts.Watch, "watch", false, "watch the workflow until it completes, only works when a single workflow is retried")
	command.Flags().BoolVar(&cliSubmitOpts.Log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&retryOpts.restartSuccessful, "restart-successful", false, "indicates to restart successful nodes matching the --node-field-selector")
	command.Flags().StringVar(&retryOpts.nodeFieldSelector, "node-field-selector", "", "selector of nodes to reset, eg: --node-field-selector inputs.parameters.myparam.value=abc")
	command.Flags().StringVarP(&retryOpts.labelSelector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&retryOpts.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	command.Flags().BoolVar(&retryOpts.forceName, "name", false, "force the argument to be treated as a name")
	command.Flags().BoolVar(&retryOpts.forceUID, "uid", false, "force the argument to be treated as a UID")
	command.MarkFlagsMutuallyExclusive("name", "uid")
	return command
}

// retryArchivedWorkflows retries workflows by given retryArgs or workflow names/UIDs
func retryArchivedWorkflows(ctx context.Context, archiveServiceClient workflowarchivepkg.ArchivedWorkflowServiceClient, serviceClient workflowpkg.WorkflowServiceClient, retryOpts retryOps, cliSubmitOpts common.CliSubmitOpts, args []string) error {
	selector, err := fields.ParseSelector(retryOpts.nodeFieldSelector)
	if err != nil {
		return fmt.Errorf("unable to parse node field selector '%s': %w", retryOpts.nodeFieldSelector, err)
	}
	var wfs wfv1.Workflows
	if retryOpts.hasSelector() {
		wfs, err = listArchivedWorkflows(ctx, archiveServiceClient, retryOpts.fieldSelector, retryOpts.labelSelector, 0)
		if err != nil {
			return err
		}
	}

	// Add workflows from args - auto-detect UID vs NAME
	for _, identifier := range args {
		wf := wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: retryOpts.namespace,
			},
		}
		isUID := isUID(identifier)
		if retryOpts.forceUID {
			isUID = true
		} else if retryOpts.forceName {
			isUID = false
		}
		if isUID {
			wf.UID = types.UID(identifier)
		} else {
			wf.Name = identifier
		}
		wfs = append(wfs, wf)
	}

	var lastRetried *wfv1.Workflow
	retriedIdentifiers := make(map[string]bool)
	for _, wf := range wfs {
		// Use UID if available, otherwise use namespace/name for deduplication
		var identifier string
		if wf.UID != "" {
			identifier = "uid:" + string(wf.UID)
		} else {
			identifier = "name:" + wf.Namespace + "/" + wf.Name
		}

		if _, ok := retriedIdentifiers[identifier]; ok {
			// de-duplication in case there is an overlap between the selector and given workflow names
			continue
		}
		retriedIdentifiers[identifier] = true

		req := &workflowarchivepkg.RetryArchivedWorkflowRequest{
			Namespace:         wf.Namespace,
			RestartSuccessful: retryOpts.restartSuccessful,
			NodeFieldSelector: selector.String(),
			Parameters:        cliSubmitOpts.Parameters,
		}
		if wf.UID != "" {
			req.Uid = string(wf.UID)
		} else {
			req.Name = wf.Name
		}

		lastRetried, err = archiveServiceClient.RetryArchivedWorkflow(ctx, req)
		if err != nil {
			return err
		}
		printWorkflow(lastRetried, cliSubmitOpts.Output.String())
	}
	if len(retriedIdentifiers) == 1 {
		// watch or wait when there is only one workflow retried
		return common.WaitWatchOrLog(ctx, serviceClient, lastRetried.Namespace, []string{lastRetried.Name}, cliSubmitOpts)
	}
	return nil
}
