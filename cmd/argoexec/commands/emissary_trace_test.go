package commands

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// ctxWithSpan returns a context carrying a fixed, sampled span context, so the
// traceparent injected from it is predictable.
func ctxWithSpan(t *testing.T) context.Context {
	t.Helper()
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))
}

const wantTraceParent = "TRACEPARENT=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

func TestInjectTraceParent(t *testing.T) {
	t.Run("AddsToChildEnv", func(t *testing.T) {
		env := injectTraceParent(ctxWithSpan(t), []string{"FOO=bar"})
		assert.Equal(t, []string{"FOO=bar", wantTraceParent}, env)
	})

	t.Run("ReplacesPodTraceParent", func(t *testing.T) {
		// The controller injects a pod-level traceparent into every container.
		// The child must see this process's span instead, exactly once.
		env := injectTraceParent(ctxWithSpan(t), []string{
			"FOO=bar",
			"TRACEPARENT=00-00000000000000000000000000000001-0000000000000002-01",
		})
		assert.Equal(t, []string{"FOO=bar", wantTraceParent}, env)
	})

	t.Run("NoSpanLeavesEnvAlone", func(t *testing.T) {
		env := injectTraceParent(context.Background(), []string{"FOO=bar"})
		assert.Equal(t, []string{"FOO=bar"}, env)
	})

	t.Run("DoesNotMutateCallersProcessEnv", func(t *testing.T) {
		// The injection must reach the child only: argoexec's own
		// context variables stay as the pod spec set them.
		t.Setenv("TRACEPARENT", "00-00000000000000000000000000000001-0000000000000002-01")
		injectTraceParent(ctxWithSpan(t), []string{})
		assert.Equal(t, "00-00000000000000000000000000000001-0000000000000002-01", os.Getenv("TRACEPARENT"))
	})
}

func TestSetEnvVar(t *testing.T) {
	t.Run("AppendsWhenAbsent", func(t *testing.T) {
		assert.Equal(t, []string{"FOO=bar", "BAZ=qux"}, setEnvVar([]string{"FOO=bar"}, "BAZ", "qux"))
	})

	t.Run("ReplacesInPlace", func(t *testing.T) {
		assert.Equal(t, []string{"FOO=new", "BAZ=qux"}, setEnvVar([]string{"FOO=old", "BAZ=qux"}, "FOO", "new"))
	})

	t.Run("EmptyValue", func(t *testing.T) {
		assert.Equal(t, []string{"FOO="}, setEnvVar([]string{"FOO=old"}, "FOO", ""))
	})

	t.Run("DoesNotMatchLongerName", func(t *testing.T) {
		// TRACEPARENT_EXTRA must not be mistaken for TRACEPARENT.
		assert.Equal(t, []string{"FOOBAR=keep", "FOO=new"}, setEnvVar([]string{"FOOBAR=keep"}, "FOO", "new"))
	})
}
