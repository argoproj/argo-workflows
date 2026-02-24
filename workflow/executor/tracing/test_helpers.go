package tracing

import (
	"context"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

// CreateDefaultTestTracing creates a Tracing instance with a TestTracingExporter for testing.
// It returns the Tracing instance and the exporter so tests can query collected spans.
func CreateDefaultTestTracing(ctx context.Context) (*Tracing, *telemetry.TestTracingExporter, error) {
	te := telemetry.NewTestTracingExporter()
	baseTracing, err := telemetry.NewTracing(ctx, telemetry.TestTracingScopeName, tracesdk.WithSyncer(te))
	if err != nil {
		return nil, nil, err
	}
	return &Tracing{Tracing: baseTracing}, te, nil
}
