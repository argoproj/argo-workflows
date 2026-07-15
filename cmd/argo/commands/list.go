package commands

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"

	argotime "github.com/argoproj/pkg/time"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	cmdcommon "github.com/argoproj/argo-workflows/v4/cmd/argo/commands/common"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/printer"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type listFlags struct {
	namespace      string
	status         []string
	completed      bool
	running        bool
	resubmitted    bool
	prefix         string
	output         cmdcommon.EnumFlagValue
	createdSince   string
	finishedBefore string
	chunkSize      int64
	noHeaders      bool
	labels         string
	fields         string
}

var (
	// finishedAt and creationTimestamp must be included to have a consistent display order of workflows
	nameFields    = "metadata,items.metadata.name,items.metadata.creationTimestamp,items.status.finishedAt"
	defaultFields = "metadata,items.metadata,items.spec,items.status.phase,items.status.message,items.status.finishedAt,items.status.startedAt,items.status.estimatedDuration,items.status.progress"
)

func (f listFlags) displayFields() string {
	switch f.output.String() {
	case "name":
		return nameFields
	case "json", "yaml", "wide":
		return ""
	default:
		return defaultFields
	}
}

func NewListCommand() *cobra.Command {
	var (
		listArgs      = listFlags{output: cmdcommon.NewPrintWorkflowOutputValue("")}
		allNamespaces bool
	)
	command := &cobra.Command{
		Use:   "list",
		Short: "list workflows",
		Example: `# List all workflows:
  argo list

# List all workflows from all namespaces:
  argo list -A

# List all running workflows:
  argo list --running

# List all completed workflows:
  argo list --completed

 # List workflows created within the last 10m:
  argo list --since 10m

# List workflows that finished more than 2h ago:
  argo list --older 2h

# List workflows with more information (such as parameters):
  argo list -o wide

# List workflows in YAML format:
  argo list -o yaml

# List workflows that have both labels:
  argo list -l label1=value1,label2=value2
`,

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, apiClient, err := client.NewAPIClient(ctx)
			if err != nil {
				return err
			}
			serviceClient := apiClient.NewWorkflowServiceClient(ctx)
			if !allNamespaces {
				listArgs.namespace = client.Namespace(ctx)
			}
			workflows, err := listWorkflows(ctx, serviceClient, listArgs)
			if err != nil {
				return err
			}
			return printer.PrintWorkflows(workflows, os.Stdout, printer.PrintOpts{
				NoHeaders: listArgs.noHeaders,
				Namespace: allNamespaces,
				Output:    listArgs.output.String(),
			})
		},
	}
	command.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Show workflows from all namespaces")
	command.Flags().StringVar(&listArgs.prefix, "prefix", "", "Filter workflows by prefix")
	command.Flags().StringVar(&listArgs.finishedBefore, "older", "", "List completed workflows finished before the specified duration (e.g. 10m, 3h, 1d)")
	command.Flags().StringSliceVar(&listArgs.status, "status", []string{}, "Filter by status (comma separated)")
	command.Flags().BoolVar(&listArgs.completed, "completed", false, "Show completed workflows. Mutually exclusive with --running.")
	command.Flags().BoolVar(&listArgs.running, "running", false, "Show running workflows. Mutually exclusive with --completed.")
	command.Flags().BoolVar(&listArgs.resubmitted, "resubmitted", false, "Show resubmitted workflows")
	command.Flags().VarP(&listArgs.output, "output", "o", "Output format. "+listArgs.output.Usage())
	command.Flags().StringVar(&listArgs.createdSince, "since", "", "Show only workflows created after than a relative duration")
	command.Flags().Int64VarP(&listArgs.chunkSize, "chunk-size", "", 0, "Return large lists in chunks rather than all at once. Pass 0 to disable.")
	command.Flags().BoolVar(&listArgs.noHeaders, "no-headers", false, "Don't print headers (default print headers).")
	command.Flags().StringVarP(&listArgs.labels, "selector", "l", "", "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	command.Flags().StringVar(&listArgs.fields, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	return command
}

func listWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, flags listFlags) (wfv1.Workflows, error) {
	listOpts := &metav1.ListOptions{
		Limit: flags.chunkSize,
	}
	labelSelector, err := labels.Parse(flags.labels)
	if err != nil {
		return nil, err
	}
	if len(flags.status) != 0 {
		req, _ := labels.NewRequirement(common.LabelKeyPhase, selection.In, flags.status)
		if req != nil {
			labelSelector = labelSelector.Add(*req)
		}
	}
	if flags.completed && flags.running {
		return nil, errors.New("--completed and --running cannot be used together")
	}
	if flags.completed {
		req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"true"})
		labelSelector = labelSelector.Add(*req)
	}
	if flags.running {
		req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
		labelSelector = labelSelector.Add(*req)
	}
	if flags.resubmitted {
		req, _ := labels.NewRequirement(common.LabelKeyPreviousWorkflowName, selection.Exists, []string{})
		labelSelector = labelSelector.Add(*req)
	}
	listOpts.LabelSelector = labelSelector.String()
	listOpts.FieldSelector = flags.fields
	var workflows wfv1.Workflows
	for {
		logging.RequireLoggerFromContext(ctx).WithField("listOpts", listOpts).Debug(ctx, "Listing workflows")
		wfList, err := serviceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
			Namespace:   flags.namespace,
			ListOptions: listOpts,
			Fields:      flags.displayFields(),
		})
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, wfList.Items...)
		if wfList.Continue == "" {
			break
		}
		listOpts.Continue = wfList.Continue
	}
	workflows = workflows.
		Filter(func(wf wfv1.Workflow) bool {
			return strings.HasPrefix(wf.Name, flags.prefix)
		})
	if flags.createdSince != "" && flags.finishedBefore != "" {
		startTime, err := argotime.ParseSince(flags.createdSince)
		if err != nil {
			return nil, err
		}
		endTime, err := argotime.ParseSince(flags.finishedBefore)
		if err != nil {
			return nil, err
		}
		workflows = workflows.Filter(wfv1.WorkflowRanBetween(*startTime, *endTime))
	} else {
		if flags.createdSince != "" {
			t, err := argotime.ParseSince(flags.createdSince)
			if err != nil {
				return nil, err
			}
			workflows = workflows.Filter(wfv1.WorkflowCreatedAfter(*t))
		}
		if flags.finishedBefore != "" {
			t, err := argotime.ParseSince(flags.finishedBefore)
			if err != nil {
				return nil, err
			}
			workflows = workflows.Filter(wfv1.WorkflowFinishedBefore(*t))
		}
	}
	sort.Sort(workflows)
	return workflows, nil
}
