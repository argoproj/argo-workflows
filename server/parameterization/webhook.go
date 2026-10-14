package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	authorizationclient "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// AdmissionHandler is a read-only validating webhook, not a workflow controller.
// Dependencies are fetched using the webhook's narrowly scoped read credentials,
// after a SubjectAccessReview for the actual admission caller on EVERY read.
// This avoids granting the webhook permission to impersonate arbitrary users.
type AdmissionHandler struct {
	Namespace     string
	Config        *rest.Config
	Authorization authorizationclient.AuthorizationV1Interface
}

// Admission enters outside Argo Server's gRPC/HTTP middleware.
func admissionContext(ctx context.Context) context.Context {
	if logging.GetLoggerFromContextOrNil(ctx) == nil {
		ctx, _ = logging.NewSlogLogger(logging.Info, logging.JSON).InContext(ctx)
	}
	return ctx
}

func (h AdmissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := admissionContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var review admissionv1.AdmissionReview
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&review); err != nil || review.Request == nil {
		http.Error(w, "invalid admission review", http.StatusBadRequest)
		return
	}
	response := &admissionv1.AdmissionResponse{UID: review.Request.UID, Allowed: true}
	if err := h.validate(ctx, review.Request); err != nil {
		response.Allowed = false
		response.Result = &metav1.Status{Code: http.StatusUnprocessableEntity, Message: err.Error()}
	}
	_ = json.NewEncoder(w).Encode(admissionv1.AdmissionReview{TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"}, Response: response})
}

func (h AdmissionHandler) validate(ctx context.Context, req *admissionv1.AdmissionRequest) error {
	if req.Resource.Group != "argoproj.io" || req.Resource.Resource != "workflows" || (req.Operation != admissionv1.Create && req.Operation != admissionv1.Update) {
		return nil
	}
	var next wfv1.Workflow
	if err := json.Unmarshal(req.Object.Raw, &next); err != nil {
		return fmt.Errorf("invalid Workflow")
	}
	var old *wfv1.Workflow
	if req.Operation == admissionv1.Update {
		old = &wfv1.Workflow{}
		if err := json.Unmarshal(req.OldObject.Raw, old); err != nil {
			return fmt.Errorf("invalid previous Workflow")
		}
	}
	if !AdoptsParameterization(&next) && !AdoptsParameterization(old) {
		return nil
	}
	if req.Namespace == "" || req.Namespace != h.Namespace || next.Namespace != req.Namespace {
		return fmt.Errorf("workflow is outside the admission namespace")
	}
	if req.UserInfo.Username == "" {
		return fmt.Errorf("admission caller is required")
	}
	// No dependency reads for ordinary controller updates and native controls.
	if old != nil && next.Name == old.Name && next.UID == old.UID && sameParameterIdentity(&next, old) && p.Digest(contractSpec(&next)) == p.Digest(contractSpec(old)) && !resetsNativeExecution(&next, old) {
		return nil
	}
	if h.Config == nil || h.Authorization == nil {
		return fmt.Errorf("parameter admission dependencies are unavailable")
	}
	cfg := rest.CopyConfig(h.Config)
	cfg.Wrap(func(transport http.RoundTripper) http.RoundTripper {
		return &callerReadTransport{base: transport, authorization: h.Authorization, actor: req.UserInfo}
	})
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("cannot create admission client")
	}
	wc, err := wfclient.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("cannot create admission client")
	}
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("cannot create admission client")
	}
	ctx = context.WithValue(ctx, auth.DynamicKey, dc)
	ctx = context.WithValue(ctx, auth.WfKey, wc)
	ctx = context.WithValue(ctx, auth.KubeKey, kc)
	return (Store{Client: dc, Namespace: req.Namespace}).ValidateNativeWorkflow(ctx, &next, old)
}

type callerReadTransport struct {
	base          http.RoundTripper
	authorization authorizationclient.AuthorizationV1Interface
	actor         authenticationv1.UserInfo
}

func (t *callerReadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	attributes, err := readAttributes(req)
	if err != nil {
		return nil, err
	}
	extra := map[string]authorizationv1.ExtraValue{}
	for key, value := range t.actor.Extra {
		extra[key] = authorizationv1.ExtraValue(value)
	}
	check, err := t.authorization.SubjectAccessReviews().Create(req.Context(), &authorizationv1.SubjectAccessReview{Spec: authorizationv1.SubjectAccessReviewSpec{User: t.actor.Username, UID: t.actor.UID, Groups: t.actor.Groups, Extra: extra, ResourceAttributes: attributes}}, metav1.CreateOptions{})
	if err != nil || check == nil || !check.Status.Allowed || check.Status.Denied {
		return nil, fmt.Errorf("admission caller cannot read the required definition")
	}
	return t.base.RoundTrip(req)
}

func readAttributes(req *http.Request) (*authorizationv1.ResourceAttributes, error) {
	if req.Method != http.MethodGet || req.URL.Query().Get("watch") == "true" {
		return nil, fmt.Errorf("admission clients only support resource reads")
	}
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	a := &authorizationv1.ResourceAttributes{Verb: "list"}
	switch {
	case len(parts) >= 4 && parts[0] == "apis":
		a.Group, a.Version, parts = parts[1], parts[2], parts[3:]
	case len(parts) >= 3 && parts[0] == "api":
		a.Version, parts = parts[1], parts[2:]
	default:
		return nil, fmt.Errorf("unsupported admission read")
	}
	if len(parts) >= 3 && parts[0] == "namespaces" {
		a.Namespace, parts = parts[1], parts[2:]
	}
	if len(parts) < 1 || len(parts) > 2 {
		return nil, fmt.Errorf("unsupported admission resource read")
	}
	a.Resource = parts[0]
	if len(parts) == 2 {
		a.Name, a.Verb = parts[1], "get"
	}
	return a, nil
}
