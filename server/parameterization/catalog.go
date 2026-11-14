package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type ResourceType struct {
	Resource    string `json:"resource"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// The catalog is deliberately limited to this extension. It never acts as an
// arbitrary Kubernetes proxy or lists Secret contents. All reads use the caller's
// existing client and namespace, with normal Kubernetes RBAC and pagination.
var resourceTypes = []ResourceType{
	{"parametertemplates", "ParameterTemplate", "Typed input schema, defaults, derivation and sensitive field contracts"},
	{"workflowforms", "WorkflowForm", "Reusable labels, widgets and dynamic option sources"},
	{"workflowparameterbindings", "WorkflowParameterBinding", "WorkflowTemplate, parameter model, form and input mapping"},
	{"parametersets", "ParameterSet", "Editable saved inputs and current revision reference"},
	{"parameterrevisions", "ParameterRevision", "Immutable input snapshots, sources and version history"},
	{"actiondefinitions", "ActionDefinition", "Action input/output contracts and native implementation"},
	{"actionresulttemplates", "ActionResultTemplate", "Reusable public result sources, schemas and views, bound per action call"},
	{"workflowactionbindings", "WorkflowActionBinding", "Workflow call sites bound to reusable actions"},
	{"workflowsubmissions", "WorkflowSubmission", "Frozen submission contract and native Workflow identity"},
	{"parameterretries", "ParameterRetry", "Recorded native node retry attempts"},
	{"parameternodeinputs", "ParameterNodeInput", "Typed inputs supplied to native suspend nodes"},
	{"resourceoperations", "ResourceOperation", "Optional external operation contract; requires a separately installed operation controller"},
}

func (s Store) catalog(ctx context.Context, resource, name, cursor string) (any, error) {
	if resource == "" {
		return resourceTypes, nil
	}
	allowed := false
	for _, typ := range resourceTypes {
		if typ.Resource == resource {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("unsupported parameter resource type")
	}
	if name != "" {
		if len(validation.IsDNS1123Subdomain(name)) != 0 {
			return nil, fmt.Errorf("invalid resource name")
		}
		return s.get(ctx, resource, name)
	}
	return s.resource(resource).List(ctx, metav1.ListOptions{Limit: 50, Continue: cursor})
}

// catalogMetadata uses the caller's existing transport and Kubernetes pagination.
// Exclude annotations/managedFields too: kubectl's last-applied annotation may
// contain the complete document. The exact resource is fetched only on demand.
func (s Store) catalogMetadata(ctx context.Context, resource, cursor string) (*unstructured.UnstructuredList, error) {
	allowed := false
	for _, typ := range resourceTypes {
		if typ.Resource == resource {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("unsupported parameter resource type")
	}
	var list *unstructured.UnstructuredList
	if s.MetadataClient != nil {
		raw, err := s.MetadataClient.Get().AbsPath("/apis/", p.Group, p.Version, "namespaces", s.Namespace, resource).
			SetHeader("Accept", "application/json;as=PartialObjectMetadataList;g=meta.k8s.io;v=v1").
			Param("limit", strconv.Itoa(50)).Param("continue", cursor).DoRaw(ctx)
		if err != nil {
			return nil, err
		}
		list = &unstructured.UnstructuredList{}
		if err = json.Unmarshal(raw, list); err != nil {
			return nil, err
		}
	} else {
		// Direct Store clients can omit the metadata transport; RBAC still applies.
		var err error
		list, err = s.resource(resource).List(ctx, metav1.ListOptions{Limit: 50, Continue: cursor})
		if err != nil {
			return nil, err
		}
	}
	for i := range list.Items {
		list.Items[i] = minimalMetadata(list.Items[i])
	}
	list.SetAPIVersion("meta.k8s.io/v1")
	list.SetKind("PartialObjectMetadataList")
	return list, nil
}

func minimalMetadata(obj unstructured.Unstructured) unstructured.Unstructured {
	out := unstructured.Unstructured{Object: map[string]any{"apiVersion": "meta.k8s.io/v1", "kind": "PartialObjectMetadata"}}
	out.SetName(obj.GetName())
	out.SetNamespace(obj.GetNamespace())
	out.SetUID(obj.GetUID())
	out.SetResourceVersion(obj.GetResourceVersion())
	out.SetCreationTimestamp(obj.GetCreationTimestamp())
	return out
}

// historySummaries projects the page after committed-chain verification.
// Restoring a revision must still read its immutable full record.
func historySummaries(list *unstructured.UnstructuredList) *unstructured.UnstructuredList {
	out := &unstructured.UnstructuredList{}
	out.SetAPIVersion(p.Group + "/" + p.Version)
	out.SetKind("ParameterRevisionSummaryList")
	out.SetContinue(list.GetContinue())
	for _, obj := range list.Items {
		summary := minimalMetadata(obj)
		summary.SetAPIVersion(p.Group + "/" + p.Version)
		summary.SetKind("ParameterRevisionSummary")
		spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
		fields := map[string]any{}
		for _, key := range []string{"set", "parent", "template", "digest", "actor", "savedAt", "message", "restoredFrom"} {
			if value, exists := spec[key]; exists {
				fields[key] = value
			}
		}
		errors, _ := spec["errors"].([]any)
		fields["validationErrorCount"] = int64(len(errors))
		summary.Object["spec"] = fields
		out.Items = append(out.Items, summary)
	}
	if out.Items == nil {
		out.Items = []unstructured.Unstructured{}
	}
	return out
}
