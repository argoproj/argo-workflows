package parameterization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

const Prefix = "/api/v1/parameterization/"

type Handler struct {
	Gatekeeper               auth.Gatekeeper
	Workflows                workflowapi.WorkflowServiceServer
	Namespace                string
	OptionProviders          map[string]OptionProvider
	SecretWriter             SecretWriter
	AllowLoopbackSecretInput bool
	HistoryPager             *HistoryPager
}
type namespaceRequest struct{ Namespace string }

func (n namespaceRequest) GetNamespace() string { return n.Namespace }

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix), "/"), "/")
	if len(parts) < 2 || len(parts) > 3 || len(validation.IsDNS1123Label(parts[0])) > 0 {
		h.fail(w, 400, "InvalidPath", "namespace and operation required")
		return
	}
	ns, operation := parts[0], parts[1]
	if h.Namespace != "" && h.Namespace != ns {
		h.fail(w, 403, "NamespaceDenied", "namespace is outside this server's scope")
		return
	}
	md := metadata.MD{}
	if authorization := r.Header.Get("Authorization"); authorization != "" {
		md.Append("authorization", authorization)
	}
	for _, cookie := range r.Header.Values("Cookie") {
		md.Append("cookie", cookie)
	}
	ctx, err := h.Gatekeeper.ContextWithRequest(metadata.NewIncomingContext(r.Context(), md), namespaceRequest{ns})
	if err != nil {
		h.fail(w, 403, "Forbidden", "authentication or namespace authorization failed")
		return
	}
	actor := "server-service-account"
	if claims := auth.GetClaims(ctx); claims != nil {
		actor = claims.Subject
		if claims.ServiceAccountName != "" {
			actor = "system:serviceaccount:" + claims.ServiceAccountNamespace + ":" + claims.ServiceAccountName
		}
	}
	store := Store{Client: auth.GetDynamicClient(ctx), Namespace: ns, Actor: actor, OptionProviders: h.OptionProviders, HistoryPager: h.HistoryPager}
	if client, ok := ctx.Value(auth.KubeKey).(kubernetes.Interface); ok && client != nil {
		store.MetadataClient = client.CoreV1().RESTClient()
	}
	name := ""
	if len(parts) > 2 {
		name = parts[2]
		if len(validation.IsDNS1123Label(name)) > 0 {
			h.fail(w, 400, "InvalidName", "invalid resource name")
			return
		}
	}
	var result any
	switch {
	case operation == "node-results" && name != "" && r.Method == http.MethodGet:
		result, err = h.nodeResults(ctx, store, name, r.URL.Query().Get("uid"), r.URL.Query().Get("nodeId"), r.URL.Query().Get("history"))
	case operation == "definition-publications" && name != "" && r.Method == http.MethodPost:
		var req DefinitionPublishRequest
		if err = read(r, &req); err == nil {
			result, err = store.publishDefinition(ctx, name, req)
		}
	case operation == "definition-publications" && name != "" && r.Method == http.MethodGet:
		result, err = store.definitionVersions(ctx, name, r.URL.Query().Get("family"), r.URL.Query().Get("continue"))
	case operation == "definition-diff" && name != "" && r.Method == http.MethodPost:
		var req DiffRequest
		if err = read(r, &req); err == nil {
			result, err = store.definitionDiff(ctx, name, req)
		}
	case operation == "definitions" && name != "" && r.Method == http.MethodPut:
		var req DefinitionWrite
		if err = read(r, &req); err == nil {
			result, err = store.writeDefinition(ctx, name, r.URL.Query().Get("name"), req)
		}
	case operation == "catalog" && r.Method == http.MethodGet:
		if r.URL.Query().Get("view") == "metadata" && r.URL.Query().Get("name") == "" {
			result, err = store.catalogMetadata(ctx, name, r.URL.Query().Get("continue"))
		} else {
			result, err = store.catalog(ctx, name, r.URL.Query().Get("name"), r.URL.Query().Get("continue"))
		}
	case operation == "capabilities" && r.Method == http.MethodGet:
		result = capabilities(h.SecretWriter != nil)
	case operation == "action-reviews" && name != "" && r.Method == http.MethodGet:
		result, err = store.actionReview(ctx, name)
	case operation == "bindings" && r.Method == http.MethodGet:
		if name != "" {
			result, err = store.get(ctx, "workflowparameterbindings", name)
		} else {
			result, err = store.resource("workflowparameterbindings").List(ctx, metav1.ListOptions{Limit: 100, Continue: r.URL.Query().Get("continue")})
		}
	case operation == "models" && r.Method == http.MethodGet && name != "":
		result, err = store.get(ctx, "parametertemplates", name)
	case operation == "forms" && r.Method == http.MethodGet && name != "":
		result, err = store.get(ctx, "workflowforms", name)
	case operation == "sets" && name != "" && r.Method == http.MethodGet:
		result, err = store.get(ctx, "parametersets", name)
	case operation == "sets" && name != "" && r.Method == http.MethodPut:
		var req SaveRequest
		if err = read(r, &req); err == nil {
			result, err = store.Save(ctx, name, req)
		}
	case operation == "history" && name != "" && r.Method == http.MethodGet:
		var page *unstructured.UnstructuredList
		page, err = store.History(ctx, name, 25, r.URL.Query().Get("continue"))
		if err == nil && r.URL.Query().Get("view") == "summary" {
			result = historySummaries(page)
		} else {
			result = page
		}
	case operation == "revisions" && name != "" && r.Method == http.MethodGet:
		result, err = store.get(ctx, "parameterrevisions", name)
	case operation == "diff" && r.Method == http.MethodPost:
		var req DiffRequest
		if err = read(r, &req); err == nil {
			result, err = store.compare(ctx, req.Left, req.Right)
			if err == nil {
				result, err = diffPage(result.(map[string]any), req)
			}
		}
	case operation == "execution-diff" && r.Method == http.MethodPost:
		var req ExecutionDiffRequest
		if err = read(r, &req); err == nil {
			var full *ExecutionComparison
			full, err = h.executionDiff(ctx, store, req)
			if err == nil {
				result, err = executionDiffPage(full, req)
			}
		}
	case operation == "preview" && r.Method == http.MethodPost:
		var req struct {
			Binding  string `json:"binding"`
			Revision string `json:"revision"`
		}
		if err = read(r, &req); err == nil {
			result, err = store.Preview(ctx, req.Binding, req.Revision)
		}
	case operation == "options" && r.Method == http.MethodPost:
		var req OptionsRequest
		if err = read(r, &req); err == nil {
			result, err = h.options(ctx, store, req)
		}
	case operation == "retry-preview" && r.Method == http.MethodPost:
		var req PartialRetryRequest
		if err = read(r, &req); err == nil {
			result, _, err = h.partialRetryPreview(ctx, store, req)
		}
	case operation == "retry" && r.Method == http.MethodPost:
		var req RetryRequest
		if err = read(r, &req); err == nil {
			result, err = h.retry(ctx, store, req)
		}
	case operation == "secrets" && r.Method == http.MethodPost:
		if !h.secretTransportAllowed(r) {
			h.fail(w, 400, "SecureTransportRequired", "secret input requires HTTPS")
			return
		}
		var req SecretWriteRequest
		if err = read(r, &req); err == nil {
			result, err = h.writeSecret(ctx, store, req)
		}
	case operation == "inputs" && name != "" && r.Method == http.MethodGet:
		result, err = h.pendingInputs(ctx, store, name)
	case operation == "edit-context" && name != "" && r.Method == http.MethodGet:
		result, err = h.editContext(ctx, store, SourceExecution{Workflow: name, UID: r.URL.Query().Get("uid")})
	case operation == "inputs" && name != "" && r.Method == http.MethodPost:
		var req NodeInputRequest
		if err = read(r, &req); err == nil {
			result, err = h.submitInput(ctx, store, name, req)
		}
	case operation == "submit" && r.Method == http.MethodPost:
		var req SubmitRequest
		if err = read(r, &req); err == nil {
			result, err = h.submit(ctx, store, req)
		}
	default:
		h.fail(w, 404, "NotFound", "unsupported endpoint or method")
		return
	}
	if err != nil {
		status := 400
		code := "InvalidRequest"
		if apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) || grpcstatus.Code(err) == codes.Aborted {
			status = 409
			code = "Conflict"
		}
		if apierrors.IsForbidden(err) || grpcstatus.Code(err) == codes.PermissionDenied {
			status = 403
			code = "Forbidden"
		}
		if apierrors.IsNotFound(err) {
			status = 404
			code = "NotFound"
		}
		if apierrors.IsResourceExpired(err) {
			status = 410
			code = "HistoryExpired"
		}
		h.fail(w, status, code, err.Error())
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}
func (h Handler) fail(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}
func read(r *http.Request, target any) error {
	b, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024+1))
	if err != nil || len(b) > 1024*1024 {
		return fmt.Errorf("request exceeds 1 MiB or cannot be read")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON request")
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request must contain one JSON object")
	}
	return nil
}
func (s Store) compare(ctx context.Context, left, right string) (any, error) {
	a, err := s.get(ctx, "parameterrevisions", left)
	if err != nil {
		return nil, err
	}
	b, err := s.get(ctx, "parameterrevisions", right)
	if err != nil {
		return nil, err
	}
	var av, bv Revision
	if err = decode(a, &av); err != nil {
		return nil, err
	}
	if err = decode(b, &bv); err != nil {
		return nil, err
	}
	// Save rejects plaintext, but revisions can also be imported through Kubernetes.
	// Apply the union of both versions' declarations before serializing any diff.
	sensitive := append(append([]string{}, av.Model.SensitivePaths...), bv.Model.SensitivePaths...)
	for _, revision := range []Revision{av, bv} {
		validationErrors := p.ValidateSensitiveReferences(sensitive, revision.Values)
		for _, validationError := range validationErrors {
			if validationError.Code == "SecretReferenceRequired" {
				return nil, fmt.Errorf("sensitive revision value must use a Secret reference before comparison")
			}
		}
	}
	rules := bv.Model.IdentityRules
	identityChanged := p.Digest(av.Model.IdentityRules) != p.Digest(bv.Model.IdentityRules)
	if identityChanged {
		rules = nil
	}
	return map[string]any{"left": left, "right": right,
		"changes":              p.Diff(av.Values, bv.Values, rules),
		"modelChanged":         p.Digest(av.Model) != p.Digest(bv.Model),
		"modelChanges":         p.Diff(p.Clone(av.Model), p.Clone(bv.Model), nil),
		"sourceChanges":        p.Diff(p.Clone(av.Sources), p.Clone(bv.Sources), nil),
		"identityRulesChanged": identityChanged, "sensitivePaths": sensitive}, nil
}

type SubmitRequest struct {
	Binding         string           `json:"binding"`
	Revision        string           `json:"revision"`
	ExecutionDigest string           `json:"executionDigest"`
	IdempotencyKey  string           `json:"idempotencyKey"`
	SourceExecution *SourceExecution `json:"sourceExecution,omitempty"`
}

func (h Handler) submit(ctx context.Context, s Store, req SubmitRequest) (any, error) {
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 128 {
		return nil, fmt.Errorf("idempotencyKey must have 1-128 characters")
	}
	name := "submission-" + p.Digest(req.IdempotencyKey)[:32]
	receipt, err := s.get(ctx, "workflowsubmissions", name)
	var wf *wfv1.Workflow
	if err == nil {
		digest, _, _ := unstructured.NestedString(receipt.Object, "spec", "requestDigest")
		if digest != p.Digest(req) {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: p.Group, Resource: "workflowsubmissions"}, name)
		}
		raw, _, _ := unstructured.NestedMap(receipt.Object, "spec", "workflow")
		b, _ := json.Marshal(raw)
		wf = &wfv1.Workflow{}
		if err = json.Unmarshal(b, wf); err != nil {
			return nil, err
		}
	} else {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		if req.SourceExecution != nil {
			origin, sourceErr := h.editContext(ctx, s, *req.SourceExecution)
			if sourceErr != nil {
				return nil, sourceErr
			}
			next, sourceErr := s.get(ctx, "parameterrevisions", req.Revision)
			if sourceErr != nil {
				return nil, sourceErr
			}
			var revision Revision
			if sourceErr = decode(next, &revision); sourceErr != nil {
				return nil, sourceErr
			}
			if origin.Binding != req.Binding || revision.Set != origin.Set.GetName() {
				return nil, fmt.Errorf("new submission must retain the source binding and parameter set")
			}
		}
		preview, previewErr := s.Preview(ctx, req.Binding, req.Revision)
		if previewErr != nil {
			return nil, previewErr
		}
		if len(preview.Errors) > 0 {
			return nil, fmt.Errorf("revision is not valid for submission")
		}
		if req.ExecutionDigest != preview.ExecutionDigest {
			return nil, apierrors.NewConflict(schema.GroupResource{Group: p.Group, Resource: "workflowparameterbindings"}, req.Binding, fmt.Errorf("preview is stale; preview the current definitions before submitting"))
		}
		wf = preview.Workflow
		wf.Name = "parameters-" + p.Digest(req.IdempotencyKey)[:32]
		wf.Annotations = map[string]string{p.Group + "/revision": req.Revision, p.Group + "/execution-digest": preview.ExecutionDigest, p.Group + "/submission": name}
		if req.SourceExecution != nil {
			wf.Annotations[p.Group+"/source-workflow"] = req.SourceExecution.Workflow
			wf.Annotations[p.Group+"/source-workflow-uid"] = req.SourceExecution.UID
		}
		receipt = object("WorkflowSubmission", name, s.Namespace, map[string]any{"requestDigest": p.Digest(req), "revision": req.Revision, "binding": req.Binding, "executionDigest": preview.ExecutionDigest, "workflow": wf, "definitionVersions": preview.DefinitionVersions, "inputForms": preview.InputForms, "definitions": preview.Definitions, "sourceExecution": req.SourceExecution})
		if _, err = s.resource("workflowsubmissions").Create(ctx, receipt, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return h.submit(ctx, s, req)
			}
			return nil, err
		}
	}
	existing, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(s.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err == nil {
		if existing.Annotations[p.Group+"/submission"] != name {
			return nil, fmt.Errorf("workflow identity collision")
		}
		return existing, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	return h.Workflows.CreateWorkflow(ctx, &workflowapi.WorkflowCreateRequest{Namespace: s.Namespace, Workflow: wf})
}
