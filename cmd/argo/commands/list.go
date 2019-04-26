package commands

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/argoproj/pkg/humanize"
	argotime "github.com/argoproj/pkg/time"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

type listFlags struct {
	allNamespaces bool   // --all-namespaces
	status        string // --status
	completed     bool   // --completed
	running       bool   // --running
	output        string // --output
	since         string // --since
}

func NewListCommand() *cobra.Command {
	var (
		listArgs listFlags
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "list workflows",
		Run: func(cmd *cobra.Command, args []string) {
			var wfClient v1alpha1.WorkflowInterface
			if listArgs.allNamespaces {
				wfClient = InitWorkflowClient(apiv1.NamespaceAll)
			} else {
				wfClient = InitWorkflowClient()
			}
			listOpts := metav1.ListOptions{}
			labelSelector := labels.NewSelector()
			if listArgs.status != "" {
				req, _ := labels.NewRequirement(common.LabelKeyPhase, selection.In, strings.Split(listArgs.status, ","))
				labelSelector = labelSelector.Add(*req)
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
			wfList, err := wfClient.List(listOpts)
			if err != nil {
				log.Fatal(err)
			}
			var workflows []wfv1.Workflow
			if listArgs.since == "" {
				workflows = wfList.Items
			} else {
				workflows = make([]wfv1.Workflow, 0)
				minTime, err := argotime.ParseSince(listArgs.since)
				if err != nil {
					log.Fatal(err)
				}
				for _, wf := range wfList.Items {
					if wf.Status.FinishedAt.IsZero() || wf.ObjectMeta.CreationTimestamp.After(*minTime) {
						workflows = append(workflows, wf)
					}
				}
			}
			sort.Sort(ByFinishedAt(workflows))

			switch listArgs.output {
			case "", "wide":
				printTable(workflows, &listArgs)
			case "name":
				for _, wf := range workflows {
					fmt.Println(wf.ObjectMeta.Name)
				}
			default:
				log.Fatalf("Unknown output mode: %s", listArgs.output)
			}
		},
	}
	command.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "Show workflows from all namespaces")
	command.Flags().StringVar(&listArgs.status, "status", "", "Filter by status (comma separated)")
	command.Flags().BoolVar(&listArgs.completed, "completed", false, "Show only completed workflows")
	command.Flags().BoolVar(&listArgs.running, "running", false, "Show only running workflows")
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
	command.Flags().StringVar(&listArgs.since, "since", "", "Show only workflows newer than a relative duration")
	return command
}

func printTable(wfList []wfv1.Workflow, listArgs *listFlags) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		fmt.Fprint(w, "NAMESPACE\t")
	}
	fmt.Fprint(w, "NAME\tSTATUS\tAGE\tDURATION\tPRIORITY")
	if listArgs.output == "wide" {
		fmt.Fprint(w, "\tP/R/C\tPARAMETERS")
	}
	fmt.Fprint(w, "\n")
	for _, wf := range wfList {
		ageStr := humanize.RelativeDurationShort(wf.ObjectMeta.CreationTimestamp.Time, time.Now())
		durationStr := humanize.RelativeDurationShort(wf.Status.StartedAt.Time, wf.Status.FinishedAt.Time)
		if listArgs.allNamespaces {
			fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Namespace)
		}
		var priority int
		if wf.Spec.Priority != nil {
			priority = int(*wf.Spec.Priority)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d", wf.ObjectMeta.Name, workflowStatus(&wf), ageStr, durationStr, priority)
		if listArgs.output == "wide" {
			pending, running, completed := countPendingRunningCompleted(&wf)
			fmt.Fprintf(w, "\t%d/%d/%d", pending, running, completed)
			fmt.Fprintf(w, "\t%s", parameterString(wf.Spec.Arguments.Parameters))
		}
		fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}

func countPendingRunningCompleted(wf *wfv1.Workflow) (int, int, int) {
	pending := 0
	running := 0
	completed := 0
	err := util.DecompressWorkflow(wf)
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range wf.Status.Nodes {
		tmpl := wf.GetTemplate(node.TemplateName)
		if tmpl == nil || !tmpl.IsPodType() {
			continue
		}
		if node.Completed() {
			completed++
		} else if node.Phase == wfv1.NodeRunning {
			running++
		} else {
			pending++
		}
	}
	return pending, running, completed
}

// parameterString returns a human readable display string of the parameters, truncating if necessary
func parameterString(params []wfv1.Parameter) string {
	truncateString := func(str string, num int) string {
		bnoden := str
		if len(str) > num {
			if num > 3 {
				num -= 3
			}
			bnoden = str[0:num-15] + "..." + str[len(str)-15:]
		}
		return bnoden
	}

	pStrs := make([]string, 0)
	for _, p := range params {
		if p.Value != nil {
			str := fmt.Sprintf("%s=%s", p.Name, truncateString(*p.Value, 50))
			pStrs = append(pStrs, str)
		}
	}
	return strings.Join(pStrs, ",")
}

// ByFinishedAt is a sort interface which sorts running jobs earlier before considering FinishedAt
type ByFinishedAt []wfv1.Workflow

func (f ByFinishedAt) Len() int      { return len(f) }
func (f ByFinishedAt) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f ByFinishedAt) Less(i, j int) bool {
	iStart := f[i].ObjectMeta.CreationTimestamp
	iFinish := f[i].Status.FinishedAt
	jStart := f[j].ObjectMeta.CreationTimestamp
	jFinish := f[j].Status.FinishedAt
	if iFinish.IsZero() && jFinish.IsZero() {
		return !iStart.Before(&jStart)
	}
	if iFinish.IsZero() && !jFinish.IsZero() {
		return true
	}
	if !iFinish.IsZero() && jFinish.IsZero() {
		return false
	}
	return jFinish.Before(&iFinish)
}

// workflowStatus returns a human readable inferred workflow status based on workflow phase and conditions
func workflowStatus(wf *wfv1.Workflow) wfv1.NodePhase {
	switch wf.Status.Phase {
	case wfv1.NodeRunning:
		if util.IsWorkflowSuspended(wf) {
			return "Running (Suspended)"
		}
		return wf.Status.Phase
	case wfv1.NodeFailed:
		if util.IsWorkflowTerminated(wf) {
			return "Failed (Terminated)"
		}
		return wf.Status.Phase
	case "", wfv1.NodePending:
		if !wf.ObjectMeta.CreationTimestamp.IsZero() {
			return wfv1.NodePending
		}
		return "Unknown"
	default:
		return wf.Status.Phase
	}
}
