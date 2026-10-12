package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestExecutionDiffFreezesBindingsAndSeparatesLayers(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w}
	action := ActionDefinition{InputSchema: map[string]any{"type": "object"}, OutputSchema: map[string]any{"type": "object"}, Template: &wfv1.Template{Name: "run", Container: &corev1.Container{Image: "alpine:3.22", Command: []string{"echo"}}}}
	_, err := s.resource("actiondefinitions").Create(ctx, object("ActionDefinition", "before", "lab", action), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowactionbindings").Create(ctx, object("WorkflowActionBinding", "calls", "lab", ActionBinding{WorkflowTemplate: "native", Calls: []ActionCall{{Template: "main", ActionDefinition: "before"}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	binding, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(binding.Object, "calls", "spec", "actionBinding"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, binding, metav1.UpdateOptions{})
	require.NoError(t, err)
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	raw, err := h.submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "layered"})
	require.NoError(t, err)
	wf := raw.(*wfv1.Workflow)
	wf.UID = "execution-uid"
	_, err = w.client.ArgoprojV1alpha1().Workflows("lab").Update(ctx, wf, metav1.UpdateOptions{})
	require.NoError(t, err)
	// Replace live definitions and remove the old action entirely. The left side
	// must still show the recorded binding/implementation, without extra writes.
	action.Template.Container.Image = "alpine:3.21"
	action.InputSchema["additionalProperties"] = false
	_, err = s.resource("actiondefinitions").Create(ctx, object("ActionDefinition", "after", "lab", action), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowactionbindings").Create(ctx, object("WorkflowActionBinding", "calls-next", "lab", ActionBinding{WorkflowTemplate: "native", Calls: []ActionCall{{Template: "main", ActionDefinition: "after"}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(binding.Object, "calls-next", "spec", "actionBinding"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, binding, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, s.resource("actiondefinitions").Delete(ctx, "before", metav1.DeleteOptions{}))
	set, err := s.get(ctx, "parametersets", "sample")
	require.NoError(t, err)
	saved, err := s.Save(ctx, "sample", SaveRequest{Template: "typed", Values: map[string]any{"count": 4, "enabled": false}, ResourceVersion: set.GetResourceVersion()})
	require.NoError(t, err)
	next, _, _ := unstructured.NestedString(saved.Object, "spec", "revision")
	req := ExecutionDiffRequest{Source: SourceExecution{Workflow: wf.Name, UID: string(wf.UID)}, Revision: next}
	result, err := h.executionDiff(ctx, s, req)
	require.NoError(t, err)
	layers := map[string]ExecutionDiffLayer{}
	for _, layer := range result.Layers {
		layers[layer.Name] = layer
		require.True(t, layer.Available)
	}
	for _, name := range []string{"Action binding", "Action contracts", "Implementations and workflow structure", "Compiled inputs"} {
		require.NotEmpty(t, layers[name].Changes, name)
	}
	require.Contains(t, p.Clone(layers["Action binding"].Changes).([]any)[0].(map[string]any)["before"], map[string]any{"template": "main", "actionDefinition": "before"})
	require.Contains(t, result.DefinitionVersions["source"], "action/before")
	current, err := w.client.ArgoprojV1alpha1().Workflows("lab").Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, wf, current)
	require.Equal(t, 1, w.createCalls)
	req.Source.UID = "replaced"
	_, err = h.executionDiff(ctx, s, req)
	require.ErrorContains(t, err, "identity changed")
	req.Source.UID = string(wf.UID)
	other, err := s.Save(ctx, "other", SaveRequest{Template: "typed", Values: map[string]any{"count": 4}})
	require.NoError(t, err)
	req.Revision, _, _ = unstructured.NestedString(other.Object, "spec", "revision")
	_, err = h.executionDiff(ctx, s, req)
	require.ErrorContains(t, err, "another parameter set")
}

func TestExecutionDefinitionsFollowRetryAndReportLegacyAbsence(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w}
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	raw, err := h.submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "receipt-chain"})
	require.NoError(t, err)
	wf := raw.(*wfv1.Workflow)
	wf.UID = "receipt-uid"
	snapshot, err := s.executionDefinitions(ctx, wf)
	require.NoError(t, err)
	require.Equal(t, p.Clone(preview.Definitions), p.Clone(snapshot))
	wf.Annotations[p.Group+"/retry"] = "attempt"
	saved := preview.Definitions.DeepCopy()
	saved.ParameterBinding.Form = "historical-form"
	attempt := parameterapi.ParameterRetrySpec{Workflow: wf.Name, WorkflowUID: string(wf.UID), Revision: revision, Definitions: saved}
	obj, err := s.resource("parameterretries").Create(ctx, object("ParameterRetry", "attempt", "lab", attempt), metav1.CreateOptions{})
	require.NoError(t, err)
	snapshot, err = s.executionDefinitions(ctx, wf)
	require.NoError(t, err)
	require.Equal(t, "historical-form", snapshot.ParameterBinding.Form)
	delete(obj.Object["spec"].(map[string]any), "definitions")
	_, err = s.resource("parameterretries").Update(ctx, obj, metav1.UpdateOptions{})
	require.NoError(t, err)
	snapshot, err = s.executionDefinitions(ctx, wf)
	require.NoError(t, err)
	require.Nil(t, snapshot, "do not replace missing adopted snapshot with original submission definitions")
	obj.Object["spec"].(map[string]any)["workflowUID"] = "another"
	_, err = s.resource("parameterretries").Update(ctx, obj, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.executionDefinitions(ctx, wf)
	require.ErrorContains(t, err, "does not match")
}

func TestExecutionDiffLegacyReceiptAndForgedDefinitions(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w}
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	raw, err := h.submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "legacy"})
	require.NoError(t, err)
	wf := raw.(*wfv1.Workflow)
	wf.UID = "legacy-uid"
	_, err = w.client.ArgoprojV1alpha1().Workflows("lab").Update(ctx, wf, metav1.UpdateOptions{})
	require.NoError(t, err)
	receipt, err := s.get(ctx, "workflowsubmissions", wf.Annotations[p.Group+"/submission"])
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(receipt.Object, "forged", "spec", "definitions", "parameterBinding", "workflowTemplate"))
	_, err = s.resource("workflowsubmissions").Update(ctx, receipt, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.ErrorContains(t, s.ValidateNativeWorkflow(ctx, wf, nil), "snapshot differs")
	delete(receipt.Object["spec"].(map[string]any), "definitions")
	_, err = s.resource("workflowsubmissions").Update(ctx, receipt, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, s.ValidateNativeWorkflow(ctx, wf, nil), "old receipts remain admissible")
	result, err := h.executionDiff(ctx, s, ExecutionDiffRequest{Source: SourceExecution{Workflow: wf.Name, UID: string(wf.UID)}, Revision: revision})
	require.NoError(t, err)
	for i, layer := range result.Layers {
		if i < 5 {
			require.False(t, layer.Available)
			require.Contains(t, layer.Notice, "not recorded")
		} else {
			require.True(t, layer.Available)
			require.Empty(t, layer.Changes)
		}
	}
}
