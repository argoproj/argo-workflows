package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/tracing"
)

var tracingTestWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: tracing-test
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: busybox
      command: [echo, hello]
`

// TestTracingReconcileSpans verifies that reconciliation creates the expected spans
// with correct attributes, hierarchy, and trace context.
func TestTracingReconcileSpans(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	te, exporter, err := tracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	cancel, controller := newController(ctx)
	defer cancel()

	controller.tracing = te

	wf := wfv1.MustUnmarshalWorkflow(tracingTestWorkflow)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Verify workflow was processed
	assert.NotEmpty(t, woc.wf.Status.Nodes, "workflow should have nodes after reconciliation")

	// Verify reconcileWorkflow span
	reconcileSpan, err := exporter.GetSpanByName("reconcileWorkflow")
	require.NoError(t, err, "reconcileWorkflow span should exist")
	assert.Equal(t, "reconcileWorkflow", reconcileSpan.Name())
	assert.Equal(t, trace.SpanKindInternal, reconcileSpan.SpanKind())

	// Verify reconcileTaskResults span
	taskResultsSpan, err := exporter.GetSpanByName("reconcileTaskResults")
	require.NoError(t, err, "reconcileTaskResults span should exist")
	assert.Equal(t, "reconcileTaskResults", taskResultsSpan.Name())
	assert.Equal(t, trace.SpanKindInternal, taskResultsSpan.SpanKind())

	// Verify hierarchy: reconcileTaskResults -> reconcileWorkflow
	assert.Equal(t, reconcileSpan.SpanContext().SpanID(), taskResultsSpan.Parent().SpanID(),
		"reconcileTaskResults parent should be reconcileWorkflow")

	// Verify both spans share the same trace ID
	assert.Equal(t, reconcileSpan.SpanContext().TraceID(), taskResultsSpan.SpanContext().TraceID(),
		"reconcileTaskResults should share trace ID with reconcileWorkflow")
}
