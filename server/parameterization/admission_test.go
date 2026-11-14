package parameterization

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestAdmissionCreateUsesTheSameResolutionAsAPI(t *testing.T) {
	ctx, store, service, revision := previewFixture(t)
	preview, err := store.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	created, err := (Handler{Workflows: service}).submit(ctx, store, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "admission"})
	require.NoError(t, err)
	wf := created.(*wfv1.Workflow)
	require.NoError(t, store.ValidateNativeWorkflow(ctx, wf, nil))
	for _, edit := range []func(*wfv1.Workflow){
		func(w *wfv1.Workflow) {
			v := wfv1.AnyString("not-an-integer")
			w.Spec.Arguments.Parameters[0].Value = &v
		},
		func(w *wfv1.Workflow) { w.Spec.Templates[0].Container.Image = "different-image" },
		func(w *wfv1.Workflow) { w.Spec.ServiceAccountName = "different-account" },
		func(w *wfv1.Workflow) { w.Name = "copied-receipt" },
		func(w *wfv1.Workflow) { w.Annotations[p.Group+"/revision"] = "different-revision" },
	} {
		changed := wf.DeepCopy()
		edit(changed)
		require.Error(t, store.ValidateNativeWorkflow(ctx, changed, nil))
	}
	// Forging the frozen receipt still fails against the original model/binding.
	fake := wf.DeepCopy()
	fake.Annotations[p.Group+"/submission"] = "forged"
	v := wfv1.AnyString("wrong")
	fake.Spec.Arguments.Parameters[0].Value = &v
	_, err = store.resource("workflowsubmissions").Create(ctx, object("WorkflowSubmission", "forged", "lab", map[string]any{"workflow": fake, "binding": "standard", "revision": revision, "executionDigest": preview.ExecutionDigest}), metav1.CreateOptions{})
	require.NoError(t, err)
	require.ErrorContains(t, store.ValidateNativeWorkflow(ctx, fake, nil), "resolved parameter contract")
}

func TestAdmissionDoesNotRevalidateNativeControlsAndFinishedResults(t *testing.T) {
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "lab", UID: "uid", Annotations: map[string]string{p.Group + "/submission": "historical", p.Group + "/revision": "old"}}, Spec: wfv1.WorkflowSpec{Entrypoint: "run"}}
	// No store clients: status, suspend and termination must survive a missing
	// source definition or even a parameter API outage.
	store := Store{Namespace: "lab"}
	for _, edit := range []func(*wfv1.Workflow){
		func(w *wfv1.Workflow) { w.Status.Phase = wfv1.WorkflowSucceeded },
		func(w *wfv1.Workflow) { yes := true; w.Spec.Suspend = &yes },
		func(w *wfv1.Workflow) { w.Spec.Shutdown = wfv1.ShutdownStrategyTerminate },
		func(w *wfv1.Workflow) { w.Labels = map[string]string{"new": "label"} },
	} {
		next := wf.DeepCopy()
		edit(next)
		require.NoError(t, store.ValidateNativeWorkflow(context.Background(), next, wf))
	}
	next := wf.DeepCopy()
	next.Annotations = nil
	require.ErrorContains(t, store.ValidateNativeWorkflow(context.Background(), next, wf), "remove")
	next = wf.DeepCopy()
	next.Spec.Entrypoint = "changed"
	require.ErrorContains(t, store.ValidateNativeWorkflow(context.Background(), next, wf), "ParameterRetry")
	next = wf.DeepCopy()
	next.Annotations[p.Group+"/revision"] = "new"
	require.ErrorContains(t, store.ValidateNativeWorkflow(context.Background(), next, wf), "ParameterRetry")
	require.NoError(t, (Store{}).ValidateNativeWorkflow(context.Background(), &wfv1.Workflow{}, nil))
}

func TestAdmissionReviewPreservesUIDAndFailsClosedForAdoptedWorkflows(t *testing.T) {
	for _, adopted := range []bool{false, true} {
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "lab"}}
		if adopted {
			wf.Annotations = map[string]string{p.Group + "/revision": "revision"}
		}
		raw, _ := json.Marshal(wf)
		review := admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{UID: "request-uid", Namespace: "lab", Resource: metav1.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}, Operation: admissionv1.Create, UserInfo: authenticationv1.UserInfo{Username: "alice"}, Object: runtime.RawExtension{Raw: raw}}}
		body, _ := json.Marshal(review)
		w := httptest.NewRecorder()
		(AdmissionHandler{Namespace: "lab"}).ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/validate", bytes.NewReader(body)))
		var response admissionv1.AdmissionReview
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		require.Equal(t, review.Request.UID, response.Response.UID)
		require.Equal(t, !adopted, response.Response.Allowed)
	}
}

func TestAdmissionReadTransportChecksActualCallerWithoutImpersonation(t *testing.T) {
	client := kubefake.NewSimpleClientset()
	allowed := false
	client.PrependReactor("create", "subjectaccessreviews", func(action ktesting.Action) (bool, runtime.Object, error) {
		sar := action.(ktesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview)
		require.Equal(t, "alice", sar.Spec.User)
		require.Equal(t, []string{"team"}, sar.Spec.Groups)
		require.Equal(t, &authorizationv1.ResourceAttributes{Namespace: "lab", Group: p.Group, Version: p.Version, Resource: "parameterrevisions", Name: "revision", Verb: "get"}, sar.Spec.ResourceAttributes)
		return true, &authorizationv1.SubjectAccessReview{Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed}}, nil
	})
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Empty(t, r.Header.Get("Impersonate-User"))
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	transport := &callerReadTransport{base: http.DefaultTransport, authorization: client.AuthorizationV1(), actor: authenticationv1.UserInfo{Username: "alice", Groups: []string{"team"}}}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, backend.URL+"/apis/"+p.Group+"/"+p.Version+"/namespaces/lab/parameterrevisions/revision", nil)
	resp, err := transport.RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
	}
	require.Error(t, err)
	require.Zero(t, calls)
	allowed = true
	resp, err = transport.RoundTrip(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 1, calls)
	req.Method = http.MethodPost
	resp, err = transport.RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
	}
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestAdmissionResourceAttributes(t *testing.T) {
	for _, path := range []string{"/api/v1/namespaces/lab/secrets/key", "/apis/argoproj.io/v1alpha1/clusterworkflowtemplates/source", "/api/v1/namespaces/lab/configmaps"} {
		r, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://kubernetes"+path, nil)
		a, e := readAttributes(r)
		require.NoError(t, e)
		require.NotEmpty(t, a.Resource)
	}
	for _, path := range []string{"/api", "/apis/argoproj.io", "/api/v1/namespaces/lab/pods/test/exec", "/api/v1/namespaces/lab/configmaps?watch=true"} {
		r, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://kubernetes"+path, nil)
		_, e := readAttributes(r)
		require.Error(t, e)
	}
}

func TestAdmissionRequestContextSupportsNativePartialRetry(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/validate", nil)
	wf, next := partialFixture()
	plan, err := planPartialRetry(admissionContext(req.Context()), wf, next)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
}
