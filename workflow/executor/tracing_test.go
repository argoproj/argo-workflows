package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	exectracing "github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// TestTracingArtifactLoadSpans verifies the complete artifact loading span hierarchy:
// runInitContainer -> loadArtifacts -> loadArtifact -> unarchiveArtifact
// Also verifies span attributes, SpanKind, and parent relationships.
func TestTracingArtifactLoadSpans(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	tr, exporter, err := exectracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	// Create full span chain
	ctx, initSpan := tr.StartRunInitContainer(ctx, fakeWorkflow, fakeNamespace)
	ctx, loadArtifactsSpan := tr.StartLoadArtifacts(ctx)
	ctx, loadArtifactSpan := tr.StartLoadArtifact(ctx, "/tmp/test.txt")
	_, unarchiveSpan := tr.StartUnarchiveArtifact(ctx, "tar.gz")

	// End all spans
	unarchiveSpan.End()
	loadArtifactSpan.End()
	loadArtifactsSpan.End()
	initSpan.End()

	// Verify runInitContainer span and attributes
	initCollected, err := exporter.GetSpanByName("runInitContainer")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, initCollected.SpanKind())
	initAttribs := attribute.NewSet(initCollected.Attributes()...)
	val, ok := initAttribs.Value(attribute.Key(telemetry.AttribWorkflowName))
	require.True(t, ok)
	assert.Equal(t, fakeWorkflow, val.AsString())
	val, ok = initAttribs.Value(attribute.Key(telemetry.AttribWorkflowNamespace))
	require.True(t, ok)
	assert.Equal(t, fakeNamespace, val.AsString())

	// Verify loadArtifacts span
	loadArtifactsCollected, err := exporter.GetSpanByName("loadArtifacts")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, loadArtifactsCollected.SpanKind())
	assert.Equal(t, initCollected.SpanContext().SpanID(), loadArtifactsCollected.Parent().SpanID(),
		"loadArtifacts parent should be runInitContainer")

	// Verify loadArtifact span and attributes
	loadArtifactCollected, err := exporter.GetSpanByName("loadArtifact")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, loadArtifactCollected.SpanKind())
	loadArtifactAttribs := attribute.NewSet(loadArtifactCollected.Attributes()...)
	val, ok = loadArtifactAttribs.Value(attribute.Key(telemetry.AttribArtifactPath))
	require.True(t, ok)
	assert.Equal(t, "/tmp/test.txt", val.AsString())
	assert.Equal(t, loadArtifactsCollected.SpanContext().SpanID(), loadArtifactCollected.Parent().SpanID(),
		"loadArtifact parent should be loadArtifacts")

	// Verify unarchiveArtifact span and attributes
	unarchiveCollected, err := exporter.GetSpanByName("unarchiveArtifact")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, unarchiveCollected.SpanKind())
	unarchiveAttribs := attribute.NewSet(unarchiveCollected.Attributes()...)
	val, ok = unarchiveAttribs.Value(attribute.Key(telemetry.AttribArtifactArchive))
	require.True(t, ok)
	assert.Equal(t, "tar.gz", val.AsString())
	assert.Equal(t, loadArtifactCollected.SpanContext().SpanID(), unarchiveCollected.Parent().SpanID(),
		"unarchiveArtifact parent should be loadArtifact")

	// All spans should share the same trace ID
	traceID := initCollected.SpanContext().TraceID()
	assert.Equal(t, traceID, loadArtifactsCollected.SpanContext().TraceID())
	assert.Equal(t, traceID, loadArtifactCollected.SpanContext().TraceID())
	assert.Equal(t, traceID, unarchiveCollected.SpanContext().TraceID())
}

// TestTracingArtifactSaveSpans verifies the complete artifact saving span hierarchy:
// runWaitContainer -> saveArtifacts -> saveArtifact -> archiveArtifact
// Also verifies span attributes, SpanKind, and parent relationships.
func TestTracingArtifactSaveSpans(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	tr, exporter, err := exectracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	// Create full span chain
	ctx, waitSpan := tr.StartRunWaitContainer(ctx, fakeWorkflow, fakeNamespace)
	ctx, saveArtifactsSpan := tr.StartSaveArtifacts(ctx)
	ctx, saveArtifactSpan := tr.StartSaveArtifact(ctx)
	_, archiveSpan := tr.StartArchiveArtifact(ctx)

	// End all spans
	archiveSpan.End()
	saveArtifactSpan.End()
	saveArtifactsSpan.End()
	waitSpan.End()

	// Verify runWaitContainer span and attributes
	waitCollected, err := exporter.GetSpanByName("runWaitContainer")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, waitCollected.SpanKind())
	waitAttribs := attribute.NewSet(waitCollected.Attributes()...)
	val, ok := waitAttribs.Value(attribute.Key(telemetry.AttribWorkflowName))
	require.True(t, ok)
	assert.Equal(t, fakeWorkflow, val.AsString())
	val, ok = waitAttribs.Value(attribute.Key(telemetry.AttribWorkflowNamespace))
	require.True(t, ok)
	assert.Equal(t, fakeNamespace, val.AsString())

	// Verify saveArtifacts span
	saveArtifactsCollected, err := exporter.GetSpanByName("saveArtifacts")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, saveArtifactsCollected.SpanKind())
	assert.Equal(t, waitCollected.SpanContext().SpanID(), saveArtifactsCollected.Parent().SpanID(),
		"saveArtifacts parent should be runWaitContainer")

	// Verify saveArtifact span
	saveArtifactCollected, err := exporter.GetSpanByName("saveArtifact")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, saveArtifactCollected.SpanKind())
	assert.Equal(t, saveArtifactsCollected.SpanContext().SpanID(), saveArtifactCollected.Parent().SpanID(),
		"saveArtifact parent should be saveArtifacts")

	// Verify archiveArtifact span
	archiveCollected, err := exporter.GetSpanByName("archiveArtifact")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, archiveCollected.SpanKind())
	assert.Equal(t, saveArtifactCollected.SpanContext().SpanID(), archiveCollected.Parent().SpanID(),
		"archiveArtifact parent should be saveArtifact")

	// All spans should share the same trace ID
	traceID := waitCollected.SpanContext().TraceID()
	assert.Equal(t, traceID, saveArtifactsCollected.SpanContext().TraceID())
	assert.Equal(t, traceID, saveArtifactCollected.SpanContext().TraceID())
	assert.Equal(t, traceID, archiveCollected.SpanContext().TraceID())
}

// TestTracingWaitContainerChildSpans verifies all spans that are direct children of runWaitContainer:
// createTaskResult, patchTaskResult, patchTaskResultLabels, waitWorkload
func TestTracingWaitContainerChildSpans(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	tr, exporter, err := exectracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	// Create parent span
	ctx, waitSpan := tr.StartRunWaitContainer(ctx, fakeWorkflow, fakeNamespace)

	// Create all child spans
	_, createSpan := tr.StartCreateTaskResult(ctx)
	createSpan.End()

	_, patchSpan := tr.StartPatchTaskResult(ctx)
	patchSpan.End()

	_, patchLabelsSpan := tr.StartPatchTaskResultLabels(ctx)
	patchLabelsSpan.End()

	_, workloadSpan := tr.StartWaitWorkload(ctx)
	workloadSpan.End()

	waitSpan.End()

	// Get parent span
	waitCollected, err := exporter.GetSpanByName("runWaitContainer")
	require.NoError(t, err)
	parentSpanID := waitCollected.SpanContext().SpanID()
	traceID := waitCollected.SpanContext().TraceID()

	// Verify createTaskResult
	createCollected, err := exporter.GetSpanByName("createTaskResult")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, createCollected.SpanKind())
	assert.Equal(t, parentSpanID, createCollected.Parent().SpanID(), "createTaskResult parent should be runWaitContainer")
	assert.Equal(t, traceID, createCollected.SpanContext().TraceID())

	// Verify patchTaskResult
	patchCollected, err := exporter.GetSpanByName("patchTaskResult")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, patchCollected.SpanKind())
	assert.Equal(t, parentSpanID, patchCollected.Parent().SpanID(), "patchTaskResult parent should be runWaitContainer")
	assert.Equal(t, traceID, patchCollected.SpanContext().TraceID())

	// Verify patchTaskResultLabels
	patchLabelsCollected, err := exporter.GetSpanByName("patchTaskResultLabels")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, patchLabelsCollected.SpanKind())
	assert.Equal(t, parentSpanID, patchLabelsCollected.Parent().SpanID(), "patchTaskResultLabels parent should be runWaitContainer")
	assert.Equal(t, traceID, patchLabelsCollected.SpanContext().TraceID())

	// Verify waitWorkload
	workloadCollected, err := exporter.GetSpanByName("waitWorkload")
	require.NoError(t, err)
	assert.Equal(t, trace.SpanKindInternal, workloadCollected.SpanKind())
	assert.Equal(t, parentSpanID, workloadCollected.Parent().SpanID(), "waitWorkload parent should be runWaitContainer")
	assert.Equal(t, traceID, workloadCollected.SpanContext().TraceID())
}

// TestTracingLoadArtifactsIntegration tests that the actual executor loadArtifacts method creates spans correctly.
func TestTracingLoadArtifactsIntegration(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	tr, exporter, err := exectracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)

	we := WorkflowExecutor{
		Template: wfv1.Template{
			Inputs: wfv1.Inputs{
				Artifacts: []wfv1.Artifact{},
			},
		},
		Tracing: tr,
	}

	// Create parent span and call actual executor method
	ctx, parentSpan := tr.StartRunInitContainer(ctx, fakeWorkflow, fakeNamespace)
	err = we.loadArtifacts(ctx, "")
	require.NoError(t, err)
	parentSpan.End()

	// Verify loadArtifacts span was created by the executor
	loadArtifactsSpan, err := exporter.GetSpanByName("loadArtifacts")
	require.NoError(t, err, "loadArtifacts span should exist")

	parentCollected, err := exporter.GetSpanByName("runInitContainer")
	require.NoError(t, err)

	assert.Equal(t, parentCollected.SpanContext().SpanID(), loadArtifactsSpan.Parent().SpanID(),
		"loadArtifacts parent should be runInitContainer")
}
