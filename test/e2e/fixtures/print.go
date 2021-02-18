package fixtures

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/TwinProduction/go-color"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var workflowPhaseIcon = map[wfv1.WorkflowPhase]string{
	wfv1.WorkflowPending:   color.Ize(color.Yellow, "◷"),
	wfv1.WorkflowRunning:   color.Ize(color.Blue, "●"),
	wfv1.WorkflowSucceeded: color.Ize(color.Green, "✔"),
	wfv1.WorkflowFailed:    color.Ize(color.Red, "✖"),
	wfv1.WorkflowError:     color.Ize(color.Red, "⚠"),
}

var nodePhaseIcon = map[wfv1.NodePhase]string{
	wfv1.NodePending:   color.Ize(color.Yellow, "◷"),
	wfv1.NodeRunning:   color.Ize(color.Blue, "●"),
	wfv1.NodeSucceeded: color.Ize(color.Green, "✔"),
	wfv1.NodeSkipped:   color.Ize(color.Gray, "○"),
	wfv1.NodeFailed:    color.Ize(color.Red, "✖"),
	wfv1.NodeError:     color.Ize(color.Red, "⚠"),
}

func printWorkflow(wf *wfv1.Workflow) {
	println(fmt.Sprintf("%-20s %s", "Name:", wf.Name))
	println(fmt.Sprintf("%-18s %v %s", "Phase:", workflowPhaseIcon[wf.Status.Phase], wf.Status.Phase))
	println(fmt.Sprintf("%-20s %s", "Message:", wf.Status.Message))
	println(fmt.Sprintf("%-20s %s", "Duration:", time.Since(wf.Status.StartedAt.Time).Truncate(time.Second)))
	println()

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	for _, n := range wf.Status.Nodes {
		_, _ = fmt.Fprintf(w, " %s %s\t%s\t%s\t%s\n", nodePhaseIcon[n.Phase], n.Name, n.TemplateName, time.Since(n.StartedAt.Time).Truncate(time.Second), n.Message)
	}
	_ = w.Flush()
	println()
}
