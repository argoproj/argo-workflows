package parameterization

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestFrozenInputOptionsIdentityFreshValidationAndResponseRecovery(t *testing.T) {
	ctx, s, w, _ := previewFixture(t)
	service := &inputService{workflowService: w}
	h := Handler{Workflows: &interruptedInputService{inputService: service}}
	contract := FrozenInputForm{Model: p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"environment": map[string]any{"type": "string"}, "region": map[string]any{"type": "string"}}, "required": []any{"environment", "region"}, "additionalProperties": false}}, Mappings: []p.Mapping{{Path: "/region", Parameter: "region", Encoding: "string"}}, Form: map[string]any{"options": map[string]any{"/region": map[string]any{"kind": "configmap", "name": "choices", "key": "regions", "dependsOn": []string{"/environment"}}}}}
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "options-wf", Namespace: "lab", UID: "uid", ResourceVersion: "7", Annotations: map[string]string{p.Group + "/submission": "options-receipt"}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, Nodes: wfv1.Nodes{"choose": {ID: "choose", Name: "options-wf.choose", TemplateName: "choose", Type: wfv1.NodeTypeSuspend, Phase: wfv1.NodeRunning, Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "region", ValueFrom: &wfv1.ValueFrom{Supplied: &wfv1.SuppliedValueFrom{}}}}}}}}}
	_, err := w.client.ArgoprojV1alpha1().Workflows("lab").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowsubmissions").Create(ctx, object("WorkflowSubmission", "options-receipt", "lab", map[string]any{"workflow": wf, "inputForms": map[string]FrozenInputForm{"choose": contract}}), metav1.CreateOptions{})
	require.NoError(t, err)
	kube := auth.GetKubeClient(ctx)
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "choices", Namespace: "lab", UID: "choices-uid", ResourceVersion: "1"}, Data: map[string]string{"regions": `[{"value":"east","label":"East","when":{"/environment":"test"}},{"value":"west","label":"West","when":{"/environment":"production"}}]`}}
	_, err = kube.CoreV1().ConfigMaps("lab").Create(ctx, cm, metav1.CreateOptions{})
	require.NoError(t, err)
	pending, err := h.pendingInputs(ctx, s, wf.Name)
	require.NoError(t, err)
	input := InputContext{Workflow: wf.Name, UID: "uid", NodeID: "choose", ContractDigest: pending.Items[0].ContractDigest}
	request := OptionsRequest{Input: &input, Path: "/region", Values: map[string]any{"environment": "test"}}
	options, err := h.options(ctx, s, request)
	require.NoError(t, err)
	require.Len(t, options.Items, 1)
	require.Equal(t, "east", options.Items[0].Value)
	for _, change := range []func(*InputContext){func(x *InputContext) { x.UID = "wrong" }, func(x *InputContext) { x.NodeID = "wrong" }, func(x *InputContext) { x.ContractDigest = "wrong" }} {
		bad := input
		change(&bad)
		r := request
		r.Input = &bad
		_, err = h.options(ctx, s, r)
		require.Error(t, err)
	}
	bad := request
	bad.Binding = "standard"
	_, err = h.options(ctx, s, bad)
	require.ErrorContains(t, err, "mutually exclusive")
	_, err = s.Options(ctx, request, nil)
	require.ErrorContains(t, err, "workflow authorization")
	// No mutable WorkflowForm/ParameterTemplate exists for this contract.
	req := NodeInputRequest{UID: "uid", ResourceVersion: "7", NodeID: "choose", RequestID: "first", Values: map[string]any{"environment": "production", "region": "east"}}
	_, err = h.submitInput(ctx, s, wf.Name, req)
	require.ErrorContains(t, err, "selected option is unavailable")
	require.Zero(t, service.calls)
	req.Values = map[string]any{"environment": "production", "region": "west"}
	_, err = h.submitInput(ctx, s, wf.Name, req)
	require.ErrorContains(t, err, "temporary outage")
	require.Zero(t, service.calls)
	// The first dispatch did not happen. Changed source evidence requires a new request.
	cm.ResourceVersion = "2"
	_, err = kube.CoreV1().ConfigMaps("lab").Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = h.submitInput(ctx, s, wf.Name, req)
	require.True(t, apierrors.IsConflict(err))
	require.Zero(t, service.calls)
	req.RequestID = "refreshed"
	done, err := h.submitInput(ctx, s, wf.Name, req)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls)
	recordObject, err := s.get(ctx, "parameternodeinputs", done.Annotations[p.Group+"/input-"+p.Digest("choose")[:16]])
	require.NoError(t, err)
	var record parameterapi.ParameterNodeInputSpec
	require.NoError(t, decode(recordObject, &record))
	require.Contains(t, record.OptionVersions["/region"], "choices-uid:2:")
	require.NoError(t, kube.CoreV1().ConfigMaps("lab").Delete(ctx, "choices", metav1.DeleteOptions{}))
	_, err = h.submitInput(ctx, s, wf.Name, req)
	require.NoError(t, err)
	require.Equal(t, 1, service.calls, "recovery of committed input does not query provider or resume twice")
	_, err = h.options(ctx, s, request)
	require.ErrorContains(t, err, "waiting native suspend")
	pending, err = h.pendingInputs(ctx, s, wf.Name)
	require.NoError(t, err)
	require.Len(t, pending.Submitted, 1)
}

func TestInputOptionProvidersKeepCallerAuthorization(t *testing.T) {
	ctx, s, _, _ := previewFixture(t)
	client := auth.GetKubeClient(ctx).(*kubefake.Clientset)
	client.PrependReactor("get", "configmaps", func(a ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "catalog", nil)
	})
	form := parameterapi.WorkflowFormSpec{Options: map[string]OptionSource{"/region": {Kind: "configmap", Name: "catalog", Key: "choices"}}}
	_, err := s.optionsForForm(ctx, form, "/region", map[string]any{}, nil)
	require.True(t, apierrors.IsForbidden(err))
	// Even a manual option source must not silently conceal an authorization failure.
	source := form.Options["/region"]
	source.AllowManual = true
	form.Options["/region"] = source
	raw, _ := json.Marshal(form)
	var data map[string]any
	require.NoError(t, json.Unmarshal(raw, &data))
	_, err = s.validateInputOptions(ctx, &FrozenInputForm{Form: data}, map[string]any{"region": "manual"})
	require.True(t, apierrors.IsForbidden(err))
}
