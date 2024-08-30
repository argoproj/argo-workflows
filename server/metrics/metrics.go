package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics struct {
	telemetry.Metrics
}

func New(ctx context.Context, serviceName, prometheusName string, config *telemetry.Config, extraOpts ...metricsdk.Option) (*Metrics, error) {
	m, err := telemetry.NewMetrics(ctx, serviceName, prometheusName, config, extraOpts...)
	if err != nil {
		return nil, err
	}

	err = m.Populate(ctx,
		telemetry.AddVersion,
	)
	if err != nil {
		return nil, err
	}

	metrics := &Metrics{
		Metrics: *m,
	}

	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func GetServerConfig(config *config.Config) *telemetry.Config {
	// Metrics config
	modifiers := make(map[string]telemetry.Modifier)
	for name, modifier := range config.ServerMetricsConfig.Modifiers {
		modifiers[name] = telemetry.Modifier{
			Disabled:           modifier.Disabled,
			DisabledAttributes: modifier.DisabledAttributes,
			HistogramBuckets:   modifier.HistogramBuckets,
		}
	}

	metricsConfig := telemetry.Config{
		Enabled:      config.ServerMetricsConfig.Enabled == nil || *config.ServerMetricsConfig.Enabled,
		Path:         config.ServerMetricsConfig.Path,
		Port:         config.ServerMetricsConfig.Port,
		IgnoreErrors: config.ServerMetricsConfig.IgnoreErrors,
		Secure:       config.ServerMetricsConfig.GetSecure(true),
		Modifiers:    modifiers,
		Temporality:  config.ServerMetricsConfig.Temporality,
	}
	return &metricsConfig
}
