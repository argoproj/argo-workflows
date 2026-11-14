package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type interruptedInputService struct {
	*inputService
	interrupted bool
}

func (s *interruptedInputService) SetWorkflow(ctx context.Context, req *workflowapi.WorkflowSetRequest) (*wfv1.Workflow, error) {
	if !s.interrupted {
		s.interrupted = true
		return nil, fmt.Errorf("temporary outage before native write")
	}
	return s.inputService.SetWorkflow(ctx, req)
}

func secretInputFixture() FrozenInputForm {
	return FrozenInputForm{Model: p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"credential": map[string]any{"type": "object"}}, "required": []any{"credential"}, "additionalProperties": false}, SensitivePaths: []string{"/credential"}, SecretSchemas: map[string]map[string]any{"/credential": {"type": "string", "minLength": 8}}}, Mappings: []p.Mapping{{Path: "/credential", Parameter: "credential", Encoding: "secret-reference-json"}}}
}

func TestFrozenSensitiveInputWriteSubmitReceiptAndRetry(t *testing.T) {
	ctx, s, w, _ := previewFixture(t)
	service := &inputService{workflowService: w}
	h := Handler{Workflows: &interruptedInputService{inputService: service}, SecretWriter: EncryptedKubernetesSecrets{}}
	contract := secretInputFixture()
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "waiting-secret", Namespace: "lab", UID: "workflow-uid", ResourceVersion: "7", Annotations: map[string]string{p.Group + "/submission": "frozen"}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, Nodes: wfv1.Nodes{"node": {ID: "node", TemplateName: "input", Name: "waiting-secret.input", Type: wfv1.NodeTypeSuspend, Phase: wfv1.NodeRunning, Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "credential", ValueFrom: &wfv1.ValueFrom{Supplied: &wfv1.SuppliedValueFrom{}}}}}}}}}
	_, err := w.client.ArgoprojV1alpha1().Workflows("lab").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowsubmissions").Create(ctx, object("WorkflowSubmission", "frozen", "lab", map[string]any{"workflow": wf, "inputForms": map[string]FrozenInputForm{"input": contract}}), metav1.CreateOptions{})
	require.NoError(t, err)
	pending, err := h.pendingInputs(ctx, s, wf.Name)
	require.NoError(t, err)
	require.True(t, pending.EncryptedSecretInput)
	input := InputSecretContext{Workflow: wf.Name, UID: string(wf.UID), NodeID: "node", ContractDigest: pending.Items[0].ContractDigest}
	// There is no mutable catalog model: all writes must use the frozen contract.
	req := SecretWriteRequest{Input: &input, Path: "/credential", RequestID: "random-request-identifier", Value: "synthetic-runtime-token"}
	for _, change := range []func(*InputSecretContext){func(x *InputSecretContext) { x.UID = "other" }, func(x *InputSecretContext) { x.NodeID = "other" }, func(x *InputSecretContext) { x.ContractDigest = "other" }} {
		altered := input
		change(&altered)
		bad := req
		bad.Input = &altered
		_, err = h.writeSecret(ctx, s, bad)
		require.Error(t, err)
	}
	bad := req
	bad.Template = "typed"
	_, err = h.writeSecret(ctx, s, bad)
	require.ErrorContains(t, err, "mutually exclusive")
	bad = req
	bad.Value = "tiny"
	_, err = h.writeSecret(ctx, s, bad)
	require.ErrorContains(t, err, "declared schema")
	secrets, err := auth.GetKubeClient(ctx).CoreV1().Secrets("lab").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, secrets.Items)
	ref, err := h.writeSecret(ctx, s, req)
	require.NoError(t, err)
	again, err := h.writeSecret(ctx, s, req)
	require.NoError(t, err)
	require.Equal(t, ref, again)
	request := NodeInputRequest{UID: string(wf.UID), ResourceVersion: wf.ResourceVersion, NodeID: "node", RequestID: "input-request", Values: map[string]any{"credential": req.Value}}
	_, err = h.submitInput(ctx, s, wf.Name, request)
	require.ErrorContains(t, err, "schema validation")
	require.Zero(t, service.calls)
	request.Values = map[string]any{"credential": ref}
	_, err = h.submitInput(ctx, s, wf.Name, request)
	require.ErrorContains(t, err, "temporary outage")
	require.Zero(t, service.calls)
	done, err := h.submitInput(ctx, s, wf.Name, request)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls)
	_, err = h.submitInput(ctx, s, wf.Name, request)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls)
	recordObject, err := s.get(ctx, "parameternodeinputs", done.Annotations[p.Group+"/input-"+p.Digest("node")[:16]])
	require.NoError(t, err)
	var record parameterapi.ParameterNodeInputSpec
	require.NoError(t, decode(recordObject, &record))
	require.Len(t, record.SecretVersions, 1)
	serialized, _ := json.Marshal([]any{done, recordObject, pending})
	require.NotContains(t, string(serialized), req.Value)
	require.NoError(t, s.validateInputSecrets(ctx, done))
	name := ref.(map[string]any)["secretRef"].(map[string]any)["name"].(string)
	require.NoError(t, auth.GetKubeClient(ctx).CoreV1().Secrets("lab").Delete(ctx, name, metav1.DeleteOptions{}))
	require.ErrorContains(t, s.validateInputSecrets(ctx, done), "revoked")
	// Recovery of an already-applied request is a read; revocation cannot undo it.
	_, err = h.submitInput(ctx, s, wf.Name, request)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls)
	_, err = h.writeSecret(ctx, s, req)
	require.ErrorContains(t, err, "waiting native suspend")
}

func TestNativeRetryRevalidatesFrozenCredentialsButStatusUpdatesDoNot(t *testing.T) {
	ctx, s, _, _ := previewFixture(t)
	immutable := true
	_, err := auth.GetKubeClient(ctx).CoreV1().Secrets("lab").Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "original", UID: "original-uid"}, Immutable: &immutable}, metav1.CreateOptions{})
	require.NoError(t, err)
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "done", Namespace: "lab", UID: "workflow-uid", Annotations: map[string]string{p.Group + "/submission": "frozen"}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowFailed}}
	_, err = s.resource("workflowsubmissions").Create(ctx, object("WorkflowSubmission", "frozen", "lab", map[string]any{"workflow": wf, "definitions": parameterapi.ExecutionDefinitions{Versions: map[string]string{"secret/original": "original-uid:1"}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	next := wf.DeepCopy()
	next.Status.Phase = wfv1.WorkflowRunning
	require.NoError(t, s.ValidateNativeWorkflow(ctx, next, wf))
	require.NoError(t, auth.GetKubeClient(ctx).CoreV1().Secrets("lab").Delete(ctx, "original", metav1.DeleteOptions{}))
	require.ErrorContains(t, s.ValidateNativeWorkflow(ctx, next, wf), "revoked")
	unchanged := wf.DeepCopy()
	unchanged.Status.Message = "status readback"
	require.NoError(t, s.ValidateNativeWorkflow(ctx, unchanged, wf))
	// Recreating the same name cannot silently replace the pinned version.
	_, err = auth.GetKubeClient(ctx).CoreV1().Secrets("lab").Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "original", UID: "replacement-uid"}, Immutable: &immutable}, metav1.CreateOptions{})
	require.NoError(t, err)
	require.ErrorContains(t, s.ValidateNativeWorkflow(ctx, next, wf), "identity changed")
}
