package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/argoproj/pkg/humanize"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
)

func PrintWorkflows(workflows wfv1.Workflows, out io.Writer, opts PrintOpts) error {
	if len(workflows) == 0 {
		_, _ = fmt.Fprintln(out, "No workflows found")
		return nil
	}

	switch opts.Output {
	case "", "wide":
		printTable(workflows, out, opts)
	case "name":
		for _, wf := range workflows {
			_, _ = fmt.Fprintln(out, wf.ObjectMeta.Name)
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
}

func printTable(wfList []wfv1.Workflow, out io.Writer, opts PrintOpts) {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	if !opts.NoHeaders {
		if opts.Namespace {
			_, _ = fmt.Fprint(w, "NAMESPACE\t")
		}
		_, _ = fmt.Fprint(w, "NAME\tSTATUS\tAGE\tDURATION\tPRIORITY")
		if opts.Output == "wide" {
			_, _ = fmt.Fprint(w, "\tP/R/C\tPARAMETERS")
		}
		_, _ = fmt.Fprint(w, "\n")
	}
	for _, wf := range wfList {
		ageStr := humanize.RelativeDurationShort(wf.ObjectMeta.CreationTimestamp.Time, time.Now())
		durationStr := humanize.RelativeDurationShort(wf.Status.StartedAt.Time, wf.Status.FinishedAt.Time)
		if opts.Namespace {
			_, _ = fmt.Fprintf(w, "%s\t", wf.ObjectMeta.Namespace)
		}
		var priority int
		if wf.Spec.Priority != nil {
			priority = int(*wf.Spec.Priority)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d", wf.ObjectMeta.Name, WorkflowStatus(&wf), ageStr, durationStr, priority)
		if opts.Output == "wide" {
			pending, running, completed := countPendingRunningCompleted(&wf)
			_, _ = fmt.Fprintf(w, "\t%d/%d/%d", pending, running, completed)
			_, _ = fmt.Fprintf(w, "\t%s", parameterString(wf.Spec.Arguments.Parameters))
		}
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}

func countPendingRunningCompleted(wf *wfv1.Workflow) (int, int, int) {
	pending := 0
	running := 0
	completed := 0
	for _, node := range wf.Status.Nodes {
		tmpl := wf.GetTemplateByName(node.TemplateName)
		if tmpl == nil || !tmpl.IsPodType() {
			continue
		}
		if node.Fulfilled() {
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
			str := fmt.Sprintf("%s=%s", p.Name, truncateString(p.Value.String(), 50))
			pStrs = append(pStrs, str)
		}
	}
	return strings.Join(pStrs, ",")
}

// WorkflowStatus returns a human readable inferred workflow status based on workflow phase and conditions
func WorkflowStatus(wf *wfv1.Workflow) wfv1.NodePhase {
	switch wf.Status.Phase {
	case wfv1.NodeRunning:
		if util.IsWorkflowSuspended(wf) {
			return "Running (Suspended)"
		}
		return wf.Status.Phase
	case wfv1.NodeFailed:
		if wf.Spec.Shutdown != "" {
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
