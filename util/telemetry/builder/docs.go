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

func createMetricsDocs(filename string, metrics *metricsList, attribs *attributesList) {
	input, err := os.ReadFile(filename)
	if err != nil {
		recordError(err)
		return
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
		recordErrorString(fmt.Sprintf("Didn't successfully replace docs in %s", filename))
		return
	}

	output := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(output), 0444)
	if err != nil {
		recordError(err)
		return
	}
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
					rows = append(rows, []string{md.Code(attrib.displayName()), attrib.Description})
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
		}

		if metric.Notes != "" {
			markdown.PlainText(strings.Trim(metric.Notes, " \n\t\r"))
			markdown.PlainText("")
		}
	}
	markdown.Build()
	return strings.TrimSuffix(out.String(), "\n")
}
