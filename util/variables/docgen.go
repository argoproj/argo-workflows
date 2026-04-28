package variables

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	md "github.com/nao1215/markdown"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// GenerateMarkdown renders the catalog as Markdown. Sections:
// alphabetical index, by Kind, matrix by TemplateKind, by LifecyclePhase.
func GenerateMarkdown() string {
	var buf bytes.Buffer
	mdoc := md.NewMarkdown(io.Writer(&buf))
	all := All()

	mdoc.H1("Workflow variables catalog")
	mdoc.PlainText("")
	mdoc.PlainTextf("Auto-generated from `util/variables` via `GenerateMarkdown()`. %d variables registered.", len(all))
	mdoc.PlainText("")

	mdoc.H2("1. Alphabetical index")
	mdoc.PlainText("")
	writeFullTable(mdoc, all)

	mdoc.H2("2. Grouped by Kind")
	mdoc.PlainText("")
	writeByKind(mdoc, all)

	mdoc.H2("3. Matrix by TemplateKind")
	mdoc.PlainText("")
	mdoc.PlainText("Which variables are in scope for each template type. `•` = in scope, blank = not in scope.")
	mdoc.PlainText("")
	writeMatrix(mdoc, all)

	mdoc.H2("4. Grouped by LifecyclePhase")
	mdoc.PlainText("")
	writePhaseLegend(mdoc)
	writeByPhase(mdoc, all)

	_ = mdoc.Build()
	return buf.String()
}

func writeFullTable(mdoc *md.Markdown, all []*Key) {
	rows := make([][]string, len(all))
	for i, k := range all {
		rows[i] = []string{md.Code(k.template), k.kind.String(), k.valueType, joinPhases(k.phases), k.description}
	}
	table(mdoc, []string{"Key", "Kind", "Type", "Availability", "Description"}, rows)
}

func writeByKind(mdoc *md.Markdown, all []*Key) {
	groups := map[Kind][]*Key{}
	var kinds []Kind
	for _, k := range all {
		if _, ok := groups[k.kind]; !ok {
			kinds = append(kinds, k.kind)
		}
		groups[k.kind] = append(groups[k.kind], k)
	}
	slices.Sort(kinds)
	titler := cases.Title(language.English)
	for _, kd := range kinds {
		mdoc.H3(titler.String(kd.String()))
		mdoc.PlainText("")
		rows := make([][]string, len(groups[kd]))
		for i, k := range groups[kd] {
			rows[i] = []string{md.Code(k.template), k.valueType, joinPhases(k.phases), k.description}
		}
		table(mdoc, []string{"Key", "Type", "Availability", "Description"}, rows)
	}
}

func writeMatrix(mdoc *md.Markdown, all []*Key) {
	cols := append([]TemplateKind{TmplAll}, AllTemplateKinds...)
	header := append([]string{"Key"}, kindStrings(cols)...)
	rows := make([][]string, len(all))
	for i, k := range all {
		row := make([]string, len(cols)+1)
		row[0] = md.Code(k.template)
		hasAll := slices.Contains(k.appliesTo, TmplAll)
		for j, c := range cols {
			if slices.Contains(k.appliesTo, c) || (hasAll && c != TmplAll) {
				row[j+1] = "•"
			}
		}
		rows[i] = row
	}
	table(mdoc, header, rows)
}

func writePhaseLegend(mdoc *md.Markdown) {
	table(mdoc, []string{"Phase", "Meaning"}, [][]string{
		{string(PhWorkflowStart), "Globals populated once, up front, before any template runs."},
		{string(PhPreDispatch), "Immediately before a template's pod is created; pod.name / node.name / steps.name / tasks.name are set."},
		{string(PhDuringExecute), "Inside a template body; inputs.* are bound."},
		{string(PhInsideLoop), "Inside a withItems/withParam expansion; `item`, `item.<key>` are bound."},
		{string(PhInsideRetry), "Inside a retryStrategy template; retries.* are bound."},
		{string(PhAfterNodeInit), "A referenced node has been initialised (has an ID / phase). Earliest steps.X.id, steps.X.status."},
		{string(PhAfterPodStart), "The referenced node's pod has started; startedAt, ip, hostNodeName are populated."},
		{string(PhAfterNodeComplete), "The referenced node has finished (any terminal phase); finishedAt, exitCode are populated."},
		{string(PhAfterNodeSucceeded), "The referenced node has finished with Succeeded; outputs.result, outputs.parameters.*, outputs.artifacts.* are populated."},
		{string(PhAfterLoop), "Every child of a withItems/withParam group has completed; aggregated outputs appear."},
		{string(PhExitHandler), "The onExit template runs. workflow.{status,failures,duration} are final. Any earlier-phase variable is also visible here (scope accumulates)."},
		{string(PhMetricEmission), "Inside a Prometheus metric expression. Adds duration, status, exitCode, resourcesDuration.<resource>, and the current node's bare outputs.result / outputs.parameters.<name>."},
		{string(PhCronEval), "Evaluating a CronWorkflow `spec.when` or `spec.stopStrategy.expression`. Adds cronworkflow.* variables describing the cron object's identity, labels/annotations, and run counts."},
	})
}

func writeByPhase(mdoc *md.Markdown, all []*Key) {
	phases := []LifecyclePhase{
		PhWorkflowStart, PhPreDispatch, PhDuringExecute,
		PhInsideLoop, PhInsideRetry,
		PhAfterNodeInit, PhAfterPodStart, PhAfterNodeComplete, PhAfterNodeSucceeded,
		PhAfterLoop, PhExitHandler, PhMetricEmission, PhCronEval,
	}
	for _, p := range phases {
		var keys []*Key
		for _, k := range all {
			if slices.Contains(k.phases, p) {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			continue
		}
		mdoc.H3(fmt.Sprintf("%s (%d variables)", p, len(keys)))
		mdoc.PlainText("")
		rows := make([][]string, len(keys))
		for i, k := range keys {
			rows[i] = []string{md.Code(k.template), k.kind.String(), k.valueType}
		}
		table(mdoc, []string{"Key", "Kind", "Type"}, rows)
	}
}

func table(mdoc *md.Markdown, header []string, rows [][]string) {
	mdoc.CustomTable(md.TableSet{Header: header, Rows: rows}, md.TableOptions{AutoWrapText: false})
}

func joinPhases(ps []LifecyclePhase) string {
	s := make([]string, len(ps))
	for i, p := range ps {
		s[i] = string(p)
	}
	return strings.Join(s, ", ")
}

func kindStrings(ks []TemplateKind) []string {
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = string(k)
	}
	return out
}
