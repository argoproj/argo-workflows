package metrics

import (
	"go.opentelemetry.io/otel/attribute"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

// Modifier holds options to change the behaviour for a single metric
type Modifier struct {
	Disabled           bool
	DisabledAttributes []string
	HistogramBuckets   []float64
}

// Create an opentelemetry 'view' which disables whole metrics or aggregates across labels
func view(config *Config) metricsdk.Option {
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
