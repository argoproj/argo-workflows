package telemetry

import (
	"os"
	"strings"
)

const (
	otlpEndpointEnv        = "OTEL_EXPORTER_OTLP_ENDPOINT"
	otlpMetricsEndpointEnv = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"
	otlpTracesEndpointEnv  = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
)

// resolveOTLPEndpoint returns the signal-specific endpoint, falling back to the common OTLP endpoint.
func resolveOTLPEndpoint(signalEndpointEnv string) string {
	if endpoint := strings.TrimSpace(os.Getenv(signalEndpointEnv)); endpoint != "" {
		return endpoint
	}
	return strings.TrimSpace(os.Getenv(otlpEndpointEnv))
}
