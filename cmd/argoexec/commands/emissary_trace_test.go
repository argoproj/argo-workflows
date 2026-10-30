package commands

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
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

	t.Run("OverridesPodTraceParent", func(t *testing.T) {
		// The controller injects a pod-level traceparent into every container.
		// The child must see this process's span instead: exec.Cmd uses the
		// last value of a duplicated key, so ours must come after it.
		podTraceParent := "TRACEPARENT=00-00000000000000000000000000000001-0000000000000002-01"
		env := injectTraceParent(ctxWithSpan(t), []string{"FOO=bar", podTraceParent})
		assert.Equal(t, []string{"FOO=bar", podTraceParent, wantTraceParent}, env)
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

func TestWithTaskTraceContext(t *testing.T) {
	const traceParent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	t.Run("ContinuesTheTasksTrace", func(t *testing.T) {
		ctx := withTaskTraceContext(context.Background(), []poolpod.EnvVar{
			{Name: "TRACEPARENT", Value: "00-00000000000000000000000000000001-0000000000000002-01"},
			{Name: "TRACEPARENT", Value: traceParent},
			{Name: "TRACESTATE", Value: "k=v"},
		})
		sc := trace.SpanContextFromContext(ctx)
		assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", sc.TraceID().String(), "the last TRACEPARENT wins")
		assert.Equal(t, "00f067aa0ba902b7", sc.SpanID().String())
		assert.Equal(t, "k=v", sc.TraceState().String())
		assert.True(t, sc.IsRemote())
	})

	t.Run("NoTraceParentNoParent", func(t *testing.T) {
		ctx := withTaskTraceContext(context.Background(), []poolpod.EnvVar{{Name: "TRACESTATE", Value: "k=v"}})
		assert.False(t, trace.SpanContextFromContext(ctx).IsValid())
		assert.Empty(t, trace.SpanContextFromContext(ctx).TraceState().String())
	})

	t.Run("OnlyTheExactName", func(t *testing.T) {
		// As os.LookupEnv reads a pod's env: names are case-sensitive.
		ctx := withTaskTraceContext(context.Background(), []poolpod.EnvVar{{Name: "traceparent", Value: traceParent}})
		assert.False(t, trace.SpanContextFromContext(ctx).IsValid())
	})
}
