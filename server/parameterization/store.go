// Package parameterization implements an opt-in Argo Server extension using caller-scoped Kubernetes clients.
package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type Store struct {
	Client          dynamic.Interface
	MetadataClient  rest.Interface
	Namespace       string
	Actor           string
	OptionProviders map[string]OptionProvider
	HistoryPager    *HistoryPager
}

func (s Store) resource(plural string) dynamic.ResourceInterface {
	return s.Client.Resource(schema.GroupVersionResource{Group: p.Group, Version: p.Version, Resource: plural}).Namespace(s.Namespace)
}
func (s Store) get(ctx context.Context, plural, name string) (*unstructured.Unstructured, error) {
	return s.resource(plural).Get(ctx, name, metav1.GetOptions{})
}
func object(kind, name, namespace string, spec any) *unstructured.Unstructured {
	b, _ := json.Marshal(spec)
	var value map[string]any
	_ = json.Unmarshal(b, &value)
	return &unstructured.Unstructured{Object: map[string]any{"apiVersion": p.Group + "/" + p.Version, "kind": kind, "metadata": map[string]any{"name": name, "namespace": namespace}, "spec": value}}
}
func decode(obj *unstructured.Unstructured, out any) error {
	b, err := json.Marshal(obj.Object["spec"])
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

type SaveRequest struct {
	Template        string `json:"template"`
	Values          any    `json:"values"`
	ResourceVersion string `json:"resourceVersion"`
	Message         string `json:"message"`
	RestoredFrom    string `json:"restoredFrom,omitempty"`
}
type Revision struct {
	Set          string            `json:"set"`
	Parent       string            `json:"parent,omitempty"`
	Template     string            `json:"template"`
	Model        p.Model           `json:"model"`
	Values       any               `json:"values"`
	Sources      map[string]string `json:"sources"`
	Errors       []p.FieldError    `json:"errors"`
	Digest       string            `json:"digest"`
	Actor        string            `json:"actor"`
	SavedAt      string            `json:"savedAt"`
	Message      string            `json:"message,omitempty"`
	RestoredFrom string            `json:"restoredFrom,omitempty"`
}

// Save creates an immutable revision before atomically advancing the editable set.
// On a failed CAS an unreferenced revision may remain; the set pointer never refers to a missing revision.
func (s Store) Save(ctx context.Context, name string, req SaveRequest) (*unstructured.Unstructured, error) {
	current, err := s.get(ctx, "parametersets", name)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	if current == nil && req.ResourceVersion != "" || current != nil && current.GetResourceVersion() != req.ResourceVersion {
		return nil, apierrors.NewConflict(schema.GroupResource{Group: p.Group, Resource: "parametersets"}, name, fmt.Errorf("draft changed; reload and compare"))
	}
	template, err := s.get(ctx, "parametertemplates", req.Template)
	if err != nil {
		return nil, err
	}
	var model p.Model
	if err = decode(template, &model); err != nil {
		return nil, err
	}
	values, sources, errors := p.Resolve(model, req.Values)
	// Draft validation failures can be saved, but plaintext sensitive values cannot.
	for _, e := range errors {
		if (len(model.SensitivePaths) > 0 && (e.Code == "InvalidIdentityRule" || e.Code == "InvalidItemIdentity")) || e.Code == "InputComplexity" || e.Code == "SecretReferenceRequired" || e.Code == "UnsupportedKeyword" || e.Code == "UnsupportedDialect" || e.Code == "InvalidSchema" || e.Code == "UnsupportedReference" {
			return nil, fmt.Errorf("%s at %s", e.Code, e.Path)
		}
	}
	if req.RestoredFrom != "" {
		prior, restoreErr := s.get(ctx, "parameterrevisions", req.RestoredFrom)
		if restoreErr != nil {
			return nil, restoreErr
		}
		var restored Revision
		if restoreErr = decode(prior, &restored); restoreErr != nil {
			return nil, restoreErr
		}
		if restored.Set != name {
			return nil, fmt.Errorf("restore revision belongs to another parameter set")
		}
	}
	digest := p.Digest(map[string]any{"template": req.Template, "model": model, "values": values})
	parent := ""
	if current != nil {
		parent, _, _ = unstructured.NestedString(current.Object, "spec", "revision")
		oldDigest, _, _ := unstructured.NestedString(current.Object, "spec", "digest")
		if digest == oldDigest {
			return current, nil
		}
	}
	revName := "revision-" + p.Digest([]string{name, parent, digest})[:40]
	rev := Revision{Set: name, Parent: parent, Template: req.Template, Model: model, Values: values, Sources: sources, Errors: errors, Digest: digest, Actor: s.Actor, SavedAt: time.Now().UTC().Format(time.RFC3339), Message: req.Message, RestoredFrom: req.RestoredFrom}
	record := object("ParameterRevision", revName, s.Namespace, rev)
	record.SetLabels(map[string]string{p.Group + "/set": name})
	if _, err = s.resource("parameterrevisions").Create(ctx, record, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		existing, err := s.get(ctx, "parameterrevisions", revName)
		if err != nil {
			return nil, err
		}
		var prior Revision
		if err = decode(existing, &prior); err != nil {
			return nil, err
		}
		if prior.Set != name || prior.Parent != parent || prior.Digest != digest || p.Digest(prior.Values) != p.Digest(values) || p.Digest(prior.Model) != p.Digest(model) {
			return nil, fmt.Errorf("revision identity collision")
		}
	}
	next := object("ParameterSet", name, s.Namespace, map[string]any{"template": req.Template, "values": values, "revision": revName, "digest": digest})
	if current == nil {
		return s.resource("parametersets").Create(ctx, next, metav1.CreateOptions{})
	}
	next.SetLabels(current.GetLabels())
	next.SetAnnotations(current.GetAnnotations())
	next.SetResourceVersion(current.GetResourceVersion())
	next.SetUID(current.GetUID())
	return s.resource("parametersets").Update(ctx, next, metav1.UpdateOptions{})
}

func (s Store) History(ctx context.Context, name string, limit int64, token string) (*unstructured.UnstructuredList, error) {
	pager := s.HistoryPager
	if pager == nil {
		var err error
		pager, err = processHistoryPager()
		if err != nil {
			return nil, err
		}
	}
	return pager.page(ctx, s, name, limit, token)
}
