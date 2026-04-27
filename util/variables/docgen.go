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

// GenerateMarkdown renders the registered catalog as a self-contained
// Markdown document. Tables are aligned by tablewriter (via
// github.com/nao1215/markdown's CustomTable).
//
// Output sections:
//   1. Alphabetical list of every variable with its metadata.
//   2. Grouped by Kind (global, input, node-ref, …).
//   3. Matrix: rows = variables, columns = TemplateKinds; "•" indicates
//      the variable is in scope for that kind.
//   4. Grouped by LifecyclePhase.
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
	mdoc.PlainText("")
	writeByPhase(mdoc, all)

	mdoc.Build()
	return buf.String()
}

func writeFullTable(mdoc *md.Markdown, all []*Key) {
	rows := make([][]string, 0, len(all))
	for _, k := range all {
		rows = append(rows, []string{
			md.Code(k.template),
			k.kind.String(),
			k.valueType,
			joinPhases(k.phases),
			k.description,
		})
	}
	mdoc.CustomTable(
		md.TableSet{
			Header: []string{"Key", "Kind", "Type", "Availability", "Description"},
			Rows:   rows,
		},
		md.TableOptions{AutoWrapText: false},
	)
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
		rows := make([][]string, 0, len(groups[kd]))
		for _, k := range groups[kd] {
			rows = append(rows, []string{
				md.Code(k.template),
				k.valueType,
				joinPhases(k.phases),
				k.description,
			})
		}
		mdoc.CustomTable(
			md.TableSet{
				Header: []string{"Key", "Type", "Availability", "Description"},
				Rows:   rows,
			},
			md.TableOptions{AutoWrapText: false},
		)
	}
}

func writeMatrix(mdoc *md.Markdown, all []*Key) {
	cols := append([]TemplateKind{TmplAll}, AllTemplateKinds...)
	header := make([]string, 0, len(cols)+1)
	header = append(header, "Key")
	for _, c := range cols {
		header = append(header, string(c))
	}
	rows := make([][]string, 0, len(all))
	for _, k := range all {
		row := make([]string, 0, len(cols)+1)
		row = append(row, md.Code(k.template))
		hasAll := slices.Contains(k.appliesTo, TmplAll)
		for _, c := range cols {
			switch {
			case slices.Contains(k.appliesTo, c):
				row = append(row, "•")
			case hasAll && c != TmplAll:
				row = append(row, "•")
			default:
				row = append(row, "")
			}
		}
		rows = append(rows, row)
	}
	mdoc.CustomTable(
		md.TableSet{Header: header, Rows: rows},
		md.TableOptions{AutoWrapText: false},
	)
}

func writePhaseLegend(mdoc *md.Markdown) {
	mdoc.CustomTable(
		md.TableSet{
			Header: []string{"Phase", "Meaning"},
			Rows: [][]string{
				{string(PhWorkflowStart), "Globals populated once, up front, before any template runs."},
				{string(PhPreDispatch), "Immediately before a template's pod is created; pod.name / node.name / steps.name / tasks.name are set."},
				{string(PhDuringExecute), "Inside a template body; inputs.* are bound."},
				{string(PhInsideLoop), "Inside a withItems/withParam expansion; item, item.<key> are bound."},
				{string(PhInsideRetry), "Inside a retryStrategy template; retries.* are bound."},
				{string(PhAfterNodeInit), "A referenced node has been initialised (has an ID / phase). Earliest steps.X.id, steps.X.status."},
				{string(PhAfterPodStart), "The referenced node's pod has started; startedAt, ip, hostNodeName are populated."},
				{string(PhAfterNodeComplete), "The referenced node has finished (any terminal phase); finishedAt, exitCode are populated."},
				{string(PhAfterNodeSucceeded), "The referenced node has finished with Succeeded; outputs.result, outputs.parameters.*, outputs.artifacts.* are populated."},
				{string(PhAfterLoop), "Every child of a withItems/withParam group has completed; aggregated outputs appear."},
				{string(PhExitHandler), "The onExit template runs. workflow.{status,failures,duration} are final. Any earlier-phase variable is also visible here (scope accumulates)."},
			},
		},
		md.TableOptions{AutoWrapText: false},
	)
}

func writeByPhase(mdoc *md.Markdown, all []*Key) {
	phases := []LifecyclePhase{
		PhWorkflowStart, PhPreDispatch, PhDuringExecute,
		PhInsideLoop, PhInsideRetry,
		PhAfterNodeInit, PhAfterPodStart, PhAfterNodeComplete, PhAfterNodeSucceeded,
		PhAfterLoop, PhExitHandler,
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
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			rows = append(rows, []string{md.Code(k.template), k.kind.String(), k.valueType})
		}
		mdoc.CustomTable(
			md.TableSet{
				Header: []string{"Key", "Kind", "Type"},
				Rows:   rows,
			},
			md.TableOptions{AutoWrapText: false},
		)
	}
}

func joinPhases(ps []LifecyclePhase) string {
	if len(ps) == 0 {
		return ""
	}
	s := make([]string, len(ps))
	for i, p := range ps {
		s[i] = string(p)
	}
	return strings.Join(s, ", ")
}
