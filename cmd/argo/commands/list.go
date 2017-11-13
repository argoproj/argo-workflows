package commands

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list commands",
	Run:   listWorkflows,
}

func listWorkflows(cmd *cobra.Command, args []string) {
	wfClient := initWorkflowClient()
	wfList, err := wfClient.ListWorkflows(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tAGE")
	for _, wf := range wfList.Items {
		cTime := time.Unix(wf.ObjectMeta.CreationTimestamp.Unix(), 0)
		now := time.Now()
		hrTimeDiff := humanize.RelTime(cTime, now, "", "")
		fmt.Fprintf(w, "%s\t%s\t%s\n", wf.ObjectMeta.Name, worklowStatus(&wf), hrTimeDiff)
	}
	w.Flush()
}

func worklowStatus(wf *wfv1.Workflow) string {
	if wf.Status.Nodes != nil {
		node, ok := wf.Status.Nodes[wf.ObjectMeta.Name]
		if ok {
			return node.Status
		}
	}
	if !wf.ObjectMeta.CreationTimestamp.IsZero() {
		return "Created"
	}
	return "Unknown"
}
