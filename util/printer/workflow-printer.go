package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/humanize"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func PrintWorkflows(workflows wfv1.Workflows, out io.Writer, opts PrintOpts) error {
	if len(workflows) == 0 {
		if opts.Output == "json" || opts.Output == "yaml" {
			_, _ = fmt.Fprintln(out, "[]")
		} else {
			_, _ = fmt.Fprintln(out, "No workflows found")
		}
		return nil
	}

	switch opts.Output {
	case "", "wide":
		printTable(workflows, out, opts)
		printCostOptimizationNudges(workflows, out)
	case "name":
		for _, wf := range workflows {
			_, _ = fmt.Fprintln(out, wf.Name)
		}
	case "json":
		output, err := json.MarshalIndent(workflows, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, string(output))
	case "yaml":
		output, err := yaml.Marshal(workflows)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, string(output))
	default:
		return fmt.Errorf("unknown output mode: %s", opts.Output)
	}
	return nil
}

type PrintOpts struct {
	NoHeaders bool
	Namespace bool
	Output    string
	UID       bool
}

func printTable(wfList []wfv1.Workflow, out io.Writer, opts PrintOpts) {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	if !opts.NoHeaders {
		if opts.Namespace {
			_, _ = fmt.Fprint(w, "NAMESPACE\t")
		}
		_, _ = fmt.Fprint(w, "NAME\tSTATUS\tAGE\tDURATION\tPRIORITY\tMESSAGE")
		if opts.Output == "wide" {
			_, _ = fmt.Fprint(w, "\tP/R/C\tPARAMETERS")
		}
		if opts.UID {
			_, _ = fmt.Fprint(w, "\tUID")
		}
		_, _ = fmt.Fprint(w, "\n")
	}
	for _, wf := range wfList {
		ageStr := humanize.RelativeDurationShort(wf.CreationTimestamp.Time, time.Now())
		durationStr := humanize.RelativeDurationShort(wf.Status.StartedAt.Time, wf.Status.FinishedAt.Time)
		messageStr := wf.Status.Message
		if opts.Namespace {
			_, _ = fmt.Fprintf(w, "%s\t", wf.Namespace)
		}
		var priority int
		if wf.Spec.Priority != nil {
			priority = int(*wf.Spec.Priority)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s", wf.Name, WorkflowStatus(&wf), ageStr, durationStr, priority, messageStr)
		if opts.Output == "wide" {
			pending, running, completed := countPendingRunningCompletedNodes(&wf)
			_, _ = fmt.Fprintf(w, "\t%d/%d/%d", pending, running, completed)
			_, _ = fmt.Fprintf(w, "\t%s", parameterString(wf.Spec.Arguments.Parameters))
		}
		if opts.UID {
			_, _ = fmt.Fprintf(w, "\t%s", wf.UID)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}

// printCostOptimizationNudges prints cost optimization nudges for workflows
func printCostOptimizationNudges(wfList []wfv1.Workflow, out io.Writer) {
	completed, incomplete := countCompletedWorkflows(wfList)
	if completed > 100 || incomplete > 100 {
		_, _ = fmt.Fprint(out, "\nYou have at least ")
		if incomplete > 100 {
			_, _ = fmt.Fprintf(out, "%d incomplete ", incomplete)
		}
		if incomplete > 100 && completed > 100 {
			_, _ = fmt.Fprint(out, "and ")
		}
		if completed > 100 {
			_, _ = fmt.Fprintf(out, "%d completed ", completed)
		}
		_, _ = fmt.Fprintln(out, "workflows. Reducing the total number of workflows will reduce your costs.")
		_, _ = fmt.Fprintln(out, "Learn more at https://argo-workflows.readthedocs.io/en/latest/cost-optimisation/")
	}
}

// countCompletedWorkflows returns the number of completed and incomplete workflows
func countCompletedWorkflows(wfList []wfv1.Workflow) (int, int) {
	completed := 0
	incomplete := 0
	for _, wf := range wfList {
		if wf.Status.Phase.Completed() {
			completed++
		} else {
			incomplete++
		}
	}
	return completed, incomplete
}

// countPendingRunningCompletedNodes returns the number of pending, running and completed workflow nodes
func countPendingRunningCompletedNodes(wf *wfv1.Workflow) (int, int, int) {
	pending := 0
	running := 0
	completed := 0
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod {
			continue
		}
		switch {
		case node.Fulfilled():
			completed++
		case node.Phase == wfv1.NodeRunning:
			running++
		default:
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
			str := fmt.Sprintf("%s=%s", p.Name, truncateString(p.Value.String(), 50))
			pStrs = append(pStrs, str)
		}
	}
	return strings.Join(pStrs, ",")
}

// WorkflowStatus returns a human readable inferred workflow status based on workflow phase and conditions
func WorkflowStatus(wf *wfv1.Workflow) string {
	switch wf.Status.Phase {
	case wfv1.WorkflowRunning:
		if util.IsWorkflowSuspended(wf) {
			return "Running (Suspended)"
		}
	case wfv1.WorkflowFailed:
		if wf.Spec.Shutdown != "" {
			return "Failed (Terminated)"
		}
	case wfv1.WorkflowUnknown, wfv1.WorkflowPending:
		if !wf.CreationTimestamp.IsZero() {
			return "Pending"
		}
		return "Unknown"
	}
	return string(wf.Status.Phase)
}
