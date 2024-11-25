package main

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
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
	fmt.Fprintf(&out, "\n")
	for _, metric := range *metrics {
		fmt.Fprintf(&out, "#### `%s`\n\n", metric.displayName())
		fmt.Fprintf(&out, "%s.\n", metric.Description)
		if metric.ExtendedDescription != "" {
			fmt.Fprintf(&out, "%s\n", strings.Trim(metric.ExtendedDescription, " \n\t\r"))
		}
		fmt.Fprintf(&out, "\n")

		if len(metric.Attributes) > 0 {
			fmt.Fprintf(&out, "| attribute | explanation |\n")
			fmt.Fprintf(&out, "|-----------|-------------|\n")
			for _, metricAttrib := range metric.Attributes {
				if attrib := getAttribByName(metricAttrib.Name, attribs); attrib != nil {
					// Failure should already be recorded as an error
					fmt.Fprintf(&out, "| `%s` | %s |\n", attrib.displayName(), attrib.Description)
				}
			}
			fmt.Fprintf(&out, "\n")
		} else {
			fmt.Fprintf(&out, "This metric has no attributes.\n\n")
		}
		if len(metric.DefaultBuckets) > 0 {
			fmt.Fprintf(&out, "Default bucket sizes: ")
			for i, bucket := range metric.DefaultBuckets {
				if i != 0 {
					fmt.Fprintf(&out, ", ")
				}
				fmt.Fprintf(&out, "%g", bucket)
			}
			fmt.Fprintf(&out, "\n")
		}

		if metric.Notes != "" {
			fmt.Fprintf(&out, "%s\n", strings.Trim(metric.Notes, " \n\t\r"))
			fmt.Fprintf(&out, "\n")
		}
	}
	return strings.TrimSuffix(out.String(), "\n")
}
