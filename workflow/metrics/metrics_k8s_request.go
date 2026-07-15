package metrics

import (
	"context"
	"net/http"
	"time"

	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/util/k8s"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/util/telemetry/ratelimiter"
)

func addK8sRequests(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentK8sRequestTotal)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentK8sRequestDuration)
	// Register these helper methods with the global
	k8sMetrics.addK8sRequestTotal = m.AddK8sRequestTotal
	k8sMetrics.recordK8sRequestDuration = m.RecordK8sRequestDuration
	return err
}

type metricsRoundTripperContext struct {
	//nolint: containedctx
	ctx                      context.Context
	addK8sRequestTotal       func(ctx context.Context, val int64, requestKind string, requestVerb string, requestCode int)
	recordK8sRequestDuration func(ctx context.Context, durationSeconds float64, requestKind string, requestVerb string, requestCode int)
}

type metricsRoundTripper struct {
	*metricsRoundTripperContext
	roundTripper http.RoundTripper
}

// This is a messy global as we need to register as a roundtripper before
// we can instantiate metrics
var k8sMetrics metricsRoundTripperContext

func (m metricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	startTime := time.Now()
	x, err := m.roundTripper.RoundTrip(r)
	duration := time.Since(startTime)
	if x != nil && m.addK8sRequestTotal != nil {
		verb, kind := k8s.ParseRequest(r)
		m.addK8sRequestTotal(m.ctx, 1, kind, verb, x.StatusCode)
		m.recordK8sRequestDuration(m.ctx, duration.Seconds(), kind, verb, x.StatusCode)
	}
	return x, err
}

func AddMetricsTransportWrapper(ctx context.Context, config *rest.Config) *rest.Config {
	k8sMetrics.ctx = ctx
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		return &metricsRoundTripper{roundTripper: rt, metricsRoundTripperContext: &k8sMetrics}
	}
	return config
}

func addClientRateLimiterLatency(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentClientRateLimiterLatency)
	if err != nil {
		return err
	}
	// Register the metrics callback with the rate limiter instrumentation
	ratelimiter.DefaultInstrumentation.SetMetricsCallback(m.RecordClientRateLimiterLatency)
	return nil
}
