package main

import (
	"fmt"
	"os"
)

func createMetricsListGo(filename string, metrics *metricsList) error {
	err := writeMetricsListGo(filename, metrics)
	if err != nil {
		return err
	}
	return goFmtFile(filename)
}

func writeMetricsListGo(filename string, metrics *metricsList) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
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
				fmt.Fprintf(f, "\t\t{\n\t\t\tname: %s,\n", attrib.AttribName())
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
	return nil
}
