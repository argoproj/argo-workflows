package main

import (
	"fmt"
	"os"
	"strings"
)

func createMetricsHelpersGo(filename string, metrics *metricsList, attributes *attributesList) {
	writeMetricsHelpersGo(filename, metrics, attributes)
	goFmtFile(filename)
}

func writeMetricsHelpersGo(filename string, metrics *metricsList, attributes *attributesList) {
	f, err := os.Create(filename)
	if err != nil {
		recordError(err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "%s\n", generatedBanner)
	fmt.Fprintf(f, "package telemetry\n\n")
	fmt.Fprintf(f, "import (\n")
	fmt.Fprintf(f, "\t\"context\"\n\n")
	fmt.Fprintf(f, "\t\"go.opentelemetry.io/otel/metric\"\n")
	fmt.Fprintf(f, ")\n\n")

	// Generate option types and functions for metrics with optional attributes
	generatedOptions := make(map[string]bool)
	for _, metric := range *metrics {
		if hasOptionalAttributes(&metric) {
			optionTypeName := fmt.Sprintf("%sOption", metric.Name)
			if !generatedOptions[optionTypeName] {
				generateOptionType(f, &metric, optionTypeName, attributes)
				generatedOptions[optionTypeName] = true
			}
		}
	}

	// Generate helper methods for each metric
	for _, metric := range *metrics {
		switch metric.Type {
		case "Int64Counter", "Int64UpDownCounter":
			generateCounterHelper(f, &metric, attributes)
		case "Float64Histogram":
			generateHistogramHelper(f, &metric, attributes)
		case "Int64ObservableGauge", "Float64ObservableGauge":
			generateObservableHelper(f, &metric, attributes)
		}
	}
}

func hasOptionalAttributes(m *metric) bool {
	for _, attr := range m.Attributes {
		if attr.Optional {
			return true
		}
	}
	return false
}

func generateOptionType(f *os.File, m *metric, optionTypeName string, attributes *attributesList) {
	// Generate the option type
	fmt.Fprintf(f, "// %s is a functional option for configuring optional attributes on %s\n", optionTypeName, m.Name)
	fmt.Fprintf(f, "type %s func(*InstAttribs)\n\n", optionTypeName)

	// Generate option functions for each optional attribute
	for _, attr := range m.Attributes {
		if attr.Optional {
			attribDef := getAttribByName(attr.Name, attributes)
			paramName := toLowerCamelCase(attr.Name)
			paramType := attribDef.attrType()
			funcName := fmt.Sprintf("With%s", attr.Name)

			fmt.Fprintf(f, "// %s sets the %s attribute\n", funcName, attribDef.displayName())
			fmt.Fprintf(f, "func %s(%s %s) %s {\n", funcName, paramName, paramType, optionTypeName)
			fmt.Fprintf(f, "\treturn func(a *InstAttribs) {\n")
			fmt.Fprintf(f, "\t\t*a = append(*a, InstAttrib{\n")
			fmt.Fprintf(f, "\t\t\tName:  Attrib%s,\n", attr.Name)
			fmt.Fprintf(f, "\t\t\tValue: %s,\n", paramName)
			fmt.Fprintf(f, "\t\t})\n")
			fmt.Fprintf(f, "\t}\n")
			fmt.Fprintf(f, "}\n\n")
		}
	}
}

func buildParameterList(baseParams []string, m *metric, attributes *attributesList) []string {
	params := baseParams
	requiredAttrs := getRequiredAttributes(m)

	for _, attr := range requiredAttrs {
		attribDef := getAttribByName(attr.Name, attributes)
		paramName := toLowerCamelCase(attr.Name)
		paramType := attribDef.attrType()
		params = append(params, fmt.Sprintf("%s %s", paramName, paramType))
	}

	if hasOptionalAttributes(m) {
		optionTypeName := fmt.Sprintf("%sOption", m.Name)
		params = append(params, fmt.Sprintf("opts ...%s", optionTypeName))
	}

	return params
}

func buildAttributesCode(f *os.File, m *metric, methodCall string) {
	requiredAttrs := getRequiredAttributes(m)

	if len(requiredAttrs) > 0 || hasOptionalAttributes(m) {
		fmt.Fprintf(f, "\tattribs := InstAttribs{\n")
		for _, attr := range requiredAttrs {
			paramName := toLowerCamelCase(attr.Name)
			fmt.Fprintf(f, "\t\t{Name: Attrib%s, Value: %s},\n", attr.Name, paramName)
		}
		fmt.Fprintf(f, "\t}\n")

		if hasOptionalAttributes(m) {
			fmt.Fprintf(f, "\tfor _, opt := range opts {\n")
			fmt.Fprintf(f, "\t\topt(&attribs)\n")
			fmt.Fprintf(f, "\t}\n")
		}

		fmt.Fprintf(f, "\t%s, attribs)\n", methodCall)
	} else {
		fmt.Fprintf(f, "\t%s, InstAttribs{})\n", methodCall)
	}
}

func generateCounterHelper(f *os.File, m *metric, attributes *attributesList) {
	methodName := fmt.Sprintf("Add%s", m.Name)
	params := buildParameterList([]string{"ctx context.Context", "val int64"}, m, attributes)

	fmt.Fprintf(f, "// %s adds a value to the %s counter\n", methodName, m.displayName())
	fmt.Fprintf(f, "func (m *Metrics) %s(%s) {\n", methodName, strings.Join(params, ", "))
	buildAttributesCode(f, m, fmt.Sprintf("m.AddInt(ctx, Instrument%s.Name(), val", m.Name))
	fmt.Fprintf(f, "}\n\n")
}

func generateHistogramHelper(f *os.File, m *metric, attributes *attributesList) {
	methodName := fmt.Sprintf("Record%s", m.Name)
	params := buildParameterList([]string{"ctx context.Context", "val float64"}, m, attributes)

	fmt.Fprintf(f, "// %s records a value to the %s histogram\n", methodName, m.displayName())
	fmt.Fprintf(f, "func (m *Metrics) %s(%s) {\n", methodName, strings.Join(params, ", "))
	buildAttributesCode(f, m, fmt.Sprintf("m.Record(ctx, Instrument%s.Name(), val", m.Name))
	fmt.Fprintf(f, "}\n\n")
}

func generateObservableHelper(f *os.File, m *metric, attributes *attributesList) {
	methodName := fmt.Sprintf("Observe%s", m.Name)

	// Determine value type based on metric type
	valueType := "int64"
	observeMethod := "ObserveInt"
	if m.Type == "Float64ObservableGauge" {
		valueType = "float64"
		observeMethod = "ObserveFloat"
	}

	params := buildParameterList([]string{"ctx context.Context", "o metric.Observer", fmt.Sprintf("val %s", valueType)}, m, attributes)

	fmt.Fprintf(f, "// %s observes a value for the %s gauge\n", methodName, m.displayName())
	fmt.Fprintf(f, "// This is a helper method for use inside RegisterCallback functions\n")
	fmt.Fprintf(f, "func (m *Metrics) %s(%s) {\n", methodName, strings.Join(params, ", "))
	fmt.Fprintf(f, "\tinst := m.GetInstrument(Instrument%s.Name())\n", m.Name)
	fmt.Fprintf(f, "\tif inst == nil {\n")
	fmt.Fprintf(f, "\t\treturn\n")
	fmt.Fprintf(f, "\t}\n")
	buildAttributesCode(f, m, fmt.Sprintf("inst.%s(ctx, o, val", observeMethod))
	fmt.Fprintf(f, "}\n\n")
}

func getRequiredAttributes(m *metric) []allowedAttribute {
	var required []allowedAttribute
	for _, attr := range m.Attributes {
		if !attr.Optional {
			required = append(required, attr)
		}
	}
	return required
}

func toLowerCamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = rune(strings.ToLower(string(runes[0]))[0])
	return string(runes)
}
