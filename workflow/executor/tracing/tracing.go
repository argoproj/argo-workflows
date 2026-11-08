package tracing

import (
	"context"

	"go.opentelemetry.io/contrib/propagators/envcar"
	"go.opentelemetry.io/otel/propagation"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

type Tracing struct {
	*telemetry.Tracing
}

func New(ctx context.Context, serviceName string) (*Tracing, error) {
	tracing, err := telemetry.NewTracing(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	return &Tracing{
		Tracing: tracing,
	}, nil
}

func InjectTraceContext(ctx context.Context) context.Context {
	// Extract-only carrier: no SetEnvFunc, so it reads TRACEPARENT from the
	// process environment the controller injected into the pod spec.
	carrier := &envcar.Carrier{}
	prop := propagation.TraceContext{}
	return prop.Extract(ctx, carrier)
}
