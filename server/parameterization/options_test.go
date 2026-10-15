package parameterization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestAbsentOptionalOptionsDoNotRequireProviderOrDependencies(t *testing.T) {
	ctx, s, _, _ := previewFixture(t)
	model := p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object", "required": []string{"count"}, "properties": map[string]any{"count": map[string]any{"type": "integer"}, "environment": map[string]any{"type": "string"}, "region": map[string]any{"type": "string"}}}}
	_, err := s.resource("parametertemplates").Create(ctx, object("ParameterTemplate", "optional", "lab", model), metav1.CreateOptions{})
	require.NoError(t, err)
	form := map[string]any{"parameterTemplate": "optional", "options": map[string]any{"/region": OptionSource{Kind: "configmap", Name: "missing-provider", Key: "regions", DependsOn: []string{"/environment"}}}}
	_, err = s.resource("workflowforms").Create(ctx, object("WorkflowForm", "optional", "lab", form), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowparameterbindings").Create(ctx, object("WorkflowParameterBinding", "optional", "lab", Binding{WorkflowTemplate: "native", ParameterTemplate: "optional", Form: "optional", Mappings: []parameterapi.ParameterMapping{{Path: "/count", Parameter: "count", Encoding: "decimal"}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	saved, err := s.Save(ctx, "optional", SaveRequest{Template: "optional", Values: map[string]any{"count": 2}})
	require.NoError(t, err)
	revision, _, _ := unstructured.NestedString(saved.Object, "spec", "revision")
	preview, err := s.Preview(ctx, "optional", revision)
	require.NoError(t, err)
	require.Empty(t, preview.Errors)
	require.NotContains(t, preview.DefinitionVersions, "options//region")
	contract := &FrozenInputForm{Model: model, Form: form}
	require.ErrorContains(t, validateForm(parameterapi.WorkflowFormSpec{Options: map[string]OptionSource{"/region": {DependsOn: []string{"/undeclared"}}}}, model, "optional"), "invalid form option dependency")
	require.ErrorContains(t, validateForm(parameterapi.WorkflowFormSpec{Options: map[string]OptionSource{"/undeclared": {}}}, model, "optional"), "invalid form option field")
	versions, err := s.validateInputOptions(ctx, contract, map[string]any{"count": 2})
	require.NoError(t, err)
	require.Empty(t, versions)
	// Absence is distinct from false, zero, an empty string and explicit null.
	for _, value := range []any{false, 0, "", nil} {
		_, err = s.validateInputOptions(ctx, contract, map[string]any{"count": 2, "region": value})
		require.ErrorContains(t, err, "dependency /environment is absent")
	}
	_, err = s.validateInputOptions(ctx, contract, map[string]any{"count": 2, "environment": "test", "region": "east"})
	require.Error(t, err, "a supplied field still requires provider authorization")
}

func TestOptionsAreCallerScopedVersionedAndRechecked(t *testing.T) {
	ctx, s, _, revision := previewFixture(t)
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "catalog", Namespace: "lab", UID: "options", ResourceVersion: "1"}, Data: map[string]string{"choices": `[{"value":2,"label":"Two","when":{"/enabled":false}},{"value":3,"label":"Three","disabled":true}]`}}
	_, err := auth.GetKubeClient(ctx).CoreV1().ConfigMaps("lab").Create(ctx, cm, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowforms").Create(ctx, object("WorkflowForm", "form", "lab", map[string]any{"options": map[string]any{"/count": OptionSource{Kind: "configmap", Name: "catalog", Key: "choices", DependsOn: []string{"/enabled"}}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	b, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	require.NoError(t, unstructured.SetNestedField(b.Object, "form", "spec", "form"))
	_, err = s.resource("workflowparameterbindings").Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	result, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Contains(t, result.DefinitionVersions, "options//count")
	cm.Data["choices"] = `[]`
	_, err = auth.GetKubeClient(ctx).CoreV1().ConfigMaps("lab").Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err)
	result, err = s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Equal(t, "OptionUnavailable", result.Errors[0].Code)
	_, err = s.Options(context.Background(), OptionsRequest{Binding: "unknown"}, nil)
	require.Error(t, err)
}
