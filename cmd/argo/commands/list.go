package commands

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "show workflows from all namespaces")
	listCmd.Flags().StringVar(&listArgs.status, "status", "", "filter by status (comma separated)")
	listCmd.Flags().BoolVar(&listArgs.completed, "completed", false, "show only completed workflows")
	listCmd.Flags().BoolVar(&listArgs.running, "running", false, "show only running workflows")
	listCmd.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
}

type listFlags struct {
	allNamespaces bool   // --all-namespaces
	status        string // --status
	completed     bool   // --completed
	running       bool   // --running
	output        string // --output
}

var listArgs listFlags

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list workflows",
	Run:   listWorkflows,
}

var timeMagnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "0s", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1s %s", DivBy: 1},
	{D: time.Minute, Format: "%ds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1m %s", DivBy: 1},
	{D: time.Hour, Format: "%dm %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1h %s", DivBy: 1},
	{D: humanize.Day, Format: "%dh %s", DivBy: time.Hour},
	{D: 2 * humanize.Day, Format: "1d %s", DivBy: 1},
}

func listWorkflows(cmd *cobra.Command, args []string) {
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
	switch listArgs.output {
	case "", "wide":
		printTable(wfList)
	case "name":
		for _, wf := range wfList.Items {
			fmt.Println(wf.ObjectMeta.Name)
		}
	default:
		log.Fatalf("Unknown output mode: %s", listArgs.output)
	}
}

func printTable(wfList *wfv1.WorkflowList) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		fmt.Fprint(w, "NAMESPACE\t")
	}
	fmt.Fprint(w, "NAME\tSTATUS\tAGE\tDURATION")
	if listArgs.output == "wide" {
		fmt.Fprint(w, "\tPARAMETERS")
	}
	fmt.Fprint(w, "\n")
	sort.Sort(ByFinishedAt(wfList.Items))
	for _, wf := range wfList.Items {
		cTime := time.Unix(wf.ObjectMeta.CreationTimestamp.Unix(), 0)
		ageStr := humanize.CustomRelTime(cTime, time.Now(), "", "", timeMagnitudes)
		durationStr := humanizeDurationShort(wf.Status.StartedAt, wf.Status.FinishedAt)
		if listArgs.allNamespaces {
			fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Namespace)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s", wf.ObjectMeta.Name, worklowStatus(&wf), ageStr, durationStr)
		if listArgs.output == "wide" {
			fmt.Fprintf(w, "\t%s", parameterString(wf.Spec.Arguments.Parameters))
		}
		fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
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

func worklowStatus(wf *wfv1.Workflow) wfv1.NodePhase {
	if wf.Status.Phase != "" {
		return wf.Status.Phase
	}
	if !wf.ObjectMeta.CreationTimestamp.IsZero() {
		return "Pending"
	}
	return "Unknown"
}
