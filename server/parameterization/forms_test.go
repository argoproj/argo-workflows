package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestDeclaredFormGraphAndWidgets(t *testing.T) {
	model := p.Model{Schema: map[string]any{"type": "object", "properties": map[string]any{"enabled": map[string]any{"type": "boolean"}, "zones": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}, "active": map[string]any{"type": "boolean"}}}}}}}
	form := parameterapi.WorkflowFormSpec{Layout: []parameterapi.FormLayoutNode{{ID: "main", Kind: "tab"}, {ID: "zones", Kind: "tree", Parent: "main"}}, Fields: []parameterapi.FormField{{Path: "/enabled", Widget: "checkbox", Group: "main"}, {Path: "/zones", Widget: "table", Group: "zones"}, {Path: "/zones/*/name", EnabledWhen: &parameterapi.FieldPredicate{Path: "/zones/*/active", Operator: "Equals"}}}}
	require.NoError(t, validateForm(form, model, "model"))
	for _, field := range []parameterapi.FormField{{Path: "/missing"}, {Path: "/enabled", Widget: "number"}, {Path: "/enabled", Widget: "secret"}, {Path: "/zones/0/name", Group: "zones"}, {Path: "/zones", Group: "missing"}, {Path: "/enabled", VisibleWhen: &parameterapi.FieldPredicate{Path: "/zones/*/active", Operator: "Exists"}}} {
		bad := form
		bad.Fields = []parameterapi.FormField{field}
		require.Error(t, validateForm(bad, model, "model"))
	}
	form.Layout[0].Parent = "zones"
	require.ErrorContains(t, validateForm(form, model, "model"), "cycle")
}
func TestFormOptionAliasesAndArrayExpansion(t *testing.T) {
	shared := OptionSource{Kind: "configmap", Name: "choices", Key: "values"}
	specific := OptionSource{Kind: "configmap", Name: "specific", Key: "values"}
	form := parameterapi.WorkflowFormSpec{Options: map[string]OptionSource{"regions": shared, "/zones/1/name": specific}, Fields: []parameterapi.FormField{{Path: "/zones/*/name", OptionSource: "regions"}}}
	sources, err := formOptions(form)
	require.NoError(t, err)
	expanded := expandFormOptions(sources, map[string]any{"zones": []any{map[string]any{"name": "first"}, map[string]any{"name": "second"}}})
	require.Len(t, expanded, 2)
	require.Equal(t, shared, expanded["/zones/0/name"])
	require.Equal(t, specific, expanded["/zones/1/name"])
	_, ok := selectedOption(sources, "/zones/01/name")
	require.False(t, ok)
	_, ok = selectedOption(sources, "/zones/-1/name")
	require.False(t, ok)
}
func TestPreviewRefusesInvalidFormBeforeSubmission(t *testing.T) {
	ctx, s, _, revision := previewFixture(t)
	_, err := s.resource("workflowforms").Create(ctx, object("WorkflowForm", "display", "lab", map[string]any{"fields": []any{map[string]any{"path": "/count", "group": "absent"}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	b, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(b.Object, "display", "spec", "form"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.Preview(ctx, "standard", revision)
	require.ErrorContains(t, err, "layout group")
}
