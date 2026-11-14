package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestEditExecutionAndResubmitPreservesOriginalSnapshot(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w}
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	first, err := h.submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "original"})
	require.NoError(t, err)
	original := first.(*wfv1.Workflow)
	original.UID = "original-uid"
	_, err = w.client.ArgoprojV1alpha1().Workflows("lab").Update(ctx, original, metav1.UpdateOptions{})
	require.NoError(t, err)
	source := SourceExecution{Workflow: original.Name, UID: string(original.UID)}
	draft, err := s.get(ctx, "parametersets", "sample")
	require.NoError(t, err)
	saved, err := s.Save(ctx, "sample", SaveRequest{Template: "typed", Values: map[string]any{"count": 4, "enabled": false}, ResourceVersion: draft.GetResourceVersion()})
	require.NoError(t, err)
	next, _, _ := unstructured.NestedString(saved.Object, "spec", "revision")
	context, err := h.editContext(ctx, s, source)
	require.NoError(t, err)
	require.Equal(t, "standard", context.Binding)
	require.Equal(t, revision, context.Revision.GetName(), "load the executed snapshot even if the draft advanced")
	require.Equal(t, next, context.Set.Object["spec"].(map[string]any)["revision"])
	preview, err = s.Preview(ctx, "standard", next)
	require.NoError(t, err)
	req := SubmitRequest{Binding: "standard", Revision: next, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "edited", SourceExecution: &source}
	w.loseResponse = true
	_, err = h.submit(ctx, s, req)
	require.ErrorContains(t, err, "response lost")
	second, err := h.submit(ctx, s, req)
	require.NoError(t, err)
	newWorkflow := second.(*wfv1.Workflow)
	require.NotEqual(t, original.Name, newWorkflow.Name)
	require.Equal(t, string(original.UID), newWorkflow.Annotations[p.Group+"/source-workflow-uid"])
	require.Equal(t, 2, w.createCalls, "response loss must not duplicate the new execution")
	unchanged, err := w.client.ArgoprojV1alpha1().Workflows("lab").Get(ctx, original.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, original, unchanged)
	_, err = h.editContext(ctx, s, SourceExecution{Workflow: original.Name, UID: "replaced-uid"})
	require.ErrorContains(t, err, "identity changed")
	req.IdempotencyKey = "wrong-binding"
	req.Binding = "different"
	_, err = h.submit(ctx, s, req)
	require.ErrorContains(t, err, "retain the source binding")
	other, err := s.Save(ctx, "other", SaveRequest{Template: "typed", Values: map[string]any{"count": 4}})
	require.NoError(t, err)
	req.Binding = "standard"
	req.Revision, _, _ = unstructured.NestedString(other.Object, "spec", "revision")
	_, err = h.submit(ctx, s, req)
	require.ErrorContains(t, err, "retain the source binding and parameter set")
	require.Equal(t, 2, w.createCalls)
}

func TestEditContextAfterOriginalRetryOfAdoptedRevision(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w}
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	first, err := h.submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "mixed-retry"})
	require.NoError(t, err)
	wf := first.(*wfv1.Workflow)
	wf.UID = "mixed-retry-uid"
	draft, err := s.get(ctx, "parametersets", "sample")
	require.NoError(t, err)
	saved, err := s.Save(ctx, "sample", SaveRequest{Template: "typed", Values: map[string]any{"count": 4, "enabled": false}, ResourceVersion: draft.GetResourceVersion()})
	require.NoError(t, err)
	next, _, _ := unstructured.NestedString(saved.Object, "spec", "revision")
	wf.Annotations[p.Group+"/revision"] = next
	wf.Annotations[p.Group+"/retry"] = "original-after-partial"
	_, err = w.client.ArgoprojV1alpha1().Workflows("lab").Update(ctx, wf, metav1.UpdateOptions{})
	require.NoError(t, err)
	receipt := object("ParameterRetry", "original-after-partial", "lab", map[string]any{"workflow": wf.Name, "workflowUID": string(wf.UID), "revision": next})
	_, err = s.resource("parameterretries").Create(ctx, receipt, metav1.CreateOptions{})
	require.NoError(t, err)
	result, err := h.editContext(ctx, s, SourceExecution{Workflow: wf.Name, UID: string(wf.UID)})
	require.NoError(t, err)
	require.Equal(t, next, result.Revision.GetName())
	receipt.Object["spec"].(map[string]any)["workflowUID"] = "another-workflow"
	_, err = s.resource("parameterretries").Update(ctx, receipt, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = h.editContext(ctx, s, SourceExecution{Workflow: wf.Name, UID: string(wf.UID)})
	require.ErrorContains(t, err, "does not match")
}
