package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/tracing"
)

// On a later operate cycle the node already exists, so executeContainer does
// not call initializeExecutableNode and the context only carries the workflow
// span. createWorkflowPod must still parent under the node span. Otherwise
// StartCreateWorkflowPod logs incorrect trace parentage with
// actualParent=workflow (#17472).
func TestLaterOperateCycleCreateWorkflowPodParentsUnderNode(t *testing.T) {
	hook := logging.NewTestHook()
	logger := logging.NewTestLogger(logging.Info, logging.Text, hook)
	ctx := logging.WithLogger(t.Context(), logger)

	wf := wfv1.MustUnmarshalWorkflow(tracingTestWorkflow)
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	te, exporter, err := tracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)
	controller.tracing = te

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	wfCtx := woc.controller.tracing.RecordStartWorkflow(ctx, wf.Name, wf.Namespace)
	tmpl := wf.Spec.Templates[0].DeepCopy()
	org := &wfv1.WorkflowStep{Name: wf.Name, Template: tmpl.Name}
	nodeCtx, node := woc.initializeExecutableNode(wfCtx, wf.Name, wfv1.NodeTypePod, "", tmpl, org, "", wfv1.NodePending, &wfv1.NodeFlag{}, true)
	nodeSpanID := trace.SpanFromContext(nodeCtx).SpanContext().SpanID()
	wfSpanID := trace.SpanFromContext(wfCtx).SpanContext().SpanID()
	require.NotEqual(t, wfSpanID, nodeSpanID)

	// Next operate cycle restores the workflow span only.
	later := woc.controller.tracing.RecoverWorkflowContext(ctx, wf.Namespace+"/"+wf.Name)
	require.Equal(t, wfSpanID, trace.SpanFromContext(later).SpanContext().SpanID())

	hook.Reset()
	exporter.Reset()
	_, err = woc.executeContainer(later, node.Name, "", tmpl, org, &executeTemplateOpts{})
	require.NoError(t, err)

	assertNoTraceParentageErrors(t, hook)
	podSpan, err := exporter.GetSpanByName("createWorkflowPod")
	require.NoError(t, err)
	assert.Equal(t, nodeSpanID, podSpan.Parent().SpanID(), "createWorkflowPod parents under the node span")
	assert.NotEqual(t, wfSpanID, podSpan.Parent().SpanID(), "not the workflow span left active on a later cycle")
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
