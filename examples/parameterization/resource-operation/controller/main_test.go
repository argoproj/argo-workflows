package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	op "github.com/argoproj/argo-workflows/v4/workflow/resourceoperation"
	core "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFrozenActionVersionCompatibility(t *testing.T) {
	ref := api.ResolvedReference{Kind: "ActionDefinition", Name: "action", UID: "identity", ResourceVersion: "7"}
	frozen := map[string]interface{}{"inputSchema": map[string]interface{}{"type": "object"}, "template": map[string]interface{}{"name": "call"}}
	digest := p.Digest(frozen)
	receipt := func(version string) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]interface{}{"spec": map[string]interface{}{"definitionVersions": map[string]interface{}{"action/action": version}, "definitions": map[string]interface{}{"actions": map[string]interface{}{"action": frozen}}}}}
	}
	legacy := receipt("identity:7:" + digest)
	require.NoError(t, validateFrozenAction(ref, legacy))
	latest := receipt("identity:spec:" + digest)
	require.NoError(t, validateFrozenAction(ref, latest))
	ref.ResourceVersion = "status-only-update"
	require.NoError(t, validateFrozenAction(ref, latest))
	require.Error(t, validateFrozenAction(ref, legacy))
	ref.Digest = "different-content"
	require.Error(t, validateFrozenAction(ref, latest))
	ref.Digest = digest
	require.NoError(t, validateFrozenAction(ref, latest))
	ref.UID = "replacement"
	require.Error(t, validateFrozenAction(ref, latest))
	ref.UID = "identity"
	delete(latest.Object["spec"].(map[string]interface{}), "definitions")
	require.Error(t, validateFrozenAction(ref, latest))
	latest = receipt("identity:spec:" + "not-a-digest")
	require.Error(t, validateFrozenAction(ref, latest))
	// Frozen typed Argo templates may serialize zero-value fields absent in the
	// source. Receipt identities refer to the immutable source, not that encoding.
	latest = receipt("identity:spec:" + digest)
	require.NoError(t, unstructured.SetNestedField(latest.Object, map[string]interface{}{}, "spec", "definitions", "actions", "action", "template", "metadata"))
	require.NoError(t, validateFrozenAction(ref, latest))
}

func adapterFixture(t *testing.T) (*api.ResourceOperation, *core.Pod, *unstructured.Unstructured, *unstructured.Unstructured) {
	t.Helper()
	o := &api.ResourceOperation{TypeMeta: metav1.TypeMeta{APIVersion: api.GroupName + "/v1alpha1", Kind: "ResourceOperation"}, ObjectMeta: metav1.ObjectMeta{Name: "operation", Namespace: "demo", Annotations: map[string]string{api.GroupName + "/invoker-pod": "invoker", api.GroupName + "/workflow": "workflow"}}, Spec: api.ResourceOperationSpec{Operation: api.OperationSnapshot{WorkflowUID: "workflow-uid", NodeID: "workflow.step", IdempotencyKey: "workflow-uid:workflow.step", Inputs: api.JSONValue{Raw: []byte(`{}`)}, Target: api.OperationTarget{ConnectionRef: "adapter", ResourceType: "report"}, ActionRef: api.ResolvedReference{Kind: "ActionDefinition", Name: "action", UID: "action-uid", ResourceVersion: "7"}}}}
	manifest, err := json.Marshal(o)
	require.NoError(t, err)
	template, err := json.Marshal(wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "create", Manifest: string(manifest)}})
	require.NoError(t, err)
	pod := &core.Pod{ObjectMeta: metav1.ObjectMeta{Name: "invoker", Namespace: "demo", OwnerReferences: []metav1.OwnerReference{{Kind: "Workflow", UID: types.UID("workflow-uid")}}, Annotations: map[string]string{"workflows.argoproj.io/node-name": "workflow.step"}}, Spec: core.PodSpec{ServiceAccountName: "workflow-executor", Containers: []core.Container{{Name: "main", Env: []core.EnvVar{{Name: "ARGO_TEMPLATE", Value: string(template)}}}}}, Status: core.PodStatus{Phase: core.PodRunning}}
	wf := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "argoproj.io/v1alpha1", "kind": "Workflow", "metadata": map[string]interface{}{"name": "workflow", "namespace": "demo", "uid": "workflow-uid", "annotations": map[string]interface{}{api.GroupName + "/submission": "receipt"}}, "spec": map[string]interface{}{}, "status": map[string]interface{}{"phase": "Running"}}}
	receipt := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": api.GroupName + "/v1alpha1", "kind": "WorkflowSubmission", "metadata": map[string]interface{}{"name": "receipt", "namespace": "demo"}, "spec": map[string]interface{}{"definitionVersions": map[string]interface{}{"action/action": "action-uid:7:digest"}}}}
	return o, pod, wf, receipt
}

func TestAdapterVerifiesInvocationBeforeDispatch(t *testing.T) {
	for _, tt := range []struct {
		name      string
		change    func(*api.ResourceOperation, *core.Pod, *unstructured.Unstructured, *unstructured.Unstructured)
		wantStop  bool
		wantError string
	}{
		{name: "active invocation"},
		{name: "namespace", change: func(o *api.ResourceOperation, _ *core.Pod, _, _ *unstructured.Unstructured) { o.Namespace = "other" }, wantError: "unsupported demo target"},
		{name: "connection", change: func(o *api.ResourceOperation, _ *core.Pod, _, _ *unstructured.Unstructured) {
			o.Spec.Operation.Target.ConnectionRef = "other"
		}, wantError: "unsupported demo target"},
		{name: "idempotency", change: func(o *api.ResourceOperation, _ *core.Pod, _, _ *unstructured.Unstructured) {
			o.Spec.Operation.IdempotencyKey = "replacement"
		}, wantError: "stable invocation"},
		{name: "pod absent", change: func(o *api.ResourceOperation, _ *core.Pod, _, _ *unstructured.Unstructured) {
			o.Annotations[api.GroupName+"/invoker-pod"] = "missing"
		}, wantError: "pod not available"},
		{name: "pod owner", change: func(_ *api.ResourceOperation, p *core.Pod, _, _ *unstructured.Unstructured) { p.OwnerReferences = nil }, wantError: "active native task"},
		{name: "pod phase", change: func(_ *api.ResourceOperation, p *core.Pod, _, _ *unstructured.Unstructured) {
			p.Status.Phase = core.PodSucceeded
		}, wantError: "active native task"},
		{name: "pod service account", change: func(_ *api.ResourceOperation, p *core.Pod, _, _ *unstructured.Unstructured) {
			p.Spec.ServiceAccountName = "other"
		}, wantError: "active native task"},
		{name: "pod node identity", change: func(_ *api.ResourceOperation, p *core.Pod, _, _ *unstructured.Unstructured) {
			p.Annotations["workflows.argoproj.io/node-name"] = "other"
		}, wantError: "active native task"},
		{name: "invalid materialized template", change: func(_ *api.ResourceOperation, p *core.Pod, _, _ *unstructured.Unstructured) {
			p.Spec.Containers[0].Env[0].Value = "{"
		}, wantError: "resolved native resource manifest"},
		{name: "input replaced", change: func(o *api.ResourceOperation, _ *core.Pod, _, _ *unstructured.Unstructured) {
			o.Spec.Operation.Inputs.Raw = []byte(`{"changed":true}`)
		}, wantError: "resolved native resource manifest"},
		{name: "workflow absent", change: func(o *api.ResourceOperation, _ *core.Pod, _, _ *unstructured.Unstructured) {
			o.Annotations[api.GroupName+"/workflow"] = "missing"
		}, wantError: "workflow identity unavailable"},
		{name: "workflow replaced", change: func(_ *api.ResourceOperation, _ *core.Pod, w, _ *unstructured.Unstructured) { w.SetUID("replacement") }, wantError: "workflow identity replaced"},
		{name: "receipt absent", change: func(_ *api.ResourceOperation, _ *core.Pod, w, _ *unstructured.Unstructured) { w.SetAnnotations(nil) }, wantError: "frozen submission unavailable"},
		{name: "action replaced", change: func(_ *api.ResourceOperation, _ *core.Pod, _, r *unstructured.Unstructured) {
			r.Object["spec"] = map[string]interface{}{}
		}, wantError: "frozen submission"},
		{name: "suspension is not cancellation", change: func(_ *api.ResourceOperation, _ *core.Pod, w, _ *unstructured.Unstructured) {
			w.Object["spec"] = map[string]interface{}{"suspend": true}
		}},
		{name: "shutdown", change: func(_ *api.ResourceOperation, _ *core.Pod, w, _ *unstructured.Unstructured) {
			w.Object["spec"] = map[string]interface{}{"shutdown": "Stop"}
		}, wantStop: true},
		{name: "workflow failed", change: func(_ *api.ResourceOperation, _ *core.Pod, w, _ *unstructured.Unstructured) {
			w.Object["status"] = map[string]interface{}{"phase": "Failed"}
		}, wantStop: true},
		{name: "workflow deleted", change: func(_ *api.ResourceOperation, _ *core.Pod, w, _ *unstructured.Unstructured) {
			now := metav1.NewTime(time.Now())
			w.SetDeletionTimestamp(&now)
		}, wantStop: true},
		{name: "claimed operation survives pod and receipt cleanup", change: func(o *api.ResourceOperation, _ *core.Pod, w, _ *unstructured.Unstructured) {
			o.Status.RequestDigest = "claimed"
			o.Annotations[api.GroupName+"/invoker-pod"] = "missing"
			w.SetAnnotations(nil)
		}},
		{name: "claimed operation notices workflow cleanup", change: func(o *api.ResourceOperation, _ *core.Pod, _, _ *unstructured.Unstructured) {
			o.Status.RequestDigest = "claimed"
			o.Annotations[api.GroupName+"/workflow"] = "missing"
		}, wantStop: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			o, pod, w, receipt := adapterFixture(t)
			if tt.change != nil {
				tt.change(o, pod, w, receipt)
			}
			a := &adapter{ns: "demo", instance: "adapter", d: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), w, receipt), k: kubefake.NewClientset(pod)}
			stop, err := a.Verify(t.Context(), o)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantStop, stop)
		})
	}
}

func TestAdapterOutputContract(t *testing.T) {
	schema := `{"type":"object","required":["state"],"properties":{"state":{"type":"string"}},"additionalProperties":false}`
	for _, tt := range []struct {
		name, schema, value string
		ok                  bool
	}{
		{"valid", schema, `{"state":"ready"}`, true},
		{"wrong type", schema, `{"state":false}`, false},
		{"missing field", schema, `{}`, false},
		{"extra field", schema, `{"state":"ready","unreviewed":true}`, false},
		{"malformed output", schema, `{`, false},
		{"malformed schema", `{`, `{}`, false},
		{"remote schema", `{"$ref":"https://invalid.example/schema"}`, `{}`, false},
		{"missing schema", "", `{}`, false},
		{"missing output", schema, "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			o := &api.ResourceOperation{}
			if tt.schema != "" {
				o.Spec.Operation.OutputSchema = &api.JSONValue{Raw: []byte(tt.schema)}
			}
			var value *api.JSONValue
			if tt.value != "" {
				value = &api.JSONValue{Raw: []byte(tt.value)}
			}
			err := (&adapter{}).ValidateOutput(t.Context(), o, value)
			if tt.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAdapterHTTPProtocol(t *testing.T) {
	snapshot := api.OperationSnapshot{IdempotencyKey: "workflow:step/a", WorkflowUID: "workflow", NodeID: "step/a", Inputs: api.JSONValue{Raw: []byte(`{"target":"demo"}`)}}
	for _, tt := range []struct {
		name, method    string
		code            int
		body, wantError string
	}{
		{"lookup absent", http.MethodGet, 404, "", ""},
		{"start", http.MethodPost, 200, `{"Found":true,"Phase":"Running","ExternalID":"external"}`, ""},
		{"cancel", http.MethodDelete, 200, `{}`, ""},
		{"lookup unavailable", http.MethodGet, 503, "private provider detail", "provider HTTP 503"},
		{"start rejected", http.MethodPost, 404, "", "provider HTTP 404"},
		{"invalid receipt", http.MethodGet, 200, `{`, "invalid provider receipt"},
		{"bounded receipt", http.MethodGet, 200, `{"Phase":"` + strings.Repeat("x", 65536) + `"}`, "invalid provider receipt"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, tt.method, r.Method)
				require.Equal(t, "/operations/workflow:step%2Fa", r.URL.EscapedPath())
				if r.Method == http.MethodPost {
					var got api.OperationSnapshot
					require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
					require.Equal(t, snapshot, got)
				}
				w.WriteHeader(tt.code)
				_, err := w.Write([]byte(tt.body))
				require.NoError(t, err)
			}))
			defer server.Close()
			a := &adapter{endpoint: server.URL, client: server.Client()}
			var result op.Observation
			var err error
			switch tt.method {
			case http.MethodGet:
				result, err = a.Lookup(t.Context(), snapshot)
			case http.MethodPost:
				result, err = a.Start(t.Context(), snapshot)
			case http.MethodDelete:
				err = a.Cancel(t.Context(), snapshot, "external")
			}
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
				if tt.method == http.MethodPost {
					require.True(t, result.Found)
					require.Equal(t, "external", result.ExternalID)
				}
			}
		})
	}
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	a := &adapter{endpoint: server.URL, client: server.Client()}
	_, err := a.Lookup(context.Background(), snapshot)
	require.EqualError(t, err, "provider transport unavailable")
	require.True(t, a.CanCancel())
}

func TestAdapterStatusConflictPreservesVersion(t *testing.T) {
	o, _, _, _ := adapterFixture(t)
	o.ResourceVersion = "7"
	o.Status.Phase = "Running"
	for _, conflict := range []bool{false, true} {
		t.Run(map[bool]string{false: "saved", true: "conflict"}[conflict], func(t *testing.T) {
			d := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
			d.PrependReactor("update", "resourceoperations", func(action ktesting.Action) (bool, runtime.Object, error) {
				require.Equal(t, "status", action.GetSubresource())
				got := action.(ktesting.UpdateAction).GetObject().(*unstructured.Unstructured)
				require.Equal(t, "7", got.GetResourceVersion())
				require.Equal(t, "Running", got.Object["status"].(map[string]interface{})["phase"])
				if conflict {
					return true, nil, kerrors.NewConflict(schema.GroupResource{Group: api.GroupName, Resource: "resourceoperations"}, o.Name, context.Canceled)
				}
				got.SetResourceVersion("8")
				return true, got, nil
			})
			copy := o.DeepCopy()
			err := (&adapter{ns: "demo", d: d}).SaveStatus(t.Context(), copy)
			if conflict {
				require.True(t, kerrors.IsConflict(err))
				require.Equal(t, "7", copy.ResourceVersion)
			} else {
				require.NoError(t, err)
				require.Equal(t, "8", copy.ResourceVersion)
			}
		})
	}
}
