package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestRecordStartWorkflowParent(t *testing.T) {
	const (
		callerTraceID = "0af7651916cd43dd8448eb211c80319c"
		callerSpanID  = "b7ad6b7169203331"
	)
	tests := []struct {
		name        string
		annotations map[string]string
		wantParent  bool
	}{
		{name: "no annotations", annotations: nil},
		{name: "unrelated annotations", annotations: map[string]string{"traceparent": "00-" + callerTraceID + "-" + callerSpanID + "-01"}},
		{name: "malformed traceparent", annotations: map[string]string{"opentelemetry.io/traceparent": "not-a-traceparent"}},
		{
			name:        "valid traceparent",
			annotations: map[string]string{"opentelemetry.io/traceparent": "00-" + callerTraceID + "-" + callerSpanID + "-01"},
			wantParent:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			trc, exporter, err := CreateDefaultTestTracing(ctx)
			require.NoError(t, err)

			trc.RecordStartWorkflow(ctx, "my-wf", "my-ns", tt.annotations)
			trc.ChangeWorkflowPhase(trc.RecoverWorkflowContext(ctx, "my-ns/my-wf"), "my-wf", "my-ns", wfv1.WorkflowRunning)
			trc.EndWorkflow(ctx, "my-wf", "my-ns", wfv1.WorkflowSucceeded)

			span, err := exporter.GetSpanByName("workflow")
			require.NoError(t, err)
			assert.Equal(t, "my-ns/my-wf", span.SpanContext().TraceState().Get("workflow"))

			phaseSpan, err := exporter.GetSpanByName("workflowPhase")
			require.NoError(t, err)
			assert.Equal(t, span.SpanContext().TraceID(), phaseSpan.SpanContext().TraceID())
			assert.Equal(t, span.SpanContext().SpanID(), phaseSpan.Parent().SpanID())

			if tt.wantParent {
				assert.Equal(t, callerTraceID, span.SpanContext().TraceID().String())
				assert.Equal(t, callerSpanID, span.Parent().SpanID().String())
				assert.True(t, span.Parent().IsRemote())
			} else {
				assert.False(t, span.Parent().IsValid(), "workflow span should be a root")
				assert.NotEqual(t, callerTraceID, span.SpanContext().TraceID().String())
			}
		})
	}
}

func TestRecordStartWorkflowUnsampledParent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	trc, exporter, err := CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	wfCtx := trc.RecordStartWorkflow(ctx, "my-wf", "my-ns", map[string]string{
		"opentelemetry.io/traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00",
	})
	trc.EndWorkflow(ctx, "my-wf", "my-ns", wfv1.WorkflowSucceeded)

	// The caller decided not to sample, so the workflow follows that decision
	sc := trace.SpanContextFromContext(wfCtx)
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", sc.TraceID().String())
	assert.False(t, sc.IsSampled())
	assert.Empty(t, exporter.GetSpansByName("workflow"))
}
