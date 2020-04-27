package commands

import (
	"log"
	"os"
	"sort"
	"strings"

	"github.com/argoproj/pkg/errors"
	argotime "github.com/argoproj/pkg/time"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/printer"
	"github.com/argoproj/argo/workflow/common"
)

type listFlags struct {
	allNamespaces bool     // --all-namespaces
	status        []string // --status
	completed     bool     // --completed
	running       bool     // --running
	prefix        string   // --prefix
	output        string   // --output
	since         string   // --since
	limit         int64    // --limit
	noHeaders     bool     // --no-headers
}

func NewListCommand() *cobra.Command {
	var (
		listArgs listFlags
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "list workflows",
		Run: func(cmd *cobra.Command, args []string) {
			listOpts := metav1.ListOptions{
				Limit: listArgs.limit,
			}
			labelSelector := labels.NewSelector()
			if len(listArgs.status) != 0 {
				req, _ := labels.NewRequirement(common.LabelKeyPhase, selection.In, listArgs.status)
				if req != nil {
					labelSelector = labelSelector.Add(*req)
				}
			}
			if listArgs.completed {
				req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"true"})
				labelSelector = labelSelector.Add(*req)
			}
			if listArgs.running {
				req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
				labelSelector = labelSelector.Add(*req)
			}
			listOpts.LabelSelector = labelSelector.String()

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			if listArgs.allNamespaces {
				namespace = ""
			}

			wfList, err := serviceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
				Namespace:   namespace,
				ListOptions: &listOpts,
			})
			errors.CheckError(err)

			tmpWorkFlows := wfList.Items
			for wfList.ListMeta.Continue != "" {
				listOpts.Continue = wfList.ListMeta.Continue
				wfList, err := serviceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
					Namespace:   namespace,
					ListOptions: &listOpts,
				})
				if err != nil {
					log.Fatal(err)
				}
				tmpWorkFlows = append(tmpWorkFlows, wfList.Items...)
			}

			var tmpWorkFlowsSelected []wfv1.Workflow
			if listArgs.prefix == "" {
				tmpWorkFlowsSelected = tmpWorkFlows
			} else {
				tmpWorkFlowsSelected = make([]wfv1.Workflow, 0)
				for _, wf := range tmpWorkFlows {
					if strings.HasPrefix(wf.ObjectMeta.Name, listArgs.prefix) {
						tmpWorkFlowsSelected = append(tmpWorkFlowsSelected, wf)
					}
				}
			}

			var workflows wfv1.Workflows
			if listArgs.since == "" {
				workflows = tmpWorkFlowsSelected
			} else {
				workflows = make(wfv1.Workflows, 0)
				minTime, err := argotime.ParseSince(listArgs.since)
				if err != nil {
					log.Fatal(err)
				}
				for _, wf := range tmpWorkFlowsSelected {
					if wf.Status.FinishedAt.IsZero() || wf.ObjectMeta.CreationTimestamp.After(*minTime) {
						workflows = append(workflows, wf)
					}
				}
			}
			sort.Sort(workflows)
			err = printer.PrintWorkflows(workflows, os.Stdout, printer.PrintOpts{
				NoHeaders: listArgs.noHeaders,
				Namespace: listArgs.allNamespaces,
				Output:    listArgs.output,
			})
			errors.CheckError(err)
		},
	}
	command.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "Show workflows from all namespaces")
	command.Flags().StringVar(&listArgs.prefix, "prefix", "", "Filter workflows by prefix")
	command.Flags().StringSliceVar(&listArgs.status, "status", []string{}, "Filter by status (comma separated)")
	command.Flags().BoolVar(&listArgs.completed, "completed", false, "Show only completed workflows")
	command.Flags().BoolVar(&listArgs.running, "running", false, "Show only running workflows")
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
	command.Flags().StringVar(&listArgs.since, "since", "", "Show only workflows newer than a relative duration")
	command.Flags().Int64Var(&listArgs.limit, "limit", 0, "Limit the total number of items returned.")
	command.Flags().BoolVar(&listArgs.noHeaders, "no-headers", false, "Don't print headers (default print headers).")
	return command
}
