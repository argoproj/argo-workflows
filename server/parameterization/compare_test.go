package parameterization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestCompareSeparatesModelSourcesAndIdentityRules(t *testing.T) {
	s := testStore()
	ctx := context.Background()
	a := Revision{Set: "set", Model: p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object"}, IdentityRules: []p.IdentityRule{{Path: "/items", KeyPath: "/id"}}, SensitivePaths: []string{"/credential"}}, Values: map[string]any{"items": []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}}, "enabled": false}, Sources: map[string]string{"/enabled": "default"}}
	b := a
	b.Model.IdentityRules = nil
	b.Model.Schema = map[string]any{"type": "object", "description": "new model"}
	b.Values = map[string]any{"items": []any{map[string]any{"id": "b"}, map[string]any{"id": "a"}}, "enabled": false}
	b.Sources = map[string]string{"/enabled": "user"}
	for name, v := range map[string]Revision{"first": a, "second": b} {
		_, err := s.resource("parameterrevisions").Create(ctx, object("ParameterRevision", name, "lab", v), metav1.CreateOptions{})
		require.NoError(t, err)
	}
	raw, err := s.compare(ctx, "first", "second")
	require.NoError(t, err)
	r := raw.(map[string]any)
	require.Equal(t, true, r["modelChanged"])
	require.Equal(t, true, r["identityRulesChanged"])
	require.NotEmpty(t, r["modelChanges"])
	require.NotEmpty(t, r["sourceChanges"])
	changes := r["changes"].([]p.Change)
	require.Len(t, changes, 1)
	require.Equal(t, "replace", changes[0].Op)
	require.Equal(t, "/items", changes[0].Path)
	require.Contains(t, r["sensitivePaths"], "/credential")
	_, err = s.compare(ctx, "missing", "second")
	require.Error(t, err)
}

func TestCompareRejectsImportedPlaintextAcrossSensitivityChange(t *testing.T) {
	ctx := context.Background()
	for _, sensitiveSide := range []string{"first", "second"} {
		t.Run(sensitiveSide, func(t *testing.T) {
			s := testStore()
			for _, name := range []string{"first", "second"} {
				v := Revision{Set: "set", Model: p.Model{Dialect: p.Dialect, Schema: map[string]any{"type": "object"}}, Values: map[string]any{"credential": "must-never-appear-in-diff"}}
				if name == sensitiveSide {
					v.Model.SensitivePaths = []string{"/credential"}
					v.Values.(map[string]any)["credential"] = map[string]any{"secretRef": map[string]any{"name": "example", "key": "value"}}
				}
				_, err := s.resource("parameterrevisions").Create(ctx, object("ParameterRevision", name, "lab", v), metav1.CreateOptions{})
				require.NoError(t, err)
			}
			result, err := s.compare(ctx, "first", "second")
			require.Error(t, err)
			require.Nil(t, result)
			require.NotContains(t, err.Error(), "must-never-appear-in-diff")
		})
	}
}
