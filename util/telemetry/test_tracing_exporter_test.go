package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTestTracingExporter_BasicSpanCollection(t *testing.T) {
	ctx := context.Background()
	te := NewTestTracingExporter()

	tracing, err := NewTracing(ctx, TestTracingScopeName, tracesdk.WithSyncer(te))
	require.NoError(t, err)

	// Start and end a span
	_, span := tracing.tracer.Start(ctx, "test-span")
	span.End()

	// Verify span was collected
	assert.Equal(t, 1, te.SpanCount())

	// Verify we can query by name
	collected, err := te.GetSpanByName("test-span")
	require.NoError(t, err)
	assert.Equal(t, "test-span", collected.Name())
}

func TestTestTracingExporter_GetSpansByName(t *testing.T) {
	ctx := context.Background()
	te := NewTestTracingExporter()

	tracing, err := NewTracing(ctx, TestTracingScopeName, tracesdk.WithSyncer(te))
	require.NoError(t, err)

	// Create multiple spans with same name
	for range 3 {
		_, span := tracing.tracer.Start(ctx, "repeated-span")
		span.End()
	}

	// Create a span with different name
	_, span := tracing.tracer.Start(ctx, "different-span")
	span.End()

	// Verify we get all spans with matching name
	spans := te.GetSpansByName("repeated-span")
	assert.Len(t, spans, 3)

	// Verify we get the different span
	spans = te.GetSpansByName("different-span")
	assert.Len(t, spans, 1)
}

func TestTestTracingExporter_GetSpanByNameAndAttributes(t *testing.T) {
	ctx := context.Background()
	te := NewTestTracingExporter()

	tracing, err := NewTracing(ctx, TestTracingScopeName, tracesdk.WithSyncer(te))
	require.NoError(t, err)

	// Create spans with different attributes
	_, span1 := tracing.tracer.Start(ctx, "attr-span",
		trace.WithAttributes(attribute.String("key", "value1")))
	span1.End()

	_, span2 := tracing.tracer.Start(ctx, "attr-span",
		trace.WithAttributes(attribute.String("key", "value2")))
	span2.End()

	// Query by name and attributes
	attribs := attribute.NewSet(attribute.String("key", "value1"))
	collected, err := te.GetSpanByNameAndAttributes("attr-span", &attribs)
	require.NoError(t, err)

	spanAttribs := attribute.NewSet(collected.Attributes()...)
	assert.True(t, spanAttribs.Equals(&attribs))
}

func TestTestTracingExporter_ParentChildRelationships(t *testing.T) {
	ctx := context.Background()
	te := NewTestTracingExporter()

	tracing, err := NewTracing(ctx, TestTracingScopeName, tracesdk.WithSyncer(te))
	require.NoError(t, err)

	// Create parent span
	ctx, parentSpan := tracing.tracer.Start(ctx, "parent-span")
	parentSpanID := parentSpan.SpanContext().SpanID()

	// Create child spans
	_, child1 := tracing.tracer.Start(ctx, "child-span-1")
	child1.End()

	_, child2 := tracing.tracer.Start(ctx, "child-span-2")
	child2.End()

	parentSpan.End()

	// Verify child spans
	children := te.GetChildSpans(parentSpanID)
	assert.Len(t, children, 2)

	// Verify root spans (should include the parent)
	roots := te.GetRootSpans()
	assert.Len(t, roots, 1)
	assert.Equal(t, "parent-span", roots[0].Name())
}

func TestTestTracingExporter_Reset(t *testing.T) {
	ctx := context.Background()
	te := NewTestTracingExporter()

	tracing, err := NewTracing(ctx, TestTracingScopeName, tracesdk.WithSyncer(te))
	require.NoError(t, err)

	// Create some spans
	_, span := tracing.tracer.Start(ctx, "span-1")
	span.End()

	_, span = tracing.tracer.Start(ctx, "span-2")
	span.End()

	assert.Equal(t, 2, te.SpanCount())

	// Reset
	te.Reset()

	// Verify spans are cleared
	assert.Equal(t, 0, te.SpanCount())
	assert.Empty(t, te.GetSpans())
}

func TestTestTracingExporter_GetSpanByName_NotFound(t *testing.T) {
	te := NewTestTracingExporter()

	_, err := te.GetSpanByName("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTestTracingExporter_GetSpanByNameAndAttributes_NotFound(t *testing.T) {
	ctx := context.Background()
	te := NewTestTracingExporter()

	tracing, err := NewTracing(ctx, TestTracingScopeName, tracesdk.WithSyncer(te))
	require.NoError(t, err)

	// Create a span with attributes
	_, span := tracing.tracer.Start(ctx, "attr-span",
		trace.WithAttributes(attribute.String("key", "value1")))
	span.End()

	// Query with different attributes
	attribs := attribute.NewSet(attribute.String("key", "different"))
	_, err = te.GetSpanByNameAndAttributes("attr-span", &attribs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
