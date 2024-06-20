package archive

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
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
		cliSubmitOpts common.CliSubmitOpts
		retryOpts     retryOps
	)
	command := &cobra.Command{
		Use:   "retry [WORKFLOW...]",
		Short: "retry zero or more workflows",
		Example: `# Retry a workflow:

  argo archive retry uid

# Retry multiple workflows:

  argo archive retry uid another-uid

# Retry multiple workflows by label selector:

  argo archive retry -l workflows.argoproj.io/test=true

# Retry multiple workflows by field selector:

  argo archive retry --field-selector metadata.namespace=argo

# Retry and wait for completion:

  argo archive retry --wait uid

# Retry and watch until completion:

  argo archive retry --watch uid
		
# Retry and tail logs until completion:

  argo archive retry --log uid
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 && !retryOpts.hasSelector() {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			archiveServiceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			retryOpts.namespace = client.Namespace()

			err = retryArchivedWorkflows(ctx, archiveServiceClient, serviceClient, retryOpts, cliSubmitOpts, args)
			errors.CheckError(err)
		},
	}

	command.Flags().StringArrayVarP(&cliSubmitOpts.Parameters, "parameter", "p", []string{}, "input parameter to override on the original workflow spec")
	command.Flags().StringVarP(&cliSubmitOpts.Output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.Wait, "wait", "w", false, "wait for the workflow to complete, only works when a single workflow is retried")
	command.Flags().BoolVar(&cliSubmitOpts.Watch, "watch", false, "watch the workflow until it completes, only works when a single workflow is retried")
	command.Flags().BoolVar(&cliSubmitOpts.Log, "log", false, "log the workflow until it completes")
	command.Flags().BoolVar(&retryOpts.restartSuccessful, "restart-successful", false, "indicates to restart successful nodes matching the --node-field-selector")
	command.Flags().StringVar(&retryOpts.nodeFieldSelector, "node-field-selector", "", "selector of nodes to reset, eg: --node-field-selector inputs.paramaters.myparam.value=abc")
	command.Flags().StringVarP(&retryOpts.labelSelector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&retryOpts.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	return command
}

// retryArchivedWorkflows retries workflows by given retryArgs or workflow names
func retryArchivedWorkflows(ctx context.Context, archiveServiceClient workflowarchivepkg.ArchivedWorkflowServiceClient, serviceClient workflowpkg.WorkflowServiceClient, retryOpts retryOps, cliSubmitOpts common.CliSubmitOpts, args []string) error {
	selector, err := fields.ParseSelector(retryOpts.nodeFieldSelector)
	if err != nil {
		return fmt.Errorf("unable to parse node field selector '%s': %s", retryOpts.nodeFieldSelector, err)
	}
	var wfs wfv1.Workflows
	if retryOpts.hasSelector() {
		wfs, err = listArchivedWorkflows(ctx, archiveServiceClient, retryOpts.namespace, retryOpts.fieldSelector, retryOpts.labelSelector, 0)
		if err != nil {
			return err
		}
	}

	for _, uid := range args {
		wfs = append(wfs, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(uid),
				Namespace: retryOpts.namespace,
			},
		})
	}

	var lastRetried *wfv1.Workflow
	retriedUids := make(map[string]bool)
	for _, wf := range wfs {
		if _, ok := retriedUids[string(wf.UID)]; ok {
			// de-duplication in case there is an overlap between the selector and given workflow names
			continue
		}
		retriedUids[string(wf.UID)] = true

		lastRetried, err = archiveServiceClient.RetryArchivedWorkflow(ctx, &workflowarchivepkg.RetryArchivedWorkflowRequest{
			Uid:               string(wf.UID),
			Namespace:         wf.Namespace,
			Name:              wf.Name,
			RestartSuccessful: retryOpts.restartSuccessful,
			NodeFieldSelector: selector.String(),
			Parameters:        cliSubmitOpts.Parameters,
		})
		if err != nil {
			return err
		}
		printWorkflow(lastRetried, cliSubmitOpts.Output)
	}
	if len(retriedUids) == 1 {
		// watch or wait when there is only one workflow retried
		common.WaitWatchOrLog(ctx, serviceClient, lastRetried.Namespace, []string{lastRetried.Name}, cliSubmitOpts)
	}
	return nil
}
