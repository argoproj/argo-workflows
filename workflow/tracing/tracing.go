package tracing

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/util/telemetry/ratelimiter"
)

type nodeSpans struct {
	node       *trace.Span
	phase      *trace.Span
	phasePhase wfv1.NodePhase
	phaseMsg   string
}

type workflowSpans struct {
	workflow *trace.Span
	phase    *trace.Span
	nodes    map[string]nodeSpans
	mutex    sync.RWMutex
}

type Tracing struct {
	*telemetry.Tracing
	workflows map[string]*workflowSpans
	mutex     sync.RWMutex
}

func New(ctx context.Context, serviceName string) (*Tracing, error) {
	tracing, err := telemetry.NewTracing(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	t := &Tracing{
		Tracing:   tracing,
		workflows: make(map[string]*workflowSpans),
	}

	// Register the tracing callback for rate limiter spans
	t.RegisterRateLimiterCallback()

	return t, nil
}

// RegisterRateLimiterCallback registers the tracing callback with the rate limiter instrumentation.
// This enables span creation for rate limiter waits when there's an active span context.
func (trc *Tracing) RegisterRateLimiterCallback() {
	ratelimiter.DefaultInstrumentation.SetTracingCallback(trc.StartWaitClientRateLimiterSafe)
}

// StartWaitClientRateLimiterSafe creates a span for rate limiter waits only when there's a valid
// span context. Unlike StartWaitClientRateLimiter, this method doesn't validate the parent span
// name, allowing spans to be created under any active context (nodePhase, reconcileWorkflow, etc.).
func (trc *Tracing) StartWaitClientRateLimiterSafe(ctx context.Context) (context.Context, trace.Span) {
	parent := trace.SpanFromContext(ctx)

	// Only create span if we have a valid tracing context
	if !parent.SpanContext().IsValid() {
		return ctx, noop.Span{} // No span context - skip
	}

	// Create span under whatever parent exists (provides more visibility)
	// Use the embedded telemetry.Tracing method
	return trc.StartWaitClientRateLimiter(ctx)
}
