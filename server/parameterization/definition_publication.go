package parameterization

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

const definitionPublicationPrefix = "parameters.workflows.example.io/definition-"
const definitionPublishedDigest = definitionPublicationPrefix + "digest"
const definitionFamily = definitionPublicationPrefix + "family"

// DefinitionPublishRequest publishes a new immutable object of the original definition kind.
// It versions that definition's spec, not a release of its transitive references.
// Preview and submit still freeze the full execution closure with caller RBAC.
type DefinitionPublishRequest struct {
	Source       string         `json:"source"`
	UID          string         `json:"uid"`
	SourceSpec   map[string]any `json:"sourceSpec"`
	Name         string         `json:"name"`
	Message      string         `json:"message,omitempty"`
	ValidateOnly bool           `json:"validateOnly,omitempty"`
}

func isPublishedDefinition(o *unstructured.Unstructured) bool {
	return o.GetAnnotations()[definitionPublishedDigest] != ""
}

func (s Store) publishDefinition(ctx context.Context, resource string, req DefinitionPublishRequest) (*unstructured.Unstructured, error) {
	expectedVersion := req.UID + ":spec:" + p.Digest(req.SourceSpec)
	kind, ok := editableDefinitions[resource]
	if !ok || req.Source == req.Name || req.UID == "" || req.SourceSpec == nil || len(req.Message) > 2048 ||
		len(validation.IsDNS1123Subdomain(req.Source)) != 0 || len(validation.IsDNS1123Subdomain(req.Name)) != 0 {
		return nil, fmt.Errorf("definition source, UID, content version and a distinct publication name are required")
	}
	conflict := func() error {
		return apierrors.NewConflict(schema.GroupResource{Group: p.Group, Resource: resource}, req.Name, fmt.Errorf("definition source or publication changed; reload before publishing"))
	}
	// The named publication is the durable receipt. A lost response never requires
	// another version, and recovery remains possible after deleting the draft.
	existing, err := s.get(ctx, resource, req.Name)
	if err == nil {
		a := existing.GetAnnotations()
		if a[definitionPublicationPrefix+"source"] == req.Source && a[definitionPublicationPrefix+"source-uid"] == req.UID &&
			a[definitionPublicationPrefix+"source-version"] == expectedVersion && a[definitionPublicationPrefix+"message"] == req.Message &&
			a[definitionPublishedDigest] == p.Digest(existing.Object["spec"]) {
			return existing, nil
		}
		return nil, conflict()
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	source, err := s.get(ctx, resource, req.Source)
	if err != nil {
		return nil, err
	}
	if string(source.GetUID()) != req.UID || definitionContentVersion(source) != expectedVersion || source.GetDeletionTimestamp() != nil {
		return nil, conflict()
	}
	status := EvaluateDefinition(source, func(key DefinitionKey) (*unstructured.Unstructured, error) {
		if key.Resource == "workflowtemplates" || key.Resource == "clusterworkflowtemplates" {
			native := s.Client.Resource(schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: key.Resource})
			if key.Resource == "clusterworkflowtemplates" {
				return native.Get(ctx, key.Name, metav1.GetOptions{})
			}
			return native.Namespace(s.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		}
		return s.get(ctx, key.Resource, key.Name)
	})
	if len(status.Errors) > 0 {
		return nil, fmt.Errorf("definition cannot be published: %s", status.Errors[0].Message)
	}
	next := object(kind, req.Name, s.Namespace, source.DeepCopy().Object["spec"])
	family := source.GetLabels()[definitionFamily]
	if !isPublishedDefinition(source) || family == "" {
		family = string(source.GetUID())
	}
	next.SetLabels(map[string]string{definitionFamily: family})
	next.SetAnnotations(map[string]string{
		definitionPublishedDigest:                      p.Digest(source.Object["spec"]),
		definitionPublicationPrefix + "source":         req.Source,
		definitionPublicationPrefix + "source-uid":     req.UID,
		definitionPublicationPrefix + "source-version": expectedVersion,
		definitionPublicationPrefix + "published-at":   time.Now().UTC().Format(time.RFC3339),
		definitionPublicationPrefix + "publisher":      s.Actor,
		definitionPublicationPrefix + "message":        req.Message,
	})
	checked, err := s.resource(resource).Create(ctx, next, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}, FieldValidation: metav1.FieldValidationStrict})
	if err != nil || req.ValidateOnly {
		return checked, err
	}
	latest, err := s.get(ctx, resource, req.Source)
	if err != nil {
		return nil, err
	}
	if definitionContentVersion(latest) != expectedVersion || latest.GetDeletionTimestamp() != nil {
		return nil, conflict()
	}
	next, err = s.resource(resource).Create(ctx, next, metav1.CreateOptions{FieldValidation: metav1.FieldValidationStrict})
	if apierrors.IsAlreadyExists(err) {
		// Bounded read-only recovery: a competing writer cannot trigger another create.
		winner, readErr := s.get(ctx, resource, req.Name)
		if readErr != nil {
			return nil, readErr
		}
		a := winner.GetAnnotations()
		if a[definitionPublicationPrefix+"source"] == req.Source && a[definitionPublicationPrefix+"source-uid"] == req.UID &&
			a[definitionPublicationPrefix+"source-version"] == expectedVersion && a[definitionPublicationPrefix+"message"] == req.Message &&
			a[definitionPublishedDigest] == p.Digest(winner.Object["spec"]) {
			return winner, nil
		}
		return nil, conflict()
	}
	return next, err
}

// Use an indexed family selector and Kubernetes continuation, not a full catalog
// scan. The family UID remains queryable after the source draft is deleted.
func (s Store) definitionVersions(ctx context.Context, resource, family, cursor string) (*unstructured.UnstructuredList, error) {
	if editableDefinitions[resource] == "" || family == "" || len(validation.IsValidLabelValue(family)) != 0 {
		return nil, fmt.Errorf("supported definition resource and source family UID required")
	}
	list, err := s.resource(resource).List(ctx, metav1.ListOptions{LabelSelector: definitionFamily + "=" + family, Limit: 25, Continue: cursor})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		o := list.Items[i]
		if !isPublishedDefinition(&o) || o.GetAnnotations()[definitionPublishedDigest] != p.Digest(o.Object["spec"]) {
			return nil, fmt.Errorf("published definition integrity check failed")
		}
		out := minimalMetadata(o)
		out.SetLabels(map[string]string{definitionFamily: family})
		annotations := map[string]string{}
		for _, key := range []string{"digest", "source", "source-uid", "source-version", "published-at", "publisher", "message"} {
			annotations[definitionPublicationPrefix+key] = o.GetAnnotations()[definitionPublicationPrefix+key]
		}
		out.SetAnnotations(annotations)
		list.Items[i] = out
	}
	list.SetAPIVersion("meta.k8s.io/v1")
	list.SetKind("PartialObjectMetadataList")
	return list, nil
}

func (s Store) definitionDiff(ctx context.Context, resource string, req DiffRequest) (map[string]any, error) {
	if editableDefinitions[resource] == "" || len(validation.IsDNS1123Subdomain(req.Left)) != 0 || len(validation.IsDNS1123Subdomain(req.Right)) != 0 {
		return nil, fmt.Errorf("supported definition resource and two names required")
	}
	left, err := s.get(ctx, resource, req.Left)
	if err != nil {
		return nil, err
	}
	right, err := s.get(ctx, resource, req.Right)
	if err != nil {
		return nil, err
	}
	comparison := map[string]any{
		"left": req.Left, "right": req.Right,
		"leftVersion": definitionContentVersion(left), "rightVersion": definitionContentVersion(right),
		"changes":      p.Diff(left.Object["spec"], right.Object["spec"], nil),
		"modelChanges": []p.Change{}, "sourceChanges": []p.Change{},
	}
	return diffPage(comparison, req)
}
