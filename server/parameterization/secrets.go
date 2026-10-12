package parameterization

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// SecretWriter is an adapter seam for a namespace-aware encrypted secret store.
// Implementations must authorize the caller, encrypt at rest, and return only a
// pinned native Secret reference. External stores may materialize a versioned
// Secret, but must never expose plaintext to ordinary parameter APIs.
type SecretWriter interface {
	Write(context.Context, SecretWriteContext, []byte) (map[string]any, error)
}

type SecretWriteContext struct {
	Namespace, Actor, Template, Path, RequestID string
}

// InputSecretContext retains the encrypted-input API name for compatibility.
type InputSecretContext = InputContext

type SecretWriteRequest struct {
	Input     *InputSecretContext `json:"input,omitempty"`
	Template  string              `json:"template"`
	Path      string              `json:"path"`
	RequestID string              `json:"requestId"`
	Value     string              `json:"value"`
}

// EncryptedKubernetesSecrets delegates encryption and key rotation to the
// Kubernetes API server. It must be explicitly enabled only on installations
// with encryption at rest for Secrets and metadata-only audit policy.
// A Secret's base64 wire representation alone is NOT encryption.
type EncryptedKubernetesSecrets struct{}

func validatePinnedSecrets(ctx context.Context, namespace string, versions map[string]string) error {
	for ref, version := range versions {
		if !strings.HasPrefix(ref, "secret/") {
			continue
		}
		secret, err := auth.GetKubeClient(ctx).CoreV1().Secrets(namespace).Get(ctx, strings.TrimPrefix(ref, "secret/"), metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("pinned Secret is revoked, unavailable or forbidden")
		}
		if err := p.ValidateExternalSecretLease(secret, time.Now()); err != nil {
			return err
		}
		// Immutable Secret data cannot change under the same UID. Metadata-only
		// resourceVersion writes and storage-key re-encryption do not change it.
		uid, _, valid := strings.Cut(version, ":")
		if !valid || secret.Immutable == nil || !*secret.Immutable || string(secret.UID) != uid {
			return fmt.Errorf("pinned Secret identity changed; retry cannot substitute another version")
		}
	}
	return nil
}

func (EncryptedKubernetesSecrets) Write(ctx context.Context, purpose SecretWriteContext, value []byte) (map[string]any, error) {
	identity := p.Digest([]string{purpose.Namespace, purpose.Actor, purpose.Template, purpose.Path, purpose.RequestID})
	name := "parameter-secret-" + identity[:32]
	client := auth.GetKubeClient(ctx).CoreV1().Secrets(purpose.Namespace)
	immutable := true
	obj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: purpose.Namespace, Annotations: map[string]string{p.Group + "/purpose": identity}},
		Immutable:  &immutable, Type: corev1.SecretTypeOpaque, Data: map[string][]byte{"value": value},
	}
	existing, err := client.Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		existing, err = client.Create(ctx, obj, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			existing, err = client.Get(ctx, name, metav1.GetOptions{})
		}
	}
	if err != nil {
		// Provider errors may include request bodies. Never propagate those.
		if apierrors.IsForbidden(err) {
			return nil, apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, name, fmt.Errorf("secret write is not authorized"))
		}
		return nil, fmt.Errorf("encrypted secret backend unavailable; retry the same requestId")
	}
	if existing.Immutable == nil || !*existing.Immutable || existing.Annotations[p.Group+"/purpose"] != identity || len(existing.Data) != 1 || subtle.ConstantTimeCompare(existing.Data["value"], value) != 1 {
		return nil, apierrors.NewConflict(schema.GroupResource{Resource: "secrets"}, name, fmt.Errorf("requestId is already used; replacement requires a new requestId"))
	}
	return map[string]any{"secretRef": map[string]any{"name": name, "key": "value"}}, nil
}

func secretLogicalSchema(model p.Model, path string) map[string]any {
	if logical := model.SecretSchemas[path]; logical != nil {
		return logical
	}
	keys := make([]string, 0, len(model.SecretSchemas))
	for key := range model.SecretSchemas {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := strings.Count(keys[i], "/*"), strings.Count(keys[j], "/*")
		if a == b {
			return keys[i] < keys[j]
		}
		return a < b
	})
	for _, key := range keys {
		if p.MatchPath(key, path) {
			return model.SecretSchemas[key]
		}
	}
	return nil
}

func validateSecretValue(model p.Model, path string, value string) error {
	declared := false
	for _, candidate := range model.SensitivePaths {
		declared = declared || p.MatchPath(candidate, path)
	}
	logical := secretLogicalSchema(model, path)
	if !declared || logical == nil {
		return fmt.Errorf("secret input requires a declared sensitive path and logical schema")
	}
	_, _, errors := p.Resolve(p.Model{Dialect: model.Dialect, Schema: logical}, value)
	if len(errors) > 0 {
		return fmt.Errorf("secret value does not satisfy its declared schema")
	}
	return nil
}

func (h Handler) writeSecret(ctx context.Context, s Store, req SecretWriteRequest) (any, error) {
	if h.SecretWriter == nil {
		return nil, fmt.Errorf("encrypted secret input is not configured")
	}
	if strings.Contains(req.Path, "*") {
		return nil, fmt.Errorf("secret write requires a concrete array item path")
	}
	if len(req.RequestID) < 16 || len(req.RequestID) > 128 || len(req.Value) > 65536 {
		return nil, fmt.Errorf("requestId must have 16-128 characters; secret input maximum is 64 KiB")
	}
	var model p.Model
	purpose := req.Template
	if req.Input != nil {
		if req.Template != "" {
			return nil, fmt.Errorf("input context and template are mutually exclusive")
		}
		contract, err := h.inputContract(ctx, s, *req.Input)
		if err != nil {
			return nil, err
		}
		model = contract.Model
		purpose = "workflow:" + req.Input.UID + "/" + req.Input.NodeID + "/" + req.Input.ContractDigest
	} else {
		template, err := s.get(ctx, "parametertemplates", req.Template)
		if err != nil {
			return nil, err
		}
		if err = decode(template, &model); err != nil {
			return nil, err
		}
	}
	if len(p.ValidateModelSchema(model)) > 0 {
		return nil, fmt.Errorf("invalid secret parameter model")
	}
	if err := validateSecretValue(model, req.Path, req.Value); err != nil {
		return nil, err
	}
	return h.SecretWriter.Write(ctx, SecretWriteContext{s.Namespace, s.Actor, purpose, req.Path, req.RequestID}, []byte(req.Value))
}

func (h Handler) secretTransportAllowed(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	// Explicit local development only. Never trust X-Forwarded-Proto from a caller.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	return h.AllowLoopbackSecretInput && err == nil && ip != nil && ip.IsLoopback()
}

func (s Store) authorizeMappedSecrets(ctx context.Context, model p.Model, values any, mappings []p.Mapping, versions map[string]string) error {
	// Reference-only mappings may drive native withParam loops. Pin and authorize
	// every consumed Secret before exposing references to the Workflow.
	for _, mapping := range mappings {
		if mapping.Encoding != "secret-reference-json" {
			continue
		}
		for _, pattern := range model.SensitivePaths {
			for _, path := range p.ExpandPaths(values, pattern) {
				if path != mapping.Path && !strings.HasPrefix(path, mapping.Path+"/") {
					continue
				}
				raw, exists := p.Pointer(values, path)
				if !exists || raw == nil {
					continue
				}
				ref := raw.(map[string]any)["secretRef"].(map[string]any)
				name, key := ref["name"].(string), ref["key"].(string)
				credential, err := auth.GetKubeClient(ctx).CoreV1().Secrets(s.Namespace).Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("secret reference is unavailable or forbidden")
				}
				if err := p.ValidateExternalSecretLease(credential, time.Now()); err != nil {
					return err
				}
				if credential.Immutable == nil || !*credential.Immutable {
					return fmt.Errorf("reproducible submission requires an immutable versioned Secret")
				}
				plaintext, ok := credential.Data[key]
				if !ok {
					return fmt.Errorf("secret key is absent")
				}
				if secretLogicalSchema(model, path) != nil {
					if err := validateSecretValue(model, path, string(plaintext)); err != nil {
						return err
					}
				}
				versions["secret/"+name] = string(credential.UID) + ":" + credential.ResourceVersion
			}
		}
	}
	return nil
}
