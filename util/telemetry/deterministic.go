package telemetry

import (
	"crypto/sha256"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

func deterministicHash(kind string, parts ...string) [sha256.Size]byte {
	return sha256.Sum256([]byte(kind + "\x00" + strings.Join(parts, "\x00")))
}

// DeterministicTraceID generates a TraceID deterministically from the given
// input strings. Callers should pass strings in coarsest-to-finest order
// and must use the same order to reproduce the same ID.
//
// Inputs are joined with a null byte separator to avoid ambiguity.
func DeterministicTraceID(parts ...string) trace.TraceID {
	for {
		h := deterministicHash("trace", parts...)
		var id trace.TraceID
		copy(id[:], h[:16])
		if id.IsValid() {
			return id
		}
		parts = append(parts, "x")
	}
}

// DeterministicSpanID generates a SpanID deterministically from the given
// input strings. Same conventions as DeterministicTraceID.
func DeterministicSpanID(parts ...string) trace.SpanID {
	for {
		h := deterministicHash("span", parts...)
		var id trace.SpanID
		copy(id[:], h[:8])
		if id.IsValid() {
			return id
		}
		parts = append(parts, "x")
	}
}
