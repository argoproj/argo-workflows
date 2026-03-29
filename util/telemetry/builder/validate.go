package main

import (
	"fmt"
	"slices"
	"strings"
)

type validator struct {
	errors []error
}

func (v *validator) recordError(format string, args ...any) {
	v.errors = append(v.errors, fmt.Errorf(format, args...))
}

func (v *validator) valid() bool {
	return len(v.errors) == 0
}

func (v *validator) printErrors() {
	for _, err := range v.errors {
		fmt.Println(err)
	}
	fmt.Printf("%d validation errors\n", len(v.errors))
}

func (v *validator) telemetryAttributes(items []hasCommon, attributes *attributesList, context string) {
	for _, item := range items {
		c := item.Common()
		for _, attribute := range c.Attributes {
			if getAttribByName(attribute.Name, attributes) == nil {
				v.recordError("%s: %s: attribute %s not defined", context, c.Name, attribute.Name)
			}
		}
	}
}

func (v *validator) attributes(attributes *attributesList) {
	if !slices.IsSortedFunc(*attributes, func(a, b attribute) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		v.recordError("attributes: Attributes must be alphabetically sorted by Name")
	}
	for _, attribute := range *attributes {
		if strings.Contains(attribute.Description, "\n") {
			v.recordError("attributes: %s: Description must be a single line", attribute.Name)
		}
	}
}

func (v *validator) buckets(m *metric) {
	if len(m.DefaultBuckets) > 0 && m.Type != "Float64Histogram" {
		v.recordError("%s: defaultBuckets can only be used with Float64Histogram ms", m.Name)
	}
}

func (v *validator) description(telemetry common) {
	if strings.Contains(telemetry.Description, "\n") {
		v.recordError("description: %s: Description must be a single line", telemetry.Name)
	}
	if strings.HasSuffix(telemetry.Description, ".") {
		v.recordError("description: %s: Description must not have a trailing period", telemetry.Name)
	}
}

func (v *validator) metrics(metrics *metricsList) {
	if !slices.IsSortedFunc(*metrics, func(a, b metric) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		v.recordError("metrics: Metrics must be alphabetically sorted by Name")
	}
	for _, metric := range *metrics {
		// This is easier than enum+custom JSON unmarshall as this is not critical code
		switch metric.Type {
		case "Float64Histogram",
			"Float64ObservableGauge",
			"Int64Counter",
			"Int64UpDownCounter",
			"Int64ObservableGauge":
			break
		default:
			v.recordError("metrics: %s: Invalid metric type %s", metric.Name, metric.Type)
		}
		v.description(metric.common)
		v.buckets(&metric)
	}
}

// We're wanting a tree without a repeat
func (v *validator) validateSpanParentage(spanToCheck span, history []string, spans *spansList) {
	if slices.Contains(history, spanToCheck.Name) {
		v.recordError("span %s appears more than once in parentage history %v", spanToCheck.Name, history)
		return
	}
	history = append(history, spanToCheck.Name)
	if spanToCheck.Root || spanToCheck.AnyParent {
		return
	}
	if len(spanToCheck.Parents) == 0 {
		v.recordError("span %s has no parents declared and is not a root", spanToCheck.Name)
		return
	}
	for _, parent := range spanToCheck.Parents {
		parentIndex := slices.IndexFunc(*spans, func(s span) bool {
			return parent == s.Name
		})
		if parentIndex != -1 {
			parentSpan := (*spans)[parentIndex]
			v.validateSpanParentage(parentSpan, history, spans)
		} else {
			v.recordError("span %s has no valid parent (%s) and is not a trace", spanToCheck.Name, parent)
		}
	}
}

func (v *validator) spans(spans *spansList) {
	if !slices.IsSortedFunc(*spans, func(a, b span) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		v.recordError("spans: Spans must be alphabetically sorted by Name")
	}
	for _, span := range *spans {
		switch span.Kind {
		case "Internal",
			"Server",
			"Client",
			"Consumer",
			"Producer",
			"": // "" is mapped to Internal
			break
		default:
			v.recordError("spans: %s: Invalid span type %s", span.Name, span.Kind)
		}
		v.description(span.common)
		v.validateSpanParentage(span, []string{}, spans)
	}
}

func validate(vals *values) validator {
	var v validator
	v.attributes(&vals.Attributes)
	v.metrics(&vals.Metrics)
	metrics := make([]hasCommon, len(vals.Metrics))
	for i, m := range vals.Metrics {
		metrics[i] = m
	}
	v.telemetryAttributes(metrics, &vals.Attributes, "metrics")
	v.spans(&vals.Spans)
	spans := make([]hasCommon, len(vals.Spans))
	for i, s := range vals.Spans {
		spans[i] = s
	}
	v.telemetryAttributes(spans, &vals.Attributes, "spans")
	return v
}
