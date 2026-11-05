package tracing

import (
	"context"
	"strings"

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

// TraceContextFromEnv reads the W3C trace context the controller gave a
// task, as TRACEPARENT and TRACESTATE in the env lookupEnv reads, or nil if
// it has no traceparent. Unlike InjectTraceContext it reads any env, such as
// a pooled task's, not only the process's own.
func TraceContextFromEnv(lookupEnv func(string) (string, bool)) map[string]string {
	carrier := propagation.MapCarrier{}
	for _, key := range (propagation.TraceContext{}).Fields() {
		if v, _ := lookupEnv(strings.ToUpper(key)); v != "" {
			carrier.Set(key, v)
		}
	}
	if carrier.Get("traceparent") == "" {
		return nil
	}
	return carrier
}

func InjectTraceContext(ctx context.Context) context.Context {
	// Extract-only carrier: no SetEnvFunc, so it reads TRACEPARENT from the
	// process environment the controller injected into the pod spec.
	carrier := &envcar.Carrier{}
	prop := propagation.TraceContext{}
	return prop.Extract(ctx, carrier)
}
