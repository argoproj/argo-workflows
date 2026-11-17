package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
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
