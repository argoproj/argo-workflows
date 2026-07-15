package telemetry

import (
	"time"

	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

type MetricsConfig struct {
	Enabled      bool
	Path         string
	Port         int
	TTL          time.Duration
	IgnoreErrors bool
	Secure       bool
	Modifiers    map[string]MetricsModifier
	Temporality  metricsdk.TemporalitySelector
}

type TracingConfig struct {
	Modifiers map[string]TracingModifier
}
