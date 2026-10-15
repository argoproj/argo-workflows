package parameterization

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfClient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	wfFake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type workflowService struct {
	workflowapi.WorkflowServiceServer
	client       wfClient.Interface
	createCalls  int
	loseResponse bool
}

func (s *workflowService) CreateWorkflow(ctx context.Context, req *workflowapi.WorkflowCreateRequest) (*wfv1.Workflow, error) {
	s.createCalls++
	wf, err := s.client.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, req.Workflow, metav1.CreateOptions{})
	if err == nil && s.loseResponse {
		s.loseResponse = false
		return nil, errors.New("response lost after server committed")
	}
	return wf, err
}
func previewFixture(t *testing.T) (context.Context, Store, *workflowService, string) {
	t.Helper()
	s := testStore()
	ctx := logging.TestContext(t.Context())
	binding := object("WorkflowParameterBinding", "standard", "lab", Binding{WorkflowTemplate: "native", ParameterTemplate: "typed", Mappings: []p.Mapping{{Path: "/count", Parameter: "count", Encoding: "decimal"}}})
	_, err := s.resource("workflowparameterbindings").Create(ctx, binding, metav1.CreateOptions{})
	require.NoError(t, err)
	native := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "native", Namespace: "lab", ResourceVersion: "1"}, Spec: wfv1.WorkflowSpec{Entrypoint: "main", Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "count"}}}, Templates: []wfv1.Template{{Name: "main", Container: &corev1.Container{Image: "alpine:3.22", Command: []string{"echo"}, Args: []string{"{{workflow.parameters.count}}"}}}}}}
	// This generated client does not have apply configurations or NewClientset.
	wc := wfFake.NewSimpleClientset(native)
	ctx = context.WithValue(ctx, auth.WfKey, wc)
	ctx = context.WithValue(ctx, auth.KubeKey, kubefake.NewSimpleClientset())
	draft, err := s.Save(ctx, "sample", SaveRequest{Template: "typed", Values: map[string]any{"count": 2, "enabled": false}})
	require.NoError(t, err)
	revision, _, _ := unstructured.NestedString(draft.Object, "spec", "revision")
	return ctx, s, &workflowService{client: wc}, revision
}
func TestSubmitFreezesRevisionAndRecoversLostResponse(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w}
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Equal(t, "2", preview.Workflow.Spec.Arguments.Parameters[0].Value.String())
	request := SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "one-intent"}
	w.loseResponse = true
	_, err = h.submit(ctx, s, request)
	require.ErrorContains(t, err, "response lost")
	_, err = h.submit(ctx, s, request)
	require.NoError(t, err)
	require.Equal(t, 1, w.createCalls, "a lost response must not create a second Workflow")
	workflows, err := w.client.ArgoprojV1alpha1().Workflows("lab").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, workflows.Items, 1)
	request.Revision = "another-revision"
	_, err = h.submit(ctx, s, request)
	require.True(t, apierrors.IsAlreadyExists(err), "same key with different content conflicts")
}
func TestStalePreviewAndExplicitActionBinding(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w}
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	action := ActionDefinition{InputSchema: map[string]any{"type": "object"}, OutputSchema: map[string]any{"type": "object"}, Template: &wfv1.Template{Name: "implementation", Container: &corev1.Container{Image: "alpine:3.22", Command: []string{"echo"}, Args: []string{"bound action"}}}}
	_, err = s.resource("actiondefinitions").Create(ctx, object("ActionDefinition", "echo-v1", "lab", action), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowactionbindings").Create(ctx, object("WorkflowActionBinding", "calls", "lab", ActionBinding{WorkflowTemplate: "native", Calls: []ActionCall{{Template: "main", ActionDefinition: "echo-v1"}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	binding, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(binding.Object, "calls", "spec", "actionBinding"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, binding, metav1.UpdateOptions{})
	require.NoError(t, err)
	current, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.NotEqual(t, preview.ExecutionDigest, current.ExecutionDigest)
	require.Equal(t, []string{"bound action"}, current.Workflow.Spec.Templates[0].Container.Args)
	_, err = h.submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "stale"})
	require.ErrorContains(t, err, "stale")
	require.Zero(t, w.createCalls)
}
func TestSecretReferenceNeverBecomesAnArgument(t *testing.T) {
	ctx, s, _, revision := previewFixture(t)
	ro, err := s.get(ctx, "parameterrevisions", revision)
	require.NoError(t, err)
	var rev Revision
	require.NoError(t, decode(ro, &rev))
	rev.Model.SensitivePaths = []string{"/credential"}
	rev.Values.(map[string]any)["credential"] = map[string]any{"secretRef": map[string]any{"name": "credential-v1", "key": "token"}}
	// Fixture models an already validated immutable snapshot, without putting plaintext in it.
	_, err = s.resource("parameterrevisions").Create(ctx, object("ParameterRevision", "secret-revision", "lab", rev), metav1.CreateOptions{})
	require.NoError(t, err)
	binding, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedSlice(binding.Object, []any{map[string]any{"path": "/credential", "template": "main", "environment": "TOKEN"}}, "spec", "secrets"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, binding, metav1.UpdateOptions{})
	require.NoError(t, err)
	immutable := true
	_, err = auth.GetKubeClient(ctx).CoreV1().Secrets("lab").Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credential-v1", Namespace: "lab"}, Immutable: &immutable, Data: map[string][]byte{"token": []byte("do-not-serialize")}}, metav1.CreateOptions{})
	require.NoError(t, err)
	preview, err := s.Preview(ctx, "standard", "secret-revision")
	require.NoError(t, err)
	raw, err := json.Marshal(preview)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "do-not-serialize")
	require.Equal(t, "credential-v1", preview.Workflow.Spec.Templates[0].Container.Env[0].ValueFrom.SecretKeyRef.Name)
}

type testGatekeeper struct {
	auth.Gatekeeper
	client dynamic.Interface
	denied bool
}

func (g *testGatekeeper) ContextWithRequest(ctx context.Context, req any) (context.Context, error) {
	if g.denied {
		return nil, errors.New("denied")
	}
	return context.WithValue(ctx, auth.DynamicKey, g.client), nil
}
func TestHTTPLimitsScopeAndKubernetesForbidden(t *testing.T) {
	fake := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	h := Handler{Gatekeeper: &testGatekeeper{client: fake}, Namespace: "lab"}
	for _, path := range []string{Prefix + "other/sets/test", Prefix + "bad_name/sets/test"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		require.True(t, w.Code == 400 || w.Code == 403)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPut, Prefix+"lab/sets/test", strings.NewReader(`{"unknown":true}`)))
	require.Equal(t, 400, w.Code)
}

func TestRetryReceiptKeepsOriginalWorkflowAndSurvivesLostResponse(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "failed", Namespace: "lab", UID: "unique", ResourceVersion: "17", Annotations: map[string]string{p.Group + "/revision": revision}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowFailed}}
	_, err := w.client.ArgoprojV1alpha1().Workflows("lab").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	native := &retryService{WorkflowServiceServer: w, client: w.client, loseResponse: true}
	h := Handler{Workflows: native}
	req := RetryRequest{Workflow: "failed", UID: "unique", ResourceVersion: "17", IdempotencyKey: "attempt"}
	_, err = h.retry(ctx, s, req)
	require.ErrorContains(t, err, "response lost")
	result, err := h.retry(ctx, s, req)
	require.NoError(t, err)
	require.Equal(t, 1, native.calls)
	require.Equal(t, wf.UID, result.UID)
	require.Equal(t, revision, result.Annotations[p.Group+"/revision"])
	req.UID = "replaced"
	_, err = h.retry(ctx, s, req)
	require.ErrorContains(t, err, "identity changed")
}

type retryService struct {
	workflowapi.WorkflowServiceServer
	client       wfClient.Interface
	calls        int
	loseResponse bool
}

func (s *retryService) RetryWorkflow(ctx context.Context, req *workflowapi.WorkflowRetryRequest) (*wfv1.Workflow, error) {
	s.calls++
	proof, _ := p.RetryProof(ctx)
	wf, err := s.client.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	wf.Annotations[p.Group+"/retry"] = proof.Receipt
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.ResourceVersion = "18"
	wf, err = s.client.ArgoprojV1alpha1().Workflows(req.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
	if s.loseResponse {
		s.loseResponse = false
		return nil, errors.New("response lost")
	}
	return wf, err
}
