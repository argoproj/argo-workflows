package ratelimiter

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// DefaultInstrumentation is the global instance - populated lazily after metrics/tracing init
var DefaultInstrumentation = &LazyInstrumentation{}

// LazyInstrumentation provides late-binding for metrics and tracing callbacks.
// It allows the rate limiter wrapper to be set up before metrics/tracing are initialized.
type LazyInstrumentation struct {
	mu            sync.RWMutex
	recordLatency func(ctx context.Context, seconds float64)
	startWaitSpan func(ctx context.Context) (context.Context, trace.Span)
}

// SetMetricsCallback registers the callback for recording rate limiter latency metrics
func (l *LazyInstrumentation) SetMetricsCallback(fn func(ctx context.Context, seconds float64)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.recordLatency = fn
}

// SetTracingCallback registers the callback for starting rate limiter wait spans
func (l *LazyInstrumentation) SetTracingCallback(fn func(ctx context.Context) (context.Context, trace.Span)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.startWaitSpan = fn
}

// RecordLatency records the time spent waiting for rate limiter
func (l *LazyInstrumentation) RecordLatency(ctx context.Context, duration time.Duration) {
	l.mu.RLock()
	fn := l.recordLatency
	l.mu.RUnlock()
	if fn != nil {
		fn(ctx, duration.Seconds())
	}
}

// StartWaitSpan optionally starts a span (returns no-op if inappropriate context or no callback)
func (l *LazyInstrumentation) StartWaitSpan(ctx context.Context) (context.Context, func()) {
	l.mu.RLock()
	fn := l.startWaitSpan
	l.mu.RUnlock()
	if fn != nil {
		newCtx, span := fn(ctx)
		return newCtx, func() { span.End() }
	}
	return ctx, func() {} // no-op
}

// Ensure LazyInstrumentation implements Instrumentation
var _ Instrumentation = &LazyInstrumentation{}
