package ratelimiter

import (
	"context"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// Instrumentation abstracts metrics and tracing for rate limiter
type Instrumentation interface {
	// RecordLatency records the time spent waiting for rate limiter
	RecordLatency(ctx context.Context, duration time.Duration)
	// StartWaitSpan optionally starts a span (returns no-op if inappropriate context)
	StartWaitSpan(ctx context.Context) (context.Context, func())
}

// AddRateLimiterWrapper wraps the config's rate limiter with an instrumented version
// that records the wait time metric and creates spans
func AddRateLimiterWrapper(_ context.Context, config *rest.Config) *rest.Config {
	if config.RateLimiter == nil {
		// If no rate limiter is set, create one using the QPS and Burst settings
		if config.QPS <= 0 || config.Burst <= 0 {
			// No rate limiting configured
			return config
		}
		config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(config.QPS, config.Burst)
	}
	config.RateLimiter = &instrumentedFlowControl{
		instrumentation: DefaultInstrumentation,
		inner:           config.RateLimiter,
	}
	return config
}

// instrumentedFlowControl wraps a flowcontrol.RateLimiter and records the wait time
type instrumentedFlowControl struct {
	instrumentation Instrumentation
	inner           flowcontrol.RateLimiter
}

func (r *instrumentedFlowControl) Accept() {
	// Accept() has no context - use background context for metrics only
	startTime := time.Now()
	r.inner.Accept()
	duration := time.Since(startTime)
	r.instrumentation.RecordLatency(context.Background(), duration)
}

func (r *instrumentedFlowControl) TryAccept() bool {
	return r.inner.TryAccept()
}

func (r *instrumentedFlowControl) Stop() {
	r.inner.Stop()
}

func (r *instrumentedFlowControl) QPS() float32 {
	return r.inner.QPS()
}

func (r *instrumentedFlowControl) Wait(ctx context.Context) error {
	// Start span (if appropriate parent exists)
	ctx, endSpan := r.instrumentation.StartWaitSpan(ctx)
	defer endSpan()

	startTime := time.Now()
	err := r.inner.Wait(ctx)
	duration := time.Since(startTime)

	r.instrumentation.RecordLatency(ctx, duration)
	return err
}

var _ flowcontrol.RateLimiter = &instrumentedFlowControl{}
