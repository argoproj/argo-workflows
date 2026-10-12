package parameterization

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func definitionFixture() (*unstructured.Unstructured, map[DefinitionKey]*unstructured.Unstructured) {
	objects := map[DefinitionKey]*unstructured.Unstructured{}
	model := object("ParameterTemplate", "model", "lab", map[string]any{"dialect": p.Dialect, "schema": map[string]any{"type": "object", "required": []string{"count"}, "properties": map[string]any{"count": map[string]any{"type": "integer"}}}})
	form := object("WorkflowForm", "form", "lab", map[string]any{"parameterTemplate": "model", "fields": []any{map[string]any{"path": "/count", "widget": "number"}}})
	native := object("WorkflowTemplate", "native", "lab", map[string]any{"entrypoint": "work", "arguments": map[string]any{"parameters": []any{map[string]any{"name": "count"}}}, "templates": []any{map[string]any{"name": "work", "container": map[string]any{"image": "alpine"}}}})
	native.SetAPIVersion("argoproj.io/v1alpha1")
	binding := object("WorkflowParameterBinding", "binding", "lab", Binding{WorkflowTemplate: "native", ParameterTemplate: "model", Form: "form", Mappings: []p.Mapping{{Path: "/count", Parameter: "count", Encoding: "decimal"}}})
	for resource, o := range map[string]*unstructured.Unstructured{"parametertemplates": model, "workflowforms": form, "workflowtemplates": native, "workflowparameterbindings": binding} {
		o.SetUID(types.UID("uid-" + resource))
		o.SetResourceVersion("1")
		o.SetGeneration(1)
		objects[DefinitionKey{resource, o.GetName()}] = o
	}
	return binding, objects
}

func TestDefinitionStatusClosureMissingRecoveryAndSchema(t *testing.T) {
	binding, objects := definitionFixture()
	lookup := func(k DefinitionKey) (*unstructured.Unstructured, error) {
		if o := objects[k]; o != nil {
			return o, nil
		}
		return nil, fmt.Errorf("missing")
	}
	check := func() parameterapi.DefinitionStatus { return EvaluateDefinition(binding, lookup) }
	status := check()
	require.Empty(t, status.Errors)
	require.Equal(t, metav1.ConditionTrue, status.Conditions[0].Status)
	require.Len(t, status.ResolvedReferences, 3)
	// No concrete input values are supplied; required count is not a definition error.
	model := objects[DefinitionKey{"parametertemplates", "model"}]
	delete(objects, DefinitionKey{"parametertemplates", "model"})
	require.Equal(t, metav1.ConditionFalse, check().Conditions[0].Status)
	model.SetUID("recreated")
	objects[DefinitionKey{"parametertemplates", "model"}] = model
	require.Empty(t, check().Errors)
	require.Equal(t, "recreated", check().ResolvedReferences[0].UID)
	require.NoError(t, unstructured.SetNestedField(model.Object, "unsupported", "spec", "schema", "format"))
	require.Equal(t, metav1.ConditionFalse, check().Conditions[0].Status)
	unstructured.RemoveNestedField(model.Object, "spec", "schema", "format")
	require.NoError(t, unstructured.SetNestedField(model.Object, "string", "spec", "schema", "properties", "count", "type"))
	require.Equal(t, metav1.ConditionFalse, check().Conditions[0].Status)
}

func TestDefinitionStatusContextAndStableObservations(t *testing.T) {
	form := object("WorkflowForm", "reusable", "lab", map[string]any{})
	status := EvaluateDefinition(form, func(DefinitionKey) (*unstructured.Unstructured, error) {
		t.Fatal("should not look up data")
		return nil, nil
	})
	require.Equal(t, metav1.ConditionUnknown, status.Conditions[0].Status)
	binding, objects := definitionFixture()
	lookup := func(k DefinitionKey) (*unstructured.Unstructured, error) { return objects[k], nil }
	before, err := stableDefinitionStatus(binding, EvaluateDefinition(binding, lookup))
	require.NoError(t, err)
	binding.Object["status"] = before
	for _, o := range objects {
		o.SetResourceVersion("status-only-new-rv")
	}
	after, err := stableDefinitionStatus(binding, EvaluateDefinition(binding, lookup))
	require.NoError(t, err)
	require.Equal(t, before, after)
	model := objects[DefinitionKey{"parametertemplates", "model"}]
	require.NoError(t, unstructured.SetNestedField(model.Object, "changed", "spec", "schema", "description"))
	after, err = stableDefinitionStatus(binding, EvaluateDefinition(binding, lookup))
	require.NoError(t, err)
	require.NotEqual(t, before, after)
}

func TestDefinitionControllerStatusOnlyReverseReferenceReconciliation(t *testing.T) {
	_, objects := definitionFixture()
	var items []runtime.Object
	lists := map[schema.GroupVersionResource]string{}
	for r, k := range editableDefinitions {
		lists[definitionGVR(r)] = k + "List"
	}
	lists[definitionGVR("workflowtemplates")] = "WorkflowTemplateList"
	for _, o := range objects {
		items = append(items, o.DeepCopy())
	}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), lists, items...)
	c, err := NewDefinitionController(client, "lab", false)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx, nil) }()
	get := func(resource, name string) *unstructured.Unstructured {
		o, e := client.Resource(definitionGVR(resource)).Namespace("lab").Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, e)
		return o
	}
	condition := func(resource, name string) string {
		s, _, _ := unstructured.NestedSlice(get(resource, name).Object, "status", "conditions")
		if len(s) == 0 {
			return ""
		}
		return s[0].(map[string]any)["status"].(string)
	}
	require.Eventually(t, func() bool { return condition("workflowparameterbindings", "binding") == "True" }, 4*time.Second, 10*time.Millisecond)
	m := get("parametertemplates", "model")
	m.SetGeneration(2)
	require.NoError(t, unstructured.SetNestedField(m.Object, "string", "spec", "schema", "properties", "count", "type"))
	_, err = client.Resource(definitionGVR("parametertemplates")).Namespace("lab").Update(ctx, m, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return condition("workflowparameterbindings", "binding") == "False" && condition("workflowforms", "form") == "False"
	}, 4*time.Second, 10*time.Millisecond)
	m = get("parametertemplates", "model")
	m.SetGeneration(3)
	require.NoError(t, unstructured.SetNestedField(m.Object, "integer", "spec", "schema", "properties", "count", "type"))
	_, err = client.Resource(definitionGVR("parametertemplates")).Namespace("lab").Update(ctx, m, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return condition("workflowparameterbindings", "binding") == "True" }, 4*time.Second, 10*time.Millisecond)
	// Reconciliation after a status notification is idempotent, not a write loop.
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("controller did not stop")
	}
	writes := 0
	for _, a := range client.Actions() {
		if a.GetVerb() == "update" {
			if a.GetSubresource() == "status" {
				require.Contains(t, editableDefinitions, a.GetResource().Resource)
				writes++
			} else {
				require.Equal(t, "parametertemplates", a.GetResource().Resource)
			}
		}
		require.NotEqual(t, "workflows", a.GetResource().Resource)
		require.NotEqual(t, "secrets", a.GetResource().Resource)
	}
	require.Less(t, writes, 18, "status-only writes must not cause a feedback loop")
}

func TestDefinitionStatusWriteDoesNotInvalidateExecutionPreview(t *testing.T) {
	ctx, s, _, revision := previewFixture(t)
	before, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	o, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	o.SetResourceVersion("status-update")
	o.Object["status"] = map[string]any{"observedGeneration": int64(1)}
	_, err = s.resource("workflowparameterbindings").UpdateStatus(ctx, o, metav1.UpdateOptions{})
	require.NoError(t, err)
	after, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Equal(t, before.ExecutionDigest, after.ExecutionDigest)
	o.SetUID("recreated")
	_, err = s.resource("workflowparameterbindings").Update(ctx, o, metav1.UpdateOptions{})
	require.NoError(t, err)
	after, err = s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.NotEqual(t, before.ExecutionDigest, after.ExecutionDigest)
}

func TestDefinitionStatusNestedNativeReferenceAndClusterScope(t *testing.T) {
	binding, objects := definitionFixture()
	native := objects[DefinitionKey{"workflowtemplates", "native"}]
	require.NoError(t, unstructured.SetNestedSlice(native.Object, []any{map[string]any{"name": "work", "dag": map[string]any{"tasks": []any{map[string]any{"name": "child", "templateRef": map[string]any{"name": "library", "template": "leaf", "clusterScope": true}}}}}}, "spec", "templates"))
	lookup := func(k DefinitionKey) (*unstructured.Unstructured, error) { return objects[k], nil }
	require.Equal(t, metav1.ConditionFalse, EvaluateDefinition(binding, lookup).Conditions[0].Status)
	library := object("ClusterWorkflowTemplate", "library", "", map[string]any{"templates": []any{map[string]any{"name": "leaf", "container": map[string]any{"image": "alpine"}}}})
	library.SetUID("library-uid")
	objects[DefinitionKey{"clusterworkflowtemplates", "library"}] = library
	status := EvaluateDefinition(binding, lookup)
	require.Empty(t, status.Errors)
	require.Len(t, status.ResolvedReferences, 4)
	require.Equal(t, []DefinitionKey{{"clusterworkflowtemplates", "library"}}, definitionDependencies(native))
	require.NoError(t, unstructured.SetNestedSlice(library.Object, []any{}, "spec", "templates"))
	require.Equal(t, metav1.ConditionFalse, EvaluateDefinition(binding, lookup).Conditions[0].Status)
}

func TestDefinitionStatusAllowsFrozenRuntimeOptionDeclarations(t *testing.T) {
	binding, objects := definitionFixture()
	var spec Binding
	require.NoError(t, decode(binding, &spec))
	spec.Inputs = []NodeInputBinding{{Template: "choose", ParameterTemplate: "model", Form: "form", Mappings: []p.Mapping{{Path: "/count", Parameter: "count", Encoding: "decimal"}}}}
	updated := object("WorkflowParameterBinding", "binding", "lab", spec)
	native := objects[DefinitionKey{"workflowtemplates", "native"}]
	templates, _, err := unstructured.NestedSlice(native.Object, "spec", "templates")
	require.NoError(t, err)
	templates = append(templates, map[string]any{"name": "choose", "suspend": map[string]any{}, "outputs": map[string]any{"parameters": []any{map[string]any{"name": "count", "valueFrom": map[string]any{"supplied": map[string]any{}}}}}})
	require.NoError(t, unstructured.SetNestedSlice(native.Object, templates, "spec", "templates"))
	form := objects[DefinitionKey{"workflowforms", "form"}]
	require.NoError(t, unstructured.SetNestedMap(form.Object, map[string]any{"/count": map[string]any{"kind": "configmap", "name": "catalog", "key": "choices"}}, "spec", "options"))
	status := EvaluateDefinition(updated, func(k DefinitionKey) (*unstructured.Unstructured, error) { return objects[k], nil })
	require.Empty(t, status.Errors)
	require.Equal(t, metav1.ConditionTrue, status.Conditions[0].Status)
}
