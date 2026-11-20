package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

// ContextWithNode installs a started node's span as the active span, so a
// child span started later (in another operate cycle) parents under it. An
// unknown workflow or node leaves the context untouched.
func TestContextWithNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	trc, _, err := CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	assert.Equal(t, ctx, trc.ContextWithNode(ctx, "wf", "ns", "node-1"), "unknown workflow")

	wfCtx := trc.RecordStartWorkflow(ctx, "wf", "ns")
	assert.Equal(t, wfCtx, trc.ContextWithNode(wfCtx, "wf", "ns", "node-1"), "unknown node")

	nodeCtx := trc.RecordStartNode(wfCtx, "wf", "ns", "node-1", "Pod", wfv1.NodePending, "")
	want := trace.SpanFromContext(nodeCtx).SpanContext()
	require.True(t, want.IsValid())

	got := trace.SpanFromContext(trc.ContextWithNode(ctx, "wf", "ns", "node-1")).SpanContext()
	assert.Equal(t, want.TraceID(), got.TraceID())
	assert.Equal(t, want.SpanID(), got.SpanID())
	assert.NotEqual(t, want.SpanID(), trace.SpanFromContext(wfCtx).SpanContext().SpanID(), "the node span, not the workflow span")
}

// A later pod reconciliation of a node that is still open records nodePhase
// under that node span, not under podReconciliation.
func TestChangeNodePhaseFromPodReconciliationParentsUnderOpenNode(t *testing.T) {
	hook := logging.NewTestHook()
	logger := logging.NewTestLogger(logging.Debug, logging.Text, hook)
	ctx := logging.WithLogger(t.Context(), logger)
	trc, exporter, err := CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	wfCtx := trc.RecordStartWorkflow(ctx, "wf", "ns")
	nodeCtx := trc.RecordStartNode(wfCtx, "wf", "ns", "node-1", "Pod", wfv1.NodePending, "")
	nodeSpanID := trace.SpanFromContext(nodeCtx).SpanContext().SpanID()

	later := trc.RecoverWorkflowContext(ctx, "ns/wf")
	reconcileCtx, reconcileSpan := trc.StartReconcileWorkflow(later)
	defer reconcileSpan.End()
	podCtx, podSpan := trc.StartPodReconciliation(reconcileCtx)
	defer podSpan.End()

	hook.Reset()
	exporter.Reset()
	trc.ChangeNodePhase(podCtx, "ns/wf", "node-1", wfv1.NodeRunning, "")
	// Ending the node exports the Running phase span.
	trc.ChangeNodePhase(podCtx, "ns/wf", "node-1", wfv1.NodeFailed, "done")

	assertNoTraceParentageErrors(t, hook)
	running := spanByPhase(t, exporter, "Running")
	assert.Equal(t, nodeSpanID, running.Parent().SpanID(), "nodePhase parents under the node span, not podReconciliation")
	assert.NotEqual(t, podSpan.SpanContext().SpanID(), running.Parent().SpanID())
}

func assertNoTraceParentageErrors(t *testing.T, hook *logging.TestHook) {
	t.Helper()
	for _, e := range hook.AllEntries() {
		if e.Level != logging.Error {
			continue
		}
		if e.Msg == "incorrect trace parentage" || e.Msg == "tracing expect node for node phase failed" {
			t.Errorf("logged %q fields=%v", e.Msg, e.Fields)
		}
	}
}

func spanByPhase(t *testing.T, exporter *telemetry.TestTracingExporter, phase string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, sp := range exporter.GetSpansByName("nodePhase") {
		for _, attr := range sp.Attributes() {
			if attr.Key == attribute.Key(telemetry.AttribNodePhase) && attr.Value.AsString() == phase {
				return sp
			}
		}
	}
	t.Fatalf("nodePhase span with phase %q not found", phase)
	return nil
}
