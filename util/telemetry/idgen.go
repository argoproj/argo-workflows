package telemetry

import (
	"context"
	"encoding/binary"
	"math/rand/v2"
	"sync/atomic"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type spanIDOverrideKey struct{}
type traceIDOverrideKey struct{}

// idOverride holds a deterministic ID that is consumed exactly once by
// the IDGenerator. After the first span is created from this context,
// subsequent spans fall back to random IDs.
type idOverride[T any] struct {
	id       T
	consumed atomic.Bool
}

func (o *idOverride[T]) take() (T, bool) {
	if o.consumed.CompareAndSwap(false, true) {
		return o.id, true
	}
	var zero T
	return zero, false
}

// WithSpanID returns a context that instructs the IDGenerator to use
// the given SpanID for the next span created from this context.
// The override is consumed once: only the immediately next span gets
// the deterministic ID; subsequent spans get random IDs.
func WithSpanID(ctx context.Context, id trace.SpanID) context.Context {
	if id.IsValid() {
		return context.WithValue(ctx, spanIDOverrideKey{}, &idOverride[trace.SpanID]{id: id})
	}
	return ctx
}

// WithTraceID returns a context that instructs the IDGenerator to use
// the given TraceID for the next root span created from this context.
// The override is consumed once, same as WithSpanID.
func WithTraceID(ctx context.Context, id trace.TraceID) context.Context {
	if id.IsValid() {
		return context.WithValue(ctx, traceIDOverrideKey{}, &idOverride[trace.TraceID]{id: id})
	}
	return ctx
}

// DeterministicIDGenerator is an IDGenerator that checks the context for
// a SpanID override, falling back to random generation otherwise.
// The fallback random code is from randomIdGenerator in opentelemetry
type DeterministicIDGenerator struct{}

var _ sdktrace.IDGenerator = &DeterministicIDGenerator{}

func (g *DeterministicIDGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	tid := trace.TraceID{}
	if override, ok := ctx.Value(traceIDOverrideKey{}).(*idOverride[trace.TraceID]); ok {
		if id, taken := override.take(); taken {
			tid = id
		}
	}
	if !tid.IsValid() {
		for {
			binary.NativeEndian.PutUint64(tid[:8], rand.Uint64())
			binary.NativeEndian.PutUint64(tid[8:], rand.Uint64())
			if tid.IsValid() {
				break
			}
		}
	}
	if override, ok := ctx.Value(spanIDOverrideKey{}).(*idOverride[trace.SpanID]); ok {
		if id, taken := override.take(); taken {
			return tid, id
		}
	}
	sid := trace.SpanID{}
	for {
		binary.NativeEndian.PutUint64(sid[:], rand.Uint64())
		if sid.IsValid() {
			break
		}
	}
	return tid, sid
}

func (g *DeterministicIDGenerator) NewSpanID(ctx context.Context, _ trace.TraceID) trace.SpanID {
	if override, ok := ctx.Value(spanIDOverrideKey{}).(*idOverride[trace.SpanID]); ok {
		if id, taken := override.take(); taken {
			return id
		}
	}
	sid := trace.SpanID{}
	for {
		binary.NativeEndian.PutUint64(sid[:], rand.Uint64())
		if sid.IsValid() {
			break
		}
	}
	return sid
}
