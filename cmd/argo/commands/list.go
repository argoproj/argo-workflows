package commands

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	wfclient "github.com/argoproj/argo/workflow/client"
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
	Short: "list commands",
	Run:   listWorkflows,
}

func listWorkflows(cmd *cobra.Command, args []string) {
	var wfClient *wfclient.WorkflowClient
	if listArgs.allNamespaces {
		wfClient = initWorkflowClient(apiv1.NamespaceAll)
	} else {
		wfClient = initWorkflowClient()
	}
	wfList, err := wfClient.ListWorkflows(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if listArgs.allNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tSTATUS\tAGE")
	} else {
		fmt.Fprintln(w, "NAME\tSTATUS\tAGE")
	}

	for _, wf := range wfList.Items {
		cTime := time.Unix(wf.ObjectMeta.CreationTimestamp.Unix(), 0)
		now := time.Now()
		hrTimeDiff := humanize.RelTime(cTime, now, "", "")
		if listArgs.allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", wf.ObjectMeta.Namespace, wf.ObjectMeta.Name, worklowStatus(&wf), hrTimeDiff)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n", wf.ObjectMeta.Name, worklowStatus(&wf), hrTimeDiff)
		}
	}
	w.Flush()
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
