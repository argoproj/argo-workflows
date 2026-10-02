package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/tracing"
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

var tracingTestWorkflowWithParent = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: tracing-test-parent
  namespace: default
  annotations:
    opentelemetry.io/traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: busybox
      command: [echo, hello]
`

// TestTracingContinuesAnnotatedTrace verifies that a workflow annotated with a traceparent
// continues the caller's trace, and passes it on to its pods.
func TestTracingContinuesAnnotatedTrace(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	te, exporter, err := tracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	cancel, controller := newController(ctx)
	defer cancel()

	controller.tracing = te

	wf := wfv1.MustUnmarshalWorkflow(tracingTestWorkflowWithParent)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	const callerTraceID = "0af7651916cd43dd8448eb211c80319c"

	reconcileSpan, err := exporter.GetSpanByName("reconcileWorkflow")
	require.NoError(t, err)
	assert.Equal(t, callerTraceID, reconcileSpan.SpanContext().TraceID().String())

	assert.Equal(t, callerTraceID, woc.wf.Annotations[common.AnnotationKeyTraceID])

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	traceparent := ""
	for _, env := range pods.Items[0].Spec.Containers[0].Env {
		if env.Name == "TRACEPARENT" {
			traceparent = env.Value
		}
	}
	assert.Contains(t, traceparent, "-"+callerTraceID+"-", "pod TRACEPARENT should continue the caller's trace")
	assert.NotContains(t, traceparent, "b7ad6b7169203331", "pod TRACEPARENT should have its own span ID")
}
