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

var metricHeaderTmpl = template.Must(template.New("header").Parse(`{{.Banner}}
    //
    //go:generate go run ./builder --metricsHelpersGo {{.Filename}}
    package telemetry

    import (
	    "context"

    	"go.opentelemetry.io/otel/metric"
    )

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

func createMetricsHelpersGo(filename string, metrics *metricsList, attributes *attributesList) error {
	err := writeMetricsHelpersGo(filename, metrics, attributes)
	if err != nil {
		return err
	}
	return goFmtFile(filename)
}

func writeMetricsHelpersGo(filename string, metrics *metricsList, attributes *attributesList) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write file header
	err = metricHeaderTmpl.Execute(f, map[string]string{"Banner": generatedBanner, "Filename": filename})
	if err != nil {
		return err
	}

	// Generate option types and functions for metrics with optional attributes
	for _, metric := range *metrics {
		if hasOptionalAttributes(&metric.Attributes) {
			optionTypeName := fmt.Sprintf("%s%sOption", metric.Name, metricType)
			err = generateMetricOption(f, &metric.common, optionTypeName, attributes)
			if err != nil {
				return err
			}
		}
	}

	// Generate helper methods for each metric
	for _, metric := range *metrics {
		switch metric.Type {
		case "Int64Counter", "Int64UpDownCounter":
			err = generateCounterHelper(f, &metric, attributes)
		case "Float64Histogram":
			err = generateHistogramHelper(f, &metric, attributes)
		case "Int64ObservableGauge", "Float64ObservableGauge":
			err = generateObservableHelper(f, &metric, attributes)
		}
		if err != nil {
			return err
		}

	}
	return nil
}

func generateCounterHelper(f io.Writer, m *metric, attributes *attributesList) error {
	methodName := fmt.Sprintf("Add%s", m.Name)
	params := buildParameterList([]string{"ctx context.Context", "val int64"}, &m.common, metricType, attributes)
	methodCall := fmt.Sprintf("m.AddInt(ctx, Instrument%s.Name(), val", m.Name)

	return counterHelperTmpl.Execute(f, map[string]string{
		"MethodName":     methodName,
		"DisplayName":    m.displayName(),
		"Params":         strings.Join(params, ", "),
		"AttributesCode": buildMetricsAttributesCode(&m.Attributes) + buildMetricsMethodCall(methodCall, &m.common),
	})
}

func generateHistogramHelper(f io.Writer, m *metric, attributes *attributesList) error {
	methodName := fmt.Sprintf("Record%s", m.Name)
	params := buildParameterList([]string{"ctx context.Context", "val float64"}, &m.common, metricType, attributes)
	methodCall := fmt.Sprintf("m.Record(ctx, Instrument%s.Name(), val", m.Name)

	return histogramHelperTmpl.Execute(f, map[string]string{
		"MethodName":     methodName,
		"DisplayName":    m.displayName(),
		"Params":         strings.Join(params, ", "),
		"AttributesCode": buildMetricsAttributesCode(&m.Attributes) + buildMetricsMethodCall(methodCall, &m.common),
	})
}

func generateObservableHelper(f io.Writer, m *metric, attributes *attributesList) error {
	methodName := fmt.Sprintf("Observe%s", m.Name)

	// Determine value type based on metric type
	valueType := "int64"
	observeMethod := "ObserveInt"
	if m.Type == "Float64ObservableGauge" {
		valueType = "float64"
		observeMethod = "ObserveFloat"
	}

	params := buildParameterList([]string{"ctx context.Context", "o metric.Observer", fmt.Sprintf("val %s", valueType)}, &m.common, metricType, attributes)
	methodCall := fmt.Sprintf("inst.%s(ctx, o, val", observeMethod)

	return observableHelperTmpl.Execute(f, map[string]string{
		"MethodName":     methodName,
		"DisplayName":    m.displayName(),
		"MetricName":     m.Name,
		"Params":         strings.Join(params, ", "),
		"AttributesCode": buildMetricsAttributesCode(&m.Attributes) + buildMetricsMethodCall(methodCall, &m.common),
	})
}

func buildMetricsAttributesCode(attributes *allowedAttributeList) string {
	requiredAttrs := getRequiredAttributes(attributes)
	var b strings.Builder

	if len(requiredAttrs) > 0 || hasOptionalAttributes(attributes) {
		b.WriteString("\tattribs := Attributes{\n")
		for _, attr := range requiredAttrs {
			paramName := toLowerCamelCase(attr.Name)
			fmt.Fprintf(&b, "\t\t{Name: %s, Value: %s},\n", attr.AttribName(), paramName)
		}
		b.WriteString("\t}\n")

		if hasOptionalAttributes(attributes) {
			b.WriteString("\tfor _, opt := range opts {\n")
			b.WriteString("\t\topt(&attribs)\n")
			b.WriteString("\t}\n")
		}
	}

	return b.String()
}

func buildMetricsMethodCall(methodCall string, c *common) string {
	requiredAttrs := getRequiredAttributes(&c.Attributes)
	if len(requiredAttrs) > 0 || hasOptionalAttributes(&c.Attributes) {
		return fmt.Sprintf("\t%s, attribs)", methodCall)
	}
	return fmt.Sprintf("\t%s, Attributes{})", methodCall)
}

var metricOptionTmpl = template.Must(template.New("optionType").Parse(`// {{.OptionTypeName}} is a functional option for configuring optional attributes on {{.MetricName}}
type {{.OptionTypeName}} func(*Attributes)

`))

var metricOptionFuncTmpl = template.Must(template.New("optionFunc").Parse(`// {{.FuncName}} sets the {{.DisplayName}} attribute
func {{.FuncName}}({{.ParamName}} {{.ParamType}}) {{.OptionTypeName}} {
	return func(a *Attributes) {
		*a = append(*a, Attribute{
			Name:  Attrib{{.AttrName}},
			Value: {{.ParamName}},
		})
	}
}

`))

func generateMetricOption(f io.Writer, c *common, optionTypeName string, attributes *attributesList) error {
	// Generate the option type
	err := metricOptionTmpl.Execute(f, map[string]string{
		"OptionTypeName": optionTypeName,
		"MetricName":     c.Name,
	})
	if err != nil {
		return err
	}

	// Generate option functions for each optional attribute
	for _, attr := range c.Attributes {
		if attr.Optional {
			attribDef := getAttribByName(attr.Name, attributes)
			paramName := toLowerCamelCase(attr.Name)
			paramType := attribDef.attrType()
			funcName := fmt.Sprintf("With%s", attr.Name)

			err = metricOptionFuncTmpl.Execute(f, map[string]string{
				"FuncName":       funcName,
				"DisplayName":    attribDef.displayName(),
				"ParamName":      paramName,
				"ParamType":      paramType,
				"OptionTypeName": optionTypeName,
				"AttrName":       attr.Name,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
