package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestSensitiveArraySavePreviewDiffAndRevocation(t *testing.T) {
	ctx, s, _, _ := previewFixture(t)
	m := p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"items": map[string]any{"type": "array", "items": map[string]any{"type": "object"}}}}, SensitivePaths: []string{"/items/*/credential"}, IdentityRules: []p.IdentityRule{{Path: "/items", KeyPath: "/id"}}, SecretSchemas: map[string]map[string]any{"/items/*/credential": {"type": "string", "minLength": 8}}}
	_, err := s.resource("parametertemplates").Update(ctx, object("ParameterTemplate", "typed", "lab", m), metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowparameterbindings").Update(ctx, object("WorkflowParameterBinding", "standard", "lab", Binding{WorkflowTemplate: "native", ParameterTemplate: "typed", Mappings: []p.Mapping{{Path: "/items", Parameter: "count", Encoding: "secret-reference-json"}}}), metav1.UpdateOptions{})
	require.NoError(t, err)
	h := Handler{SecretWriter: EncryptedKubernetesSecrets{}}
	ref, err := h.writeSecret(ctx, s, SecretWriteRequest{Template: "typed", Path: "/items/0/credential", RequestID: "one-long-request-id", Value: "synthetic-secret"})
	require.NoError(t, err)
	values := map[string]any{"items": []any{map[string]any{"id": "one", "credential": ref}, map[string]any{"id": "two", "credential": ref}}}
	first, err := s.Save(ctx, "array", SaveRequest{Template: "typed", Values: values})
	require.NoError(t, err)
	r1, _, _ := unstructured.NestedString(first.Object, "spec", "revision")
	preview, err := s.Preview(ctx, "standard", r1)
	require.NoError(t, err)
	require.Len(t, preview.DefinitionVersions, 4)
	items := values["items"].([]any)
	items[0], items[1] = items[1], items[0]
	second, err := s.Save(ctx, "array", SaveRequest{Template: "typed", Values: values, ResourceVersion: first.GetResourceVersion()})
	require.NoError(t, err)
	r2, _, _ := unstructured.NestedString(second.Object, "spec", "revision")
	diff, err := s.compare(ctx, r1, r2)
	require.NoError(t, err)
	require.Contains(t, diff.(map[string]any)["changes"], p.Change{Op: "move", Path: "/items/0", From: "/items/1", Identity: "two"})
	items[0].(map[string]any)["credential"] = "must-not-store"
	_, err = s.Save(ctx, "bad", SaveRequest{Template: "typed", Values: values})
	require.ErrorContains(t, err, "SecretReferenceRequired")
	items[0].(map[string]any)["credential"] = ref
	items[0].(map[string]any)["id"] = "one"
	_, err = s.Save(ctx, "bad", SaveRequest{Template: "typed", Values: values})
	require.ErrorContains(t, err, "InvalidItemIdentity")
	refObj := ref.(map[string]any)["secretRef"].(map[string]any)
	name := refObj["name"].(string)
	k := auth.GetKubeClient(ctx).CoreV1().Secrets("lab")
	require.NoError(t, k.Delete(ctx, name, metav1.DeleteOptions{}))
	_, err = s.Preview(ctx, "standard", r1)
	require.ErrorContains(t, err, "unavailable")
	_, err = k.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name}, Data: map[string][]byte{"value": []byte("synthetic-secret")}}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.Preview(ctx, "standard", r1)
	require.ErrorContains(t, err, "immutable")
}
