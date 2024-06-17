package metrics

import (
	"go.opentelemetry.io/otel/attribute"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

type MetricOption struct {
	Disable            bool
	DisabledAttributes []string
	HistogramBuckets   []float64
}

// Create an opentelemetry 'view' which disables whole metrics or aggregates across labels
func view(config *Config) metricsdk.Option {
	views := make([]metricsdk.View, 0)
	for metric, opt := range config.Options {
		if opt.Disable {
			views = append(views, metricsdk.NewView(metricsdk.Instrument{Name: metric},
				metricsdk.Stream{Aggregation: metricsdk.AggregationDrop{}}))
		} else if len(opt.DisabledAttributes) > 0 {
			keys := make([]attribute.Key, len(opt.DisabledAttributes))
			for i, key := range opt.DisabledAttributes {
				keys[i] = attribute.Key(key)
			}
			views = append(views, metricsdk.NewView(metricsdk.Instrument{Name: metric},
				metricsdk.Stream{AttributeFilter: attribute.NewDenyKeysFilter(keys...)}))
		}
	}
	return metricsdk.WithView(views...)
}
