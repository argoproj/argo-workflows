package archive

import (
	"context"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/printer"
)

func NewListCommand() *cobra.Command {
	var (
		selector  string
		output    = common.NewPrintWorkflowOutputValue("wide")
		chunkSize int64
		prefix  string
	)
	command := &cobra.Command{
		Use:   "list",
		Short: "list workflows in the archive",
		Example: `# List all archived workflows:
  argo archive list

# List all archived workflows fetched in chunks of 100:
  argo archive list --chunk-size 100

# List all archived workflows in YAML format:
  argo archive list -o yaml

# List archived workflows that have both labels:
  argo archive list -l key1=value1,key2=value2
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient, err := client.NewAPIClient(cmd.Context())
			if err != nil {
				return err
			}
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			if err != nil {
				return err
			}
			namespace := client.Namespace()
			workflows, err := listArchivedWorkflows(ctx, serviceClient, namespace, selector, chunkSize, prefix)
			if err != nil {
				return err
			}
			return printer.PrintWorkflows(workflows, os.Stdout, printer.PrintOpts{Output: output.String(), Namespace: true, UID: true})
		},
	}
	command.Flags().VarP(&output, "output", "o", "Output format. "+output.Usage())
	command.Flags().StringVar(&prefix, "prefix", "", "Filter workflows by prefix")
	command.Flags().StringVarP(&selector, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().Int64VarP(&chunkSize, "chunk-size", "", 0, "Return large lists in chunks rather than all at once. Pass 0 to disable.")
	return command
}

func listArchivedWorkflows(ctx context.Context, serviceClient workflowarchivepkg.ArchivedWorkflowServiceClient, namespace string, labelSelector string, chunkSize int64, prefix string) (wfv1.Workflows, error) {
	listOpts := &metav1.ListOptions{
		LabelSelector: labelSelector,
		Limit:         chunkSize,
	}
	var workflows wfv1.Workflows
	for {
		log.WithField("listOpts", listOpts).Debug()
		resp, err := serviceClient.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{Namespace: namespace, ListOptions: listOpts, NamePrefix: prefix})
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, resp.Items...)
		if resp.Continue == "" {
			break
		}
		listOpts.Continue = resp.Continue
	}
	sort.Sort(workflows)

	return workflows, nil
}
