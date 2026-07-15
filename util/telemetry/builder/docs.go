package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	md "github.com/nao1215/markdown"
)

func createMetricsDocs(filename string, metrics *metricsList, attribs *attributesList) error {
	input, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	const begin = "Generated documentation BEGIN"
	const end = "Generated documentation END"
	type stageType int
	const (
		beforeBegin stageType = iota
		seekingEnd
		finishing
	)
	stage := beforeBegin
	var cutfrom int
	for i, line := range lines {
		switch stage {
		case beforeBegin:
			if strings.Contains(line, begin) {
				stage = seekingEnd
				lines = slices.Insert(lines, i+1, metricsDocsLines(metrics, attribs))
				cutfrom = i + 2
			}
		case seekingEnd:
			if strings.Contains(line, end) {
				stage = finishing
				lines = slices.Delete(lines, cutfrom, i+1)
			}
		case finishing:
			// Do nothing
		}
	}
	if stage != finishing {
		return fmt.Errorf("Didn't successfully replace docs in %s", filename)
	}

	output := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(output), 0444)
	return err
}

func createTracingDocs(filename string, spans *spansList, attribs *attributesList) error {
	// TODO: AnyParent
	input, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	const begin = "Generated documentation BEGIN"
	const end = "Generated documentation END"
	type stageType int
	const (
		beforeBegin stageType = iota
		seekingEnd
		finishing
	)
	stage := beforeBegin
	var cutfrom int
	for i, line := range lines {
		switch stage {
		case beforeBegin:
			if strings.Contains(line, begin) {
				stage = seekingEnd
				lines = slices.Insert(lines, i+1, tracingDocsLines(spans, attribs))
				cutfrom = i + 2
			}
		case seekingEnd:
			if strings.Contains(line, end) {
				stage = finishing
				lines = slices.Delete(lines, cutfrom, i+1)
			}
		case finishing:
			// Do nothing
		}
	}
	if stage != finishing {
		return fmt.Errorf("Didn't successfully replace docs in %s", filename)
	}

	output := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(output), 0444)
	return err
}

func tracingDocsLines(spans *spansList, attribs *attributesList) string {
	var out bytes.Buffer
	outWriter := io.Writer(&out)
	markdown := md.NewMarkdown(outWriter)
	markdown.PlainText("")

	// Generate a Mermaid diagram for each trace
	for _, span := range *spans {
		if !span.Root {
			continue
		}
		markdown.H3(fmt.Sprintf("Trace: `%s`", span.Name))
		markdown.PlainText("")
		markdown.PlainText("```mermaid")
		markdown.PlainText("flowchart TD")
		for _, line := range generateMermaidSpanTree(span.Name, spans) {
			markdown.PlainText(line)
		}
		markdown.PlainText("```")
		markdown.PlainText("")
	}

	markdown.H3("Span Reference")
	markdown.PlainText("")
	for _, span := range *spans {
		markdown.H4(md.Code(span.displayName()))
		markdown.PlainText("")
		markdown.PlainTextf("%s.", span.Description)
		if span.ExtendedDescription != "" {
			markdown.PlainText(strings.Trim(span.ExtendedDescription, " \n\t\r"))
		}
		markdown.PlainText("")

		if len(span.Attributes) > 0 {
			rows := [][]string{}
			for _, spanAttrib := range span.Attributes {
				if attrib := getAttribByName(spanAttrib.Name, attribs); attrib != nil {
					name := md.Code(attrib.displayName())
					if spanAttrib.Optional {
						name = name + " (optional)"
					}
					rows = append(rows, []string{name, attrib.Description})
				}
			}

			markdown.CustomTable(md.TableSet{
				Header: []string{"attribute", "explanation"},
				Rows:   rows,
			}, md.TableOptions{AutoWrapText: false},
			)
		} else {
			markdown.PlainText("This span has no attributes.")
			markdown.PlainText("")
		}

		if span.Notes != "" {
			markdown.PlainText(strings.Trim(span.Notes, " \n\t\r"))
			markdown.PlainText("")
		}
	}
	markdown.Build()
	return strings.TrimSuffix(out.String(), "\n")
}

// generateMermaidSpanTree returns Mermaid flowchart lines showing the span hierarchy
func generateMermaidSpanTree(rootName string, spans *spansList) []string {
	var lines []string

	// Find all spans that belong to this trace (have rootName in their ancestry)
	spansInTrace := findSpansInTrace(rootName, spans)

	// Generate node definitions with links
	for _, s := range spansInTrace {
		lines = append(lines, fmt.Sprintf("    %s[<a href='#%s'>%s</a>]", s.Name, s.displayName(), s.displayName()))
	}

	// Generate edges
	for _, s := range spansInTrace {
		for _, parent := range s.Parents {
			lines = append(lines, fmt.Sprintf("    %s --> %s", parent, s.Name))
		}
	}

	return lines
}

func metricsDocsLines(metrics *metricsList, attribs *attributesList) string {
	var out bytes.Buffer
	outWriter := io.Writer(&out)
	markdown := md.NewMarkdown(outWriter)
	markdown.PlainText("")
	for _, metric := range *metrics {
		markdown.H4(md.Code(metric.displayName()))
		markdown.PlainText("")
		markdown.PlainTextf("%s.", metric.Description)
		if metric.ExtendedDescription != "" {
			markdown.PlainText(strings.Trim(metric.ExtendedDescription, " \n\t\r"))
		}
		markdown.PlainText("")

		if len(metric.Attributes) > 0 {
			rows := [][]string{}
			for _, metricAttrib := range metric.Attributes {
				if attrib := getAttribByName(metricAttrib.Name, attribs); attrib != nil {
					// Failure should already be recorded as an error
					name := md.Code(attrib.displayName())
					if metricAttrib.Optional {
						name = name + " (optional)"
					}
					rows = append(rows, []string{name, attrib.Description})
				}
			}

			markdown.CustomTable(md.TableSet{
				Header: []string{"attribute", "explanation"},
				Rows:   rows,
			}, md.TableOptions{AutoWrapText: false},
			)
		} else {
			markdown.PlainText("This metric has no attributes.")
			markdown.PlainText("")
		}
		if len(metric.DefaultBuckets) > 0 {
			buckets := ""
			for i, bucket := range metric.DefaultBuckets {
				if i != 0 {
					buckets = fmt.Sprintf("%s, ", buckets)
				}
				buckets = fmt.Sprintf("%s%g", buckets, bucket)
			}
			markdown.PlainTextf("Default bucket sizes: %s", buckets)
			markdown.PlainText("")
		}

		if metric.Notes != "" {
			markdown.PlainText(strings.Trim(metric.Notes, " \n\t\r"))
			markdown.PlainText("")
		}
	}
	markdown.Build()
	return strings.TrimSuffix(out.String(), "\n")
}
