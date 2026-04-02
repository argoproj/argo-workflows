package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

// MetricsModifier holds options to change the behaviour for a single metric
type MetricsModifier struct {
	Disabled           bool
	DisabledAttributes []string
	HistogramBuckets   []float64
}

// Create an opentelemetry 'view' which disables whole metrics or aggregates across attributes
func view(config *MetricsConfig) metricsdk.Option {
	views := make([]metricsdk.View, 0)
	for metric, modifier := range config.Modifiers {
		if modifier.Disabled {
			views = append(views, metricsdk.NewView(metricsdk.Instrument{Name: metric},
				metricsdk.Stream{Aggregation: metricsdk.AggregationDrop{}}))
		} else if len(modifier.DisabledAttributes) > 0 {
			keys := make([]attribute.Key, len(modifier.DisabledAttributes))
			for i, key := range modifier.DisabledAttributes {
				keys[i] = attribute.Key(key)
			}
			views = append(views, metricsdk.NewView(metricsdk.Instrument{Name: metric},
				metricsdk.Stream{AttributeFilter: attribute.NewDenyKeysFilter(keys...)}))
		}
	}
	return metricsdk.WithView(views...)
}

// TracingModifier holds options to change the behaviour for a trace
type TracingModifier struct {
	Disabled           bool
	DisableChildren    bool
	DisabledAttributes []string
}
