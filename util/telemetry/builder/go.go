package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func createMetricsListGo(filename string, metrics *metricsList) {
	writeMetricsListGo(filename, metrics)
	goFmtFile(filename)
}

func writeMetricsListGo(filename string, metrics *metricsList) {
	f, err := os.Create(filename)
	if err != nil {
		recordError(err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", generatedBanner)
	fmt.Fprintf(f, "//\n")
	fmt.Fprintf(f, "//go:generate go run ./builder --metricsListGo %s\n", filename)
	fmt.Fprintf(f, "package telemetry\n\n")
	for _, metric := range *metrics {
		fmt.Fprintf(f, "var Instrument%s = BuiltinInstrument{\n", metric.Name)
		fmt.Fprintf(f, "\tname: \"%s\",\n", metric.displayName())
		fmt.Fprintf(f, "\tdescription: \"%s\",\n", metric.Description)
		fmt.Fprintf(f, "\tunit: \"%s\",\n", metric.Unit)
		fmt.Fprintf(f, "\tinstType: %s,\n", metric.instrumentType())
		if len(metric.Attributes) > 0 {
			fmt.Fprintf(f, "\tattributes: []BuiltinAttribute{\n")
			for _, attrib := range metric.Attributes {
				fmt.Fprintf(f, "\t\t{\n\t\t\tname: Attrib%s,\n", attrib.Name)
				if attrib.Optional {
					fmt.Fprintf(f, "\t\t\toptional: true,\n")
				}
				fmt.Fprintf(f, "\t\t},\n")
			}
			fmt.Fprintf(f, "\t},\n")
		}
		if len(metric.DefaultBuckets) > 0 {
			fmt.Fprintf(f, "\tdefaultBuckets: []float64{\n")
			for _, bucket := range metric.DefaultBuckets {
				fmt.Fprintf(f, "\t\t%f,\n", bucket)
			}
			fmt.Fprintf(f, "\t},\n")
		}

		fmt.Fprintf(f, "}\n\n")
	}
}

func createAttributesGo(filename string, attributes *attributesList) {
	writeAttributesGo(filename, attributes)
	goFmtFile(filename)
}

func writeAttributesGo(filename string, attributes *attributesList) {
	f, err := os.Create(filename)
	if err != nil {
		recordError(err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", generatedBanner)
	fmt.Fprintf(f, "//\n")
	fmt.Fprintf(f, "//go:generate go run ./builder --attributesGo %s\n", filename)
	fmt.Fprintf(f, "package telemetry\n\nconst (\n")
	for _, attrib := range *attributes {
		fmt.Fprintf(f, "\tAttrib%s string = `%s`\n", attrib.Name, attrib.displayName())
	}
	fmt.Fprintf(f, ")\n")
}

func goFmtFile(filename string) {
	cmd := exec.Command("go", "fmt", filename)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		recordErrorString(fmt.Sprintf("%s: %s", "go fmt failed", stderr.String()))
	}
}
