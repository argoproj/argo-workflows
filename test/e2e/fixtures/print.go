package fixtures

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/TwiN/go-color"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

var workflowPhaseIcon = map[wfv1.WorkflowPhase]string{
	"":                     color.Ize(color.Gray, "?"),
	wfv1.WorkflowPending:   color.Ize(color.Yellow, "◷"),
	wfv1.WorkflowRunning:   color.Ize(color.Blue, "●"),
	wfv1.WorkflowSucceeded: color.Ize(color.Green, "✔"),
	wfv1.WorkflowFailed:    color.Ize(color.Red, "✖"),
	wfv1.WorkflowError:     color.Ize(color.Red, "⚠"),
}

var nodePhaseIcon = map[wfv1.NodePhase]string{
	"":                 color.Ize(color.Gray, "?"),
	wfv1.NodePending:   color.Ize(color.Yellow, "◷"),
	wfv1.NodeRunning:   color.Ize(color.Blue, "●"),
	wfv1.NodeSucceeded: color.Ize(color.Green, "✔"),
	wfv1.NodeSkipped:   color.Ize(color.Gray, "○"),
	wfv1.NodeFailed:    color.Ize(color.Red, "✖"),
	wfv1.NodeError:     color.Ize(color.Red, "⚠"),
}

func printWorkflow(wf *wfv1.Workflow) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 1, ' ', 0)
	_, _ = fmt.Fprintf(w, " %s %s\t%s\t%v\t%s\n", workflowPhaseIcon[wf.Status.Phase], wf.Name, "Workflow", wf.Status.GetDuration(), wf.Status.Message)
	var nodes []wfv1.NodeStatus
	for _, n := range wf.Status.Nodes {
		nodes = append(nodes, n)
	}
	// a somewhat stable ordering, not perfect, but probably good enough
	sort.Slice(nodes, func(i, j int) bool {
		return !nodes[i].StartedAt.IsZero() && nodes[i].StartedAt.Time.Before(nodes[j].StartedAt.Time)
	})
	for _, n := range nodes {
		_, _ = fmt.Fprintf(w, " └ %s %s\t%s\t%v\t%s\n", nodePhaseIcon[n.Phase], n.DisplayName, n.Type, n.GetDuration(), n.Message)
	}
	_, _ = fmt.Fprintln(w)
	_ = w.Flush()
}
