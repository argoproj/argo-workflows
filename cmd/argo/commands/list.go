package commands

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "show workflows from all namespaces")
}

type listFlags struct {
	allNamespaces bool // --all-namespaces
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
	wfList, err := wfClient.List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tSTATUS\tAGE\tDURATION")
	} else {
		fmt.Fprintln(w, "NAME\tSTATUS\tAGE\tDURATION")
	}

	for _, wf := range wfList.Items {
		cTime := time.Unix(wf.ObjectMeta.CreationTimestamp.Unix(), 0)
		ageStr := humanize.CustomRelTime(cTime, time.Now(), "", "", timeMagnitudes)
		durationStr := humanizeDurationShort(wf.Status.StartedAt, wf.Status.FinishedAt)
		if listArgs.allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", wf.ObjectMeta.Namespace, wf.ObjectMeta.Name, worklowStatus(&wf), ageStr, durationStr)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", wf.ObjectMeta.Name, worklowStatus(&wf), ageStr, durationStr)
		}
	}
	_ = w.Flush()
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
