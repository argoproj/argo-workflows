package parameterization

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

var editableDefinitions = map[string]string{
	"parametertemplates":        "ParameterTemplate",
	"workflowforms":             "WorkflowForm",
	"workflowparameterbindings": "WorkflowParameterBinding",
	"actiondefinitions":         "ActionDefinition",
	"actionresulttemplates":     "ActionResultTemplate",
	"workflowactionbindings":    "WorkflowActionBinding",
}

// DefinitionWrite changes spec only. Status, metadata and execution receipts
// cannot be supplied by this entry point. UID and resourceVersion are both
// required for updates, including validation of an unsaved editor document.
type DefinitionWrite struct {
	Spec            map[string]any `json:"spec"`
	UID             string         `json:"uid"`
	ResourceVersion string         `json:"resourceVersion"`
	ValidateOnly    bool           `json:"validateOnly"`
}

// writeDefinition uses Kubernetes strict field validation and server dry-run
// before persisting. This validates the installed CRD contract, not a concrete
// execution. Preview/submit still resolve and validate all references and values.
func (s Store) writeDefinition(ctx context.Context, resource, name string, req DefinitionWrite) (*unstructured.Unstructured, error) {
	kind, ok := editableDefinitions[resource]
	if !ok || len(validation.IsDNS1123Subdomain(name)) != 0 || req.Spec == nil {
		return nil, fmt.Errorf("a supported definition type, name and spec are required")
	}
	current, err := s.get(ctx, resource, name)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	conflict := func() (*unstructured.Unstructured, error) {
		return nil, apierrors.NewConflict(schema.GroupResource{Group: p.Group, Resource: resource}, name, fmt.Errorf("definition changed or was replaced; reload before saving"))
	}
	if current == nil {
		if req.UID != "" || req.ResourceVersion != "" {
			return conflict()
		}
		next := object(kind, name, s.Namespace, req.Spec)
		checked, validationErr := s.resource(resource).Create(ctx, next, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}, FieldValidation: metav1.FieldValidationStrict})
		if validationErr != nil || req.ValidateOnly {
			return checked, validationErr
		}
		// Create (not apply) refuses a competing publication of the same name.
		return s.resource(resource).Create(ctx, next, metav1.CreateOptions{FieldValidation: metav1.FieldValidationStrict})
	}
	if req.UID == "" || req.ResourceVersion == "" || req.UID != string(current.GetUID()) || req.ResourceVersion != current.GetResourceVersion() {
		return conflict()
	}
	if (isPublishedDefinition(current) || resource == "actiondefinitions" || resource == "workflowactionbindings") && p.Digest(current.Object["spec"]) != p.Digest(req.Spec) {
		return nil, fmt.Errorf("published definitions and action bindings are immutable; create a new named version")
	}
	next := current.DeepCopy()
	next.Object["spec"] = req.Spec
	checked, err := s.resource(resource).Update(ctx, next, metav1.UpdateOptions{DryRun: []string{metav1.DryRunAll}, FieldValidation: metav1.FieldValidationStrict})
	if err != nil || req.ValidateOnly {
		return checked, err
	}
	// Keep the original preconditions, rather than adopting a newer dry-run body.
	return s.resource(resource).Update(ctx, next, metav1.UpdateOptions{FieldValidation: metav1.FieldValidationStrict})
}

// Definition content identity is independent of status-only resourceVersion
// updates. UID still detects deletion/recreation; editors retain RV CAS.
func definitionContentVersion(o *unstructured.Unstructured) string {
	return string(o.GetUID()) + ":spec:" + p.Digest(o.Object["spec"])
}
