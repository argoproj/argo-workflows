package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"toLowerCamelCase": toLowerCamelCase,
	"join":             strings.Join,
}

var fileHeaderTmpl = template.Must(template.New("header").Parse(`{{.Banner}}
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

`))

var optionTypeTmpl = template.Must(template.New("optionType").Parse(`// {{.OptionTypeName}} is a functional option for configuring optional attributes on {{.MetricName}}
type {{.OptionTypeName}} func(*InstAttribs)

`))

var optionFuncTmpl = template.Must(template.New("optionFunc").Parse(`// {{.FuncName}} sets the {{.DisplayName}} attribute
func {{.FuncName}}({{.ParamName}} {{.ParamType}}) {{.OptionTypeName}} {
	return func(a *InstAttribs) {
		*a = append(*a, InstAttrib{
			Name:  Attrib{{.AttrName}},
			Value: {{.ParamName}},
		})
	}
}

`))

var counterHelperTmpl = template.Must(template.New("counter").Funcs(funcMap).Parse(`// {{.MethodName}} adds a value to the {{.DisplayName}} counter
func (m *Metrics) {{.MethodName}}({{.Params}}) {
{{.AttributesCode}}}

`))

var histogramHelperTmpl = template.Must(template.New("histogram").Funcs(funcMap).Parse(`// {{.MethodName}} records a value to the {{.DisplayName}} histogram
func (m *Metrics) {{.MethodName}}({{.Params}}) {
{{.AttributesCode}}}

`))

var observableHelperTmpl = template.Must(template.New("observable").Funcs(funcMap).Parse(`// {{.MethodName}} observes a value for the {{.DisplayName}} gauge
// This is a helper method for use inside RegisterCallback functions
func (m *Metrics) {{.MethodName}}({{.Params}}) {
	inst := m.GetInstrument(Instrument{{.MetricName}}.Name())
	if inst == nil {
		return
	}
{{.AttributesCode}}}

`))

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

	// Write file header
	fileHeaderTmpl.Execute(f, map[string]string{"Banner": generatedBanner})

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

func generateOptionType(f io.Writer, m *metric, optionTypeName string, attributes *attributesList) {
	// Generate the option type
	optionTypeTmpl.Execute(f, map[string]string{
		"OptionTypeName": optionTypeName,
		"MetricName":     m.Name,
	})

	// Generate option functions for each optional attribute
	for _, attr := range m.Attributes {
		if attr.Optional {
			attribDef := getAttribByName(attr.Name, attributes)
			paramName := toLowerCamelCase(attr.Name)
			paramType := attribDef.attrType()
			funcName := fmt.Sprintf("With%s", attr.Name)

			optionFuncTmpl.Execute(f, map[string]string{
				"FuncName":       funcName,
				"DisplayName":    attribDef.displayName(),
				"ParamName":      paramName,
				"ParamType":      paramType,
				"OptionTypeName": optionTypeName,
				"AttrName":       attr.Name,
			})
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

func buildAttributesCode(m *metric) string {
	requiredAttrs := getRequiredAttributes(m)
	var b strings.Builder

	if len(requiredAttrs) > 0 || hasOptionalAttributes(m) {
		b.WriteString("\tattribs := InstAttribs{\n")
		for _, attr := range requiredAttrs {
			paramName := toLowerCamelCase(attr.Name)
			fmt.Fprintf(&b, "\t\t{Name: Attrib%s, Value: %s},\n", attr.Name, paramName)
		}
		b.WriteString("\t}\n")

		if hasOptionalAttributes(m) {
			b.WriteString("\tfor _, opt := range opts {\n")
			b.WriteString("\t\topt(&attribs)\n")
			b.WriteString("\t}\n")
		}
	}

	return b.String()
}

func buildMethodCall(methodCall string, m *metric) string {
	requiredAttrs := getRequiredAttributes(m)
	if len(requiredAttrs) > 0 || hasOptionalAttributes(m) {
		return fmt.Sprintf("\t%s, attribs)", methodCall)
	}
	return fmt.Sprintf("\t%s, InstAttribs{})", methodCall)
}

func generateCounterHelper(f io.Writer, m *metric, attributes *attributesList) {
	methodName := fmt.Sprintf("Add%s", m.Name)
	params := buildParameterList([]string{"ctx context.Context", "val int64"}, m, attributes)
	methodCall := fmt.Sprintf("m.AddInt(ctx, Instrument%s.Name(), val", m.Name)

	counterHelperTmpl.Execute(f, map[string]string{
		"MethodName":     methodName,
		"DisplayName":    m.displayName(),
		"Params":         strings.Join(params, ", "),
		"AttributesCode": buildAttributesCode(m) + buildMethodCall(methodCall, m),
	})
}

func generateHistogramHelper(f io.Writer, m *metric, attributes *attributesList) {
	methodName := fmt.Sprintf("Record%s", m.Name)
	params := buildParameterList([]string{"ctx context.Context", "val float64"}, m, attributes)
	methodCall := fmt.Sprintf("m.Record(ctx, Instrument%s.Name(), val", m.Name)

	histogramHelperTmpl.Execute(f, map[string]string{
		"MethodName":     methodName,
		"DisplayName":    m.displayName(),
		"Params":         strings.Join(params, ", "),
		"AttributesCode": buildAttributesCode(m) + buildMethodCall(methodCall, m),
	})
}

func generateObservableHelper(f io.Writer, m *metric, attributes *attributesList) {
	methodName := fmt.Sprintf("Observe%s", m.Name)

	// Determine value type based on metric type
	valueType := "int64"
	observeMethod := "ObserveInt"
	if m.Type == "Float64ObservableGauge" {
		valueType = "float64"
		observeMethod = "ObserveFloat"
	}

	params := buildParameterList([]string{"ctx context.Context", "o metric.Observer", fmt.Sprintf("val %s", valueType)}, m, attributes)
	methodCall := fmt.Sprintf("inst.%s(ctx, o, val", observeMethod)

	observableHelperTmpl.Execute(f, map[string]string{
		"MethodName":     methodName,
		"DisplayName":    m.displayName(),
		"MetricName":     m.Name,
		"Params":         strings.Join(params, ", "),
		"AttributesCode": buildAttributesCode(m) + buildMethodCall(methodCall, m),
	})
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
