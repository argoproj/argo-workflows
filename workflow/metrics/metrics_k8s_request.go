package metrics

import (
	"context"
	"net/http"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
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
	ctx                            context.Context
	addK8sRequestTotal             func(ctx context.Context, val int64, requestKind string, requestVerb string, requestCode int)
	recordK8sRequestDuration       func(ctx context.Context, durationSeconds float64, requestKind string, requestVerb string, requestCode int)
	recordClientRateLimiterLatency func(ctx context.Context, val float64)
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
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		return &metricsRoundTripper{roundTripper: rt, metricsRoundTripperContext: &k8sMetrics}
	}
	return config
}

// instrumentedFlowControlRateLimiter wraps a flowcontrol.RateLimiter and records the wait time
type instrumentedFlowControlRateLimiter struct {
	rateLimiter *metricsRoundTripperContext
	inner       flowcontrol.RateLimiter
}

func (r *instrumentedFlowControlRateLimiter) Accept() {
	startTime := time.Now()
	r.inner.Accept()
	waitTime := time.Since(startTime)
	if r.rateLimiter.recordClientRateLimiterLatency != nil {
		r.rateLimiter.recordClientRateLimiterLatency(r.rateLimiter.ctx, waitTime.Seconds())
	}
}

func (r *instrumentedFlowControlRateLimiter) TryAccept() bool {
	return r.inner.TryAccept()
}

func (r *instrumentedFlowControlRateLimiter) Stop() {
	r.inner.Stop()
}

func (r *instrumentedFlowControlRateLimiter) QPS() float32 {
	return r.inner.QPS()
}

func (r *instrumentedFlowControlRateLimiter) Wait(ctx context.Context) error {
	startTime := time.Now()
	err := r.inner.Wait(ctx)
	waitTime := time.Since(startTime)
	if r.rateLimiter.recordClientRateLimiterLatency != nil {
		r.rateLimiter.recordClientRateLimiterLatency(ctx, waitTime.Seconds())
	}
	return err
}

var _ flowcontrol.RateLimiter = &instrumentedFlowControlRateLimiter{}

func addClientRateLimiterLatency(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentClientRateLimiterLatency)
	if err != nil {
		return err
	}
	// Register the helper method with the global
	k8sMetrics.recordClientRateLimiterLatency = m.RecordClientRateLimiterLatency
	return nil
}

// AddRateLimiterWrapper wraps the config's rate limiter with an instrumented version
// that records the wait time metric
func AddRateLimiterWrapper(ctx context.Context, config *rest.Config) *rest.Config {
	if config.RateLimiter == nil {
		// If no rate limiter is set, create one using the QPS and Burst settings
		if config.QPS <= 0 || config.Burst <= 0 {
			// No rate limiting configured
			return config
		}
		config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(config.QPS, config.Burst)
	}
	// Store the context for later use when metrics are initialized
	k8sMetrics.ctx = ctx
	config.RateLimiter = &instrumentedFlowControlRateLimiter{
		rateLimiter: &k8sMetrics,
		inner:       config.RateLimiter,
	}
	return config
}
