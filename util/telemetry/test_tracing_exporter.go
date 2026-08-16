package telemetry

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TestTracingScopeName is the name that the tracing running under test will have
const TestTracingScopeName string = "argo-workflows-test"

// TestTracingExporter is an opentelemetry tracing exporter, purely for use within
// tests. It collects spans in-memory and provides methods to query them by name
// and attributes for the purposes of testing only.
// This is a public structure as it is used outside of this module also.
type TestTracingExporter struct {
	mu    sync.RWMutex
	spans []sdktrace.ReadOnlySpan
}

var _ sdktrace.SpanExporter = &TestTracingExporter{}

// NewTestTracingExporter creates a new test tracing exporter
func NewTestTracingExporter() *TestTracingExporter {
	return &TestTracingExporter{
		spans: make([]sdktrace.ReadOnlySpan, 0),
	}
}

// ExportSpans implements the SpanExporter interface, collecting spans in-memory
func (t *TestTracingExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = append(t.spans, spans...)
	return nil
}

// Shutdown implements the SpanExporter interface, no-op for tests
func (t *TestTracingExporter) Shutdown(ctx context.Context) error {
	return nil
}

// Reset clears all collected spans, useful between test cases
func (t *TestTracingExporter) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = make([]sdktrace.ReadOnlySpan, 0)
}

// GetSpans returns all collected spans
func (t *TestTracingExporter) GetSpans() []sdktrace.ReadOnlySpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]sdktrace.ReadOnlySpan, len(t.spans))
	copy(result, t.spans)
	return result
}

// GetSpansByName returns all spans matching the given name
func (t *TestTracingExporter) GetSpansByName(name string) []sdktrace.ReadOnlySpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]sdktrace.ReadOnlySpan, 0)
	for _, span := range t.spans {
		if span.Name() == name {
			result = append(result, span)
		}
	}
	return result
}

// GetSpanByName returns the first span matching the given name
func (t *TestTracingExporter) GetSpanByName(name string) (sdktrace.ReadOnlySpan, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, span := range t.spans {
		if span.Name() == name {
			return span, nil
		}
	}
	return nil, fmt.Errorf("span with name %q not found", name)
}

// GetSpanByNameAndAttributes returns the first span matching the given name and attributes
func (t *TestTracingExporter) GetSpanByNameAndAttributes(name string, attribs *attribute.Set) (sdktrace.ReadOnlySpan, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, span := range t.spans {
		if span.Name() != name {
			continue
		}
		spanAttribs := attribute.NewSet(span.Attributes()...)
		if spanAttribs.Equals(attribs) {
			return span, nil
		}
	}
	return nil, fmt.Errorf("span with name %q and attribs %v not found", name, attribs)
}

// GetChildSpans returns all spans that have the given parent span ID
func (t *TestTracingExporter) GetChildSpans(parentSpanID trace.SpanID) []sdktrace.ReadOnlySpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]sdktrace.ReadOnlySpan, 0)
	for _, span := range t.spans {
		if span.Parent().SpanID() == parentSpanID {
			result = append(result, span)
		}
	}
	return result
}

// GetRootSpans returns all spans that have no parent (root spans)
func (t *TestTracingExporter) GetRootSpans() []sdktrace.ReadOnlySpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]sdktrace.ReadOnlySpan, 0)
	for _, span := range t.spans {
		if !span.Parent().IsValid() {
			result = append(result, span)
		}
	}
	return result
}

// SpanCount returns the number of collected spans
func (t *TestTracingExporter) SpanCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.spans)
}
