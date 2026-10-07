package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestDeterministicIDGenerator_NewIDs(t *testing.T) {
	gen := &DeterministicIDGenerator{}
	want := trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	ctx := WithSpanID(context.Background(), want)

	_, gotSpanID := gen.NewIDs(ctx)
	assert.Equal(t, want, gotSpanID)
}

func TestDeterministicIDGenerator_NewSpanID(t *testing.T) {
	gen := &DeterministicIDGenerator{}
	want := trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	ctx := WithSpanID(context.Background(), want)

	gotSpanID := gen.NewSpanID(ctx, trace.TraceID{})
	assert.Equal(t, want, gotSpanID)
}

func TestDeterministicIDGenerator_ConsumedOnce(t *testing.T) {
	gen := &DeterministicIDGenerator{}
	want := trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	ctx := WithSpanID(context.Background(), want)

	// First call consumes the override
	gotSpanID := gen.NewSpanID(ctx, trace.TraceID{})
	assert.Equal(t, want, gotSpanID)

	// Second call from the same context gets a random ID
	gotSpanID = gen.NewSpanID(ctx, trace.TraceID{})
	assert.NotEqual(t, want, gotSpanID)
	assert.True(t, gotSpanID.IsValid())
}

func TestDeterministicIDGenerator_FallsBackToRandom(t *testing.T) {
	gen := &DeterministicIDGenerator{}

	tid, sid := gen.NewIDs(context.Background())
	assert.True(t, tid.IsValid())
	assert.True(t, sid.IsValid())
}

func TestWithSpanID_IgnoresInvalid(t *testing.T) {
	gen := &DeterministicIDGenerator{}

	ctx := WithSpanID(context.Background(), trace.SpanID{})
	_, sid := gen.NewIDs(ctx)
	assert.True(t, sid.IsValid()) // got a random one, not zero
}
